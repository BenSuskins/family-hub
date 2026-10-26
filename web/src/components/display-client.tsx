"use client";

import Image from "next/image";
import { Fragment, FormEvent, MouseEvent, useCallback, useEffect, useRef, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { BookOpen, CalendarDays, CarFront, Check, CookingPot, ListChecks, Monitor, ShoppingBag, Sun, TrainFront, Trash2, Utensils, X, type LucideIcon } from "lucide-react";
import { KioskScratchpad } from "@/components/kiosk-scratchpad";
import { getApiBaseUrl } from "@/lib/api";
import {
  dayKeyAt,
  displayGreeting,
  displayStoragePrefix,
  householdDocumentTitle,
  isDisplaySnapshot,
  mealWindowAt,
  rankDisplayCards,
  relevantBinDate,
  relevantDepartures,
  usefulCalendarEvents,
  weatherLabel,
  type DisplayDeparture,
  type DisplayEvent,
  type MealType,
} from "@/lib/display-logic";

type DisplayChore = { ID: string; Name: string; Status: string; DueDate?: string | null; AssignedToUserID?: string | null };
type DisplayMeal = { Date: string; MealType: MealType; Name: string; RecipeID?: string | null };
type DisplayRecipe = { ID: string; Title: string; Steps?: string[]; Instructions?: string; Ingredients?: { name: string; items: string[] }[] };
type Weather = { status: string; fetchedAt?: string; location?: string; advice?: string; current?: { temperature: number; code: number; summary?: string }; high?: number; low?: number; hours?: { time: string; temperature: number; code: number; rainProbability: number }[]; days?: { date: string; high: number; low: number; code: number; rainProbability?: number }[] };
type BinSchedule = { collections?: { date: string; type: string; name: string }[]; enabled?: boolean };
type CommuteBoard = { origin: { code: string }; destination: { code: string }; fetchedAt: string; trains: { serviceId: string; status: string; expectedDeparture?: string | null; scheduledDeparture?: string | null; platform?: string | null; operator?: string }[] };
type CommuteResult = { status: string; timezone?: string; boards: CommuteBoard[] };
type DisplayPerson = { id: string; name: string };
type DisplaySettings = { id: string; name: string; enabledSections: string[]; commutePreferences?: unknown };
type Snapshot = {
  schemaVersion: 2;
  displayId: string;
  householdName?: string;
  events: DisplayEvent[];
  chores: DisplayChore[];
  meals: DisplayMeal[];
  weather?: Weather;
  departures?: { status: string; items: DisplayDeparture[] };
  bins?: BinSchedule;
  commute?: CommuteResult;
  people: DisplayPerson[];
  enabledSections: string[];
  timezone: string;
  resourceUpdatedAt: Record<string, string>;
  availability: Record<string, string>;
};
type Pairing = { challengeId: string; pairingCode: string; expiresAt: string };
type MealEditTarget = { date: string; mealType: MealType };
type EventDetails = { title: string; startTime: string; endTime?: string | null; location?: string; leaveAt?: string; drivingSeconds?: number };

const apiBase = getApiBaseUrl();
const localMocksEnabled = process.env.NODE_ENV === "development" && !process.env.NEXT_PUBLIC_FAMILY_HUB_API_URL;
const storagePrefix = displayStoragePrefix(apiBase);
const displayTokenKey = `${storagePrefix}.token`;
const snapshotKey = (id: string) => `${storagePrefix}.snapshot.${encodeURIComponent(id)}`;
const overviewSections = ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"] as const;
const sectionIcons: Record<string, LucideIcon> = {
  calendar: CalendarDays, meals: Utensils, chores: ListChecks, weather: Sun,
  departures: CarFront, trains: TrainFront, bins: Trash2, shopping: ShoppingBag,
};
const launcherApps: { name: string; icon: LucideIcon }[] = [
  { name: "Shopping", icon: ShoppingBag }, { name: "Notes", icon: BookOpen },
  { name: "Calendar", icon: CalendarDays }, { name: "Recipes", icon: CookingPot },
  { name: "Meal plan", icon: CookingPot }, { name: "Inventory", icon: Check },
];
class DisplayAccessRevoked extends Error {}

function SectionIcon({ name, size = 21 }: { name: string; size?: number }) {
  const Icon = sectionIcons[name] ?? Monitor;
  return <Icon size={size} strokeWidth={1.7} aria-hidden="true" className="display-icon" />;
}

function readStorage(key: string) {
  try { return window.localStorage.getItem(key); } catch { return null; }
}
function writeStorage(key: string, value: string) {
  try { window.localStorage.setItem(key, value); return true; } catch { return false; }
}
function removeStorage(key: string) {
  try { window.localStorage.removeItem(key); } catch { /* Browser storage can be disabled. */ }
}

function weekStartAt(date: Date, timezone: string) {
  const weekStart = new Date(`${dayKeyAt(date, timezone)}T12:00:00Z`);
  weekStart.setUTCDate(weekStart.getUTCDate() - ((weekStart.getUTCDay() + 6) % 7));
  return weekStart.toISOString().slice(0, 10);
}

function hourAt(value: string, timezone: string) {
  const hour = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", hourCycle: "h23", timeZone: timezone })
    .formatToParts(new Date(value)).find(part => part.type === "hour")?.value;
  return Number(hour ?? 0);
}

async function displayRequest<T>(path: string, token: string): Promise<T> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 8_000);
  try {
    const response = await fetch(apiBase + path, { headers: { Authorization: `Bearer ${token}` }, signal: controller.signal });
    if (response.status === 401 || response.status === 403) throw new DisplayAccessRevoked();
    if (!response.ok) throw new Error(`${path} returned ${response.status}`);
    return await response.json() as T;
  } finally { window.clearTimeout(timeout); }
}

function PairDisplay({ onClaimed }: { onClaimed: (token: string) => void }) {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [error, setError] = useState("");
  const [renewing, setRenewing] = useState(false);
  const [clock, setClock] = useState<Date | null>(null);
  const create = useCallback(async () => {
    setRenewing(true);
    setError("");
    try {
      const response = await fetch(apiBase + "/api/display-pairing/challenges", { method: "POST" });
      if (!response.ok) throw new Error("We could not start pairing. Check the connection and try again.");
      setPairing(await response.json() as Pairing);
    } catch {
      setError("We couldn’t connect to Family Hub. Check the connection, then try again.");
    } finally { setRenewing(false); }
  }, []);
  useEffect(() => {
    setClock(new Date());
    void create();
    const timer = window.setInterval(() => setClock(new Date()), 30_000);
    return () => window.clearInterval(timer);
  }, [create]);
  useEffect(() => {
    if (!pairing) return;
    let stopped = false;
    let requestInFlight = false;
    async function checkClaim() {
      if (requestInFlight || stopped) return;
      requestInFlight = true;
      try {
        const response = await fetch(`${apiBase}/api/display-pairing/challenges/${pairing!.challengeId}/claim`, {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ pairingCode: pairing!.pairingCode }),
        });
        if (response.ok) {
          const body = await response.json() as { token: string };
          if (!stopped && Date.parse(pairing!.expiresAt) > Date.now()) onClaimed(body.token);
        } else if (response.status === 409 && !stopped && Date.parse(pairing!.expiresAt) > Date.now()) {
          setError("");
        } else if (!stopped && response.status !== 409) {
          setError("The display could not check its pairing. It will keep trying.");
        }
      } catch {
        if (!stopped) setError("The display could not check its pairing. It will keep trying.");
      } finally { requestInFlight = false; }
    }
    void checkClaim();
    const timer = window.setInterval(() => {
      if (Date.parse(pairing.expiresAt) <= Date.now()) {
        stopped = true;
        setPairing(null);
        setError("This pairing code expired. Create a new one to continue.");
      } else void checkClaim();
    }, 5_000);
    return () => { stopped = true; window.clearInterval(timer); };
  }, [pairing, onClaimed]);
  const qrValue = pairing ? JSON.stringify({ apiBaseUrl: apiBase, challengeId: pairing.challengeId, pairingCode: pairing.pairingCode }) : "";
  const pairingHour = clock ? Number(new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/London", hour: "2-digit", hourCycle: "h23" }).format(clock)) : 12;
  const isDark = pairingHour >= 20 || pairingHour < 6;
  return <main className={`pair-screen${isDark ? " is-dark" : ""}`}>
    <section className="pair-card" aria-label="Family Hub display pairing">
      <Image className="brand-mark-image" src="/family-hub-mark.png" alt="Family Hub" width={80} height={80} priority />
      <h1>{error.includes("expired") ? "Let’s try that again." : "One screen for everyone."}</h1>
      <p className="pair-instructions">On your iPhone or iPad, open Family Hub → Settings → Wall displays → Scan a display code. Then approve this screen.</p>
      {pairing ? <div className="qr-wrap"><QRCodeSVG value={qrValue} size={200} level="M" includeMargin /></div> : !error && <div className="qr-empty" role="status">{renewing ? "Preparing a pairing code…" : "Pairing code unavailable"}</div>}
      {error && <p className="error-message" role="alert">{error}</p>}
      {pairing && <small>Code expires in five minutes and can be used once.</small>}
      {(!pairing || error) && <button className="action-button" disabled={renewing} onClick={() => void create()}>{renewing ? "Preparing…" : error ? "Try again" : "Create pairing code"}</button>}
      <p className="pair-note">Shared screen. No personal sign-in is stored here.</p>
    </section>
  </main>;
}

export function DisplayClient() {
  const [mockReady, setMockReady] = useState(!localMocksEnabled);
  const [token, setToken] = useState("");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [householdName, setHouseholdName] = useState("Family");
  const snapshotRef = useRef<Snapshot | null>(null);
  const [clock, setClock] = useState<Date | null>(null);
  const clockRef = useRef<Date | null>(null);
  const [selectedApp, setSelectedApp] = useState("");
  const [selectedEvent, setSelectedEvent] = useState<EventDetails | null>(null);
  const [mealEditTarget, setMealEditTarget] = useState<MealEditTarget | null>(null);
  const [selectedChore, setSelectedChore] = useState<DisplayChore | null>(null);
  const [loading, setLoading] = useState(true);
  const [failures, setFailures] = useState<string[]>([]);
  const dialogRef = useRef<HTMLElement>(null);
  const refreshInFlight = useRef(false);
  const openerRef = useRef<HTMLElement | null>(null);
  const scratchpadSaveRef = useRef<(() => Promise<void>) | null>(null);
  const closeInFlightRef = useRef(false);

  const registerScratchpadSave = useCallback((save: (() => Promise<void>) | null) => {
    scratchpadSaveRef.current = save;
  }, []);
  const close = useCallback(async () => {
    if (closeInFlightRef.current || dialogRef.current?.querySelector("canvas[data-drawing='true']")) return;
    closeInFlightRef.current = true;
    try {
      if (selectedApp === "Notes") await scratchpadSaveRef.current?.();
    } finally {
      closeInFlightRef.current = false;
      setSelectedApp(""); setSelectedEvent(null); setSelectedChore(null); setMealEditTarget(null);
    }
  }, [selectedApp]);
  clockRef.current = clock;

  const accept = useCallback((value: string) => {
    if (!writeStorage(displayTokenKey, value)) setFailures(["This browser cannot save the pairing. Keep this tab open or enable site storage."]);
    setToken(value);
  }, []);

  useEffect(() => {
    let stopped = false;
    let timer: number | undefined;
    async function initialise() {
      if (localMocksEnabled) {
        const { startLocalMocks } = await import("@/mocks/browser");
        await startLocalMocks();
      }
      if (stopped) return;
      const storedToken = readStorage(displayTokenKey) ?? "";
      let initialClock = new Date();
      if (localMocksEnabled) {
        try {
          const response = await fetch("/api/local-mocks/clock");
          if (response.ok) initialClock = new Date((await response.json() as { now: string }).now);
        } catch { /* Use the system clock if the local mock clock is unavailable. */ }
      }
      clockRef.current = initialClock;
      setClock(initialClock);
      setToken(storedToken);
      setMockReady(true);
      if (localMocksEnabled) {
        let lastMockClock = initialClock;
        let lastReadAt = Date.now();
        timer = window.setInterval(() => {
          void fetch("/api/local-mocks/clock").then(async response => {
            if (!response.ok) return;
            const nextClock = new Date((await response.json() as { now: string }).now);
            const expected = lastMockClock.getTime() + (Date.now() - lastReadAt);
            const jumped = Math.abs(nextClock.getTime() - expected) > 30_000;
            lastMockClock = nextClock;
            lastReadAt = Date.now();
            clockRef.current = nextClock;
            setClock(nextClock);
            if (jumped) window.dispatchEvent(new Event("family-hub:mock-clock-changed"));
          }).catch(() => undefined);
        }, 5_000);
      } else {
        timer = window.setInterval(() => { const nextClock = new Date(); clockRef.current = nextClock; setClock(nextClock); }, 30_000);
      }
    }
    void initialise();
    return () => { stopped = true; if (timer) window.clearInterval(timer); };
  }, []);

  const refresh = useCallback(async () => {
    if (!token || refreshInFlight.current) return;
    refreshInFlight.current = true;
    let successfulResources: string[] = [];
    try {
      const displaySettings = await displayRequest<DisplaySettings>("/api/display/settings", token);
      if (!displaySettings.id) throw new Error("The server returned incomplete display settings.");
      const storageKey = snapshotKey(displaySettings.id);
      let previous = snapshotRef.current?.displayId === displaySettings.id ? snapshotRef.current : null;
      if (!previous) {
        const stored = readStorage(storageKey);
        if (stored) {
          try {
            const decoded: unknown = JSON.parse(stored);
            if (isDisplaySnapshot(decoded) && decoded.displayId === displaySettings.id) previous = decoded as Snapshot;
            else removeStorage(storageKey);
          } catch { removeStorage(storageKey); }
        }
      }
      if (previous) {
        snapshotRef.current = previous;
        setSnapshot(previous);
        if (previous.householdName) setHouseholdName(previous.householdName);
      }
      const homeResult = await Promise.allSettled([displayRequest<{ timezone: string; familyName?: string }>("/api/display/home-info", token)]);
      if (homeResult[0].status === "rejected" && homeResult[0].reason instanceof DisplayAccessRevoked) throw homeResult[0].reason;
      const timezone = homeResult[0].status === "fulfilled" ? homeResult[0].value.timezone : previous?.timezone ?? "Europe/London";
      const nextHouseholdName = homeResult[0].status === "fulfilled" ? homeResult[0].value.familyName?.trim() || "Family" : previous?.householdName || "Family";
      setHouseholdName(nextHouseholdName);
      if (homeResult[0].status === "fulfilled") successfulResources.push("home");
      const now = localMocksEnabled ? clockRef.current ?? new Date() : new Date();
      const mealWindow = mealWindowAt(now, timezone);
      const sections = new Set(displaySettings.enabledSections ?? []);
      const fetches: [string, Promise<unknown>][] = [];
      if (sections.has("calendar") || sections.has("meals") || sections.has("chores")) {
        fetches.push(["calendar", displayRequest(`/api/display/calendar?view=week&date=${mealWindow.date}`, token)]);
      }
      if (sections.has("weather")) fetches.push(["weather", displayRequest("/api/display/weather", token)]);
      if (sections.has("departures")) fetches.push(["departures", displayRequest("/api/display/departures", token)]);
      if (sections.has("bins")) fetches.push(["bins", displayRequest("/api/display/bin-collections", token)]);
      if (sections.has("chores")) fetches.push(["people", displayRequest("/api/display/people", token)]);
      if (sections.has("trains")) {
        fetches.push(["commute", displayRequest("/api/display/commute", token)]);
        fetches.push(["trains", displayRequest("/api/display/trains", token)]);
      }
      const results = await Promise.allSettled(fetches.map(([, promise]) => promise));
      if (results.some(result => result.status === "rejected" && result.reason instanceof DisplayAccessRevoked)) throw new DisplayAccessRevoked();
      const values = new Map<string, unknown>();
      const nextFailures: string[] = homeResult[0].status === "rejected" ? ["Home settings"] : [];
      results.forEach((result, index) => {
        const key = fetches[index][0];
        if (result.status === "fulfilled") { values.set(key, result.value); successfulResources.push(key); }
        else nextFailures.push(resourceName(key));
      });
      const calendar = values.get("calendar") as { events?: DisplayEvent[]; chores?: DisplayChore[]; meals?: DisplayMeal[]; availability?: Record<string, string> } | undefined;
      const calendarAvailability = calendar?.availability ?? {};
      const resourceAvailability = { ...previous?.availability };
      for (const [key, result] of results.entries()) {
        const resource = fetches[key][0];
        resourceAvailability[resource] = result.status === "fulfilled" ? "available" : previous?.availability[resource] ? "stale" : "unavailable";
      }
      if (values.has("calendar")) {
        resourceAvailability.events = calendarAvailability.events === "unavailable"
          ? previous?.resourceUpdatedAt.events ? "stale" : "unavailable"
          : "available";
        resourceAvailability.meals = calendarAvailability.meals === "unavailable"
          ? previous?.resourceUpdatedAt.meals ? "stale" : "unavailable"
          : "available";
        resourceAvailability.chores = "available";
      } else if (sections.has("calendar") || sections.has("meals") || sections.has("chores")) {
        resourceAvailability.events = previous?.resourceUpdatedAt.events ? "stale" : "unavailable";
        resourceAvailability.meals = previous?.resourceUpdatedAt.meals ? "stale" : "unavailable";
        resourceAvailability.chores = previous?.resourceUpdatedAt.chores ? "stale" : "unavailable";
      }
      const next: Snapshot = {
        schemaVersion: 2,
        displayId: displaySettings.id,
        householdName: nextHouseholdName,
        events: calendar && calendarAvailability.events !== "unavailable" ? calendar.events ?? [] : previous?.events ?? [],
        chores: calendar?.chores ?? previous?.chores ?? [],
        meals: calendar && calendarAvailability.meals !== "unavailable" ? calendar.meals ?? [] : previous?.meals ?? [],
        weather: values.has("weather") ? values.get("weather") as Weather : previous?.weather,
        departures: values.has("departures") ? values.get("departures") as Snapshot["departures"] : previous?.departures,
        bins: values.has("bins") ? values.get("bins") as BinSchedule : previous?.bins,
        commute: values.has("trains") ? values.get("trains") as CommuteResult : previous?.commute,
        people: values.has("people") ? values.get("people") as DisplayPerson[] : previous?.people ?? [],
        enabledSections: displaySettings.enabledSections ?? [],
        timezone,
        resourceUpdatedAt: { ...previous?.resourceUpdatedAt },
        availability: resourceAvailability,
      };
      for (const resource of successfulResources) next.resourceUpdatedAt[resource] = now.toISOString();
      if (values.has("calendar")) {
        if (resourceAvailability.events === "available") next.resourceUpdatedAt.events = now.toISOString();
        if (resourceAvailability.meals === "available") next.resourceUpdatedAt.meals = now.toISOString();
        next.resourceUpdatedAt.chores = now.toISOString();
      }
      snapshotRef.current = next;
      setSnapshot(next);
      setFailures(nextFailures);
      if ((previous || successfulResources.length > 1) && !writeStorage(storageKey, JSON.stringify(next)) && !nextFailures.length) setFailures(["This browser cannot save display data. It will refresh again when reopened."]);
    } catch (reason) {
      if (reason instanceof DisplayAccessRevoked) {
        removeStorage(displayTokenKey);
        if (snapshotRef.current?.displayId) removeStorage(snapshotKey(snapshotRef.current.displayId));
        snapshotRef.current = null;
        setSnapshot(null);
        setToken("");
        setFailures([]);
      } else {
        setFailures(["Display settings could not be reached. The display will retry."]);
      }
    } finally {
      setLoading(false);
      refreshInFlight.current = false;
    }
  }, [token]);

  useEffect(() => {
    if (!localMocksEnabled || !token) return;
    const refreshForMockClock = () => void refresh();
    window.addEventListener("family-hub:mock-clock-changed", refreshForMockClock);
    return () => window.removeEventListener("family-hub:mock-clock-changed", refreshForMockClock);
  }, [token, refresh]);

  useEffect(() => {
    if (!token) return;
    void refresh();
    const timer = window.setInterval(() => void refresh(), 60_000);
    const reconnect = () => void refresh();
    window.addEventListener("online", reconnect);
    return () => { window.clearInterval(timer); window.removeEventListener("online", reconnect); };
  }, [token, refresh]);

  useEffect(() => {
    document.title = householdDocumentTitle(householdName);
  }, [householdName]);

  useEffect(() => {
    if (!selectedApp) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    openerRef.current = previousFocus;
    dialogRef.current?.focus();
    let timer = window.setTimeout(closeForIdle, 5 * 60_000);
    const resetIdle = () => { window.clearTimeout(timer); timer = window.setTimeout(closeForIdle, 5 * 60_000); };
    function closeForIdle() {
      const drawing = dialogRef.current?.querySelector("canvas[data-drawing='true']");
      if (drawing) { timer = window.setTimeout(closeForIdle, 30_000); return; }
      void close();
    }
    function keydown(event: KeyboardEvent) {
      resetIdle();
      if (event.key === "Escape") { event.preventDefault(); void close(); return; }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const controls = Array.from(dialogRef.current.querySelectorAll<HTMLElement>("button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex='0']"));
      if (!controls.length) return;
      if (event.shiftKey && document.activeElement === controls[0]) { event.preventDefault(); controls.at(-1)?.focus(); }
      else if (!event.shiftKey && document.activeElement === controls.at(-1)) { event.preventDefault(); controls[0].focus(); }
    }
    const dialog = dialogRef.current;
    dialog?.addEventListener("pointerdown", resetIdle);
    dialog?.addEventListener("input", resetIdle);
    window.addEventListener("keydown", keydown);
    return () => {
      window.clearTimeout(timer);
      dialog?.removeEventListener("pointerdown", resetIdle);
      dialog?.removeEventListener("input", resetIdle);
      window.removeEventListener("keydown", keydown);
      previousFocus?.focus();
    };
  }, [selectedApp, close]);

  const open = (app: string, event?: MouseEvent<HTMLElement>) => {
    openerRef.current = event?.currentTarget ?? document.activeElement as HTMLElement;
    setSelectedApp(app);
  };
  const openEvent = (details: EventDetails, event?: MouseEvent<HTMLElement>) => {
    setSelectedEvent(details);
    open("Family event", event);
  };
  if (!mockReady) return <main className="pair-screen"><p role="status">Starting the local display…</p></main>;
  if (!token) return <PairDisplay onClaimed={accept} />;
  if (!clock) return <main className="wall-display-shell" aria-busy="true" />;

  const current = snapshot;
  const timezone = current?.timezone ?? "Europe/London";
  const dateKey = dayKeyAt(clock, timezone);
  const mealWindow = mealWindowAt(clock, timezone);
  const sections = new Set(current?.enabledSections ?? overviewSections);
  const upcomingEvents = usefulCalendarEvents(current?.events ?? [], clock, timezone);
  const todayEvents = upcomingEvents.filter(event => dayKeyAt(new Date(event.StartTime), timezone) === dateKey);
  const nextEventDay = upcomingEvents[0] ? dayKeyAt(new Date(upcomingEvents[0].StartTime), timezone) : dateKey;
  const events = todayEvents.length ? todayEvents : upcomingEvents.filter(event => dayKeyAt(new Date(event.StartTime), timezone) === nextEventDay);
  const chores = (current?.chores ?? []).filter(item => item.Status === "pending" || item.Status === "overdue").sort((first, second) => (first.DueDate ?? "9999").localeCompare(second.DueDate ?? "9999"));
  const meals = (current?.meals ?? []).filter(item => item.Date === mealWindow.date);
  const departureItems = relevantDepartures(current?.departures?.items ?? [], clock);
  const collections = (current?.bins?.collections ?? []).filter(item => relevantBinDate(item.date, dateKey));
  const weather = current?.weather;
  const weatherReady = weather?.status === "available" || weather?.status === "ok";
  const commute = current?.commute;
  const trains = commute?.status === "available" || commute?.status === "partial" ? commute.boards.flatMap(board => board.trains.map(train => ({ board, train }))) : [];
  const departureMinutes = departureItems[0] ? Math.round((Date.parse(departureItems[0].leaveAt) - clock.getTime()) / 60_000) : Infinity;
  const localHour = Number(new Intl.DateTimeFormat("en-GB", { timeZone: timezone, hour: "2-digit", hourCycle: "h23" }).format(clock));
  const availableCards = new Set([
    ...(sections.has("calendar") && events.length ? ["calendar"] : []),
    ...(sections.has("meals") && meals.length ? ["meals"] : []),
    ...(sections.has("chores") && chores.length ? ["chores"] : []),
    ...(sections.has("departures") && departureItems.length ? ["departures"] : []),
    ...(sections.has("bins") && collections.length ? ["bins"] : []),
    ...(sections.has("trains") && trains.length ? ["trains"] : []),
  ]);
  const rankedCards = rankDisplayCards({
    hour: localHour,
    departureSoon: departureMinutes <= 45,
    dinnerNext: mealWindow.label === "Today" && mealWindow.nextMeal === "dinner",
    hasCalendarEvents: events.length > 0,
    hasOpenChores: chores.length > 0,
    binsRelevant: collections.length > 0,
    binsTomorrow: collections.some(item => item.date !== dateKey),
    trainsAvailable: trains.length > 0,
    weatherAvailable: weatherReady && Boolean(weather?.current),
  }).filter(key => availableCards.has(key)).slice(0, 5);
  const cardOrder = Object.fromEntries(rankedCards.map((key, index) => [key, index]));
  const isCardVisible = (key: string) => sections.has(key) && cardOrder[key] !== undefined;
  const cardStyle = (key: string) => ({ order: cardOrder[key] });
  const cardLayoutClass = (key: string) => cardOrder[key] === 0 ? "primary-first" : cardOrder[key] === 1 ? "primary-second" : "secondary-card";
  const isDark = localHour >= 20 || localHour < 6;
  const formattedDate = clock.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone });
  const formattedTime = clock.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });

  return <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}>
    <section className={`wall-display${isDark ? " is-dark" : ""}`} aria-label="Family Hub wall display">
      <header className="display-header"><div className="display-brand"><div><p>{formattedDate}</p><h1>{displayGreeting(localHour, departureMinutes <= 45)}</h1></div></div><div className="display-header-right">{sections.has("weather") && weatherReady && weather?.current && <button className="weather-summary" onClick={event => open("Weather", event)} aria-label="Open weather forecast"><Sun size={41} strokeWidth={1.7} aria-hidden="true"/><span><strong>{Math.round(weather.current.temperature)}°</strong><small>{weather.current.summary ?? weatherLabel(weather.current.code)} · H {Math.round(weather.high ?? weather.current.temperature)}° L {Math.round(weather.low ?? weather.current.temperature)}°</small></span></button>}<time>{formattedTime}</time></div></header>
      {failures.length > 0 && <div className="display-status" role="status">{failures.join(" and ")} {current ? "could not be updated. Showing saved information where available." : "are unavailable. The display will keep trying."}</div>}
      {loading && !current && <p className="display-loading" role="status">Loading household information…</p>}
      <div className="display-grid" data-card-count={Object.keys(cardOrder).length}>
        {isCardVisible("departures") && departureItems.length > 0 && <section style={cardStyle("departures")} className={`display-card departure-card ${cardLayoutClass("departures")}`}><div className="display-card-heading"><span><SectionIcon name="departures"/>Time to head out</span><small className="leave-pill">{departureMinutes > 0 ? `Leave in ${departureMinutes} min` : "Time to leave"}</small></div><div className="departure-time">{new Date(departureItems[0].leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</div><h2>{departureItems[0].title}</h2><p>{[departureItems[0].subtitle, departureItems[0].destination, departureItems[0].drivingSeconds ? `${Math.round(departureItems[0].drivingSeconds / 60)} min drive` : ""].filter(Boolean).join(" · ")}</p><div className="departure-footer"><small>Starts at {new Date(departureItems[0].startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</small><button className="display-card-cta" onClick={event => { const item = current?.events.find(calendarEvent => calendarEvent.ID === departureItems[0].eventID); openEvent({ title: departureItems[0].title, startTime: departureItems[0].startTime, endTime: item?.EndTime, location: item?.Location ?? departureItems[0].destination, leaveAt: departureItems[0].leaveAt, drivingSeconds: departureItems[0].drivingSeconds }, event); }}>Event details <span aria-hidden="true">→</span></button></div></section>}
        {isCardVisible("meals") && meals.length > 0 && <section style={cardStyle("meals")} className={`display-card meal-card ${cardLayoutClass("meals")}`}><div className="display-card-heading"><span><SectionIcon name="meals"/>{mealWindow.label}’s meals</span></div><div className="meal-agenda">{(["breakfast", "lunch", "dinner"] as const).map(type => { const meal = meals.find(item => item.MealType === type); return meal && <div className={`meal-agenda-row${type === mealWindow.nextMeal ? " meal-current" : ""}`} key={`${meal.Date}-${meal.MealType}`}><span>{type}</span><strong>{meal.Name}</strong>{type === mealWindow.nextMeal && <small>Up next</small>}</div>; })}</div><div className="meal-card-footer"><button className="display-card-cta" onClick={event => open("Meal plan", event)}>Edit meals <span aria-hidden="true">→</span></button></div></section>}
        {isCardVisible("calendar") && events.length > 0 && <section style={cardStyle("calendar")} className={`display-card calendar-card ${cardLayoutClass("calendar")}`} aria-label="Upcoming calendar events" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="calendar"/>Coming up</span><small>{events.length} {events.length === 1 ? "event" : "events"}</small></div>{events.slice(0, 3).map(item => <button className="display-event" key={item.ID} onClick={event => openEvent({ title: item.Title, startTime: item.StartTime, endTime: item.EndTime, location: item.Location }, event)}><time>{item.AllDay ? "All day" : new Date(item.StartTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><span className="event-line" style={{"--event-color":item.Color ?? "#718da6"} as React.CSSProperties} aria-hidden="true"/><span><strong>{item.Title}</strong>{item.Location && <small>{item.Location}</small>}</span></button>)}</section>}
        {isCardVisible("chores") && chores.length > 0 && <section style={cardStyle("chores")} className={`display-card chores-card ${cardLayoutClass("chores")}`} aria-label="Household chores" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="chores"/>A few things to do</span><small>{chores.length} left</small></div>{chores.slice(0, 4).map(chore => <button className="display-chore" key={chore.ID} onClick={event => { setSelectedChore(chore); open("Chore completion", event); }}><span className="chore-ring" aria-hidden="true"/><span><strong>{chore.Name}</strong><small>{chore.AssignedToUserID ? current?.people.find(person => person.id === chore.AssignedToUserID)?.name ?? "Assigned" : "Anyone"}</small></span></button>)}</section>}
        {isCardVisible("bins") && collections.length > 0 && <section style={cardStyle("bins")} className={`display-card bins-card ${cardLayoutClass("bins")}`} aria-label="Bin collections" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="bins"/>{collections[0].date === dateKey ? "Bins collected today" : "Bins out tonight"}</span></div>{collections.map(bin => <div className="bin-row" key={`${bin.date}-${bin.type}`}><strong>{bin.name || bin.type}</strong><span>{new Date(`${bin.date}T12:00:00`).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</span></div>)}</section>}
        {isCardVisible("trains") && trains.length > 0 && <section style={cardStyle("trains")} className={`display-card trains-card ${cardLayoutClass("trains")}`} aria-label="Next trains" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="trains"/>Next trains</span></div><small className="route-caption">{commute?.boards[0]?.origin.code} to {commute?.boards[0]?.destination.code}</small>{trains.slice(0, 3).map(({ board, train }) => <div className="train-row" key={`${board.origin.code}-${train.serviceId}`}><time>{train.expectedDeparture || train.scheduledDeparture || "—"}</time><strong>{train.platform ? `Platform ${train.platform}` : train.operator || "Train"}</strong><small>{train.status.replaceAll("_", " ")}</small></div>)}</section>}
      </div>
      <div className="display-controls"><nav className="display-launcher" aria-label="Family apps">{launcherApps.map(({ name, icon: Icon }) => <button key={name} onClick={event => open(name, event)}><Icon size={21} strokeWidth={1.7} aria-hidden="true" className="display-icon"/><span>{name}</span></button>)}</nav>
        <footer className="display-footer"><button aria-label={`Refresh display data${current ? `; last successful refresh ${latestRefresh(current.resourceUpdatedAt, timezone)}` : ""}`} onClick={() => void refresh()}>Refresh{current && <small> ({latestRefresh(current.resourceUpdatedAt, timezone)})</small>}</button></footer>
      </div>
    </section>
    {selectedApp && <div className="display-overlay" onClick={event => { if (event.target === event.currentTarget) close(); }}>
      <section ref={dialogRef} className={`display-dialog${selectedApp === "Weather" ? " weather-dialog" : ""}`} role="dialog" aria-modal="true" aria-labelledby="display-dialog-title" tabIndex={-1}>
        <header className="display-dialog-heading"><div><h2 id="display-dialog-title">{selectedApp === "Weather" ? "Weather forecast" : selectedApp}</h2></div><button className="close-button" onClick={close} aria-label="Close"><X size={24} strokeWidth={1.7} aria-hidden="true"/></button></header>
        <div className="display-dialog-content">
          {selectedApp === "Chore completion" && selectedChore ? <ChoreCompletionDialog chore={selectedChore} people={current?.people ?? []} token={token} onComplete={() => { void close(); void refresh(); }} /> : selectedApp === "Family event" && selectedEvent ? <div className="event-detail"><p>{new Date(selectedEvent.startTime).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</p><h3>{selectedEvent.title}</h3><p>{new Date(selectedEvent.startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{selectedEvent.endTime ? `–${new Date(selectedEvent.endTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}` : ""}</p>{selectedEvent.location && <p>{selectedEvent.location}</p>}{selectedEvent.leaveAt && <p className="event-travel">Leave at {new Date(selectedEvent.leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{selectedEvent.drivingSeconds ? ` · ${Math.round(selectedEvent.drivingSeconds / 60)} min drive` : ""}</p>}</div> : selectedApp === "launcher" ? <div className="launcher-grid">{[["Shopping", "＋"], ["Scratchpad", "✎"], ["Calendar", "▦"], ["Recipes", "♧"], ["Meal plan", "♨"], ["Inventory", "▤"]].map(([app, symbol]) => <button className="launcher-tile" key={app} onClick={event => open(app, event)}><span aria-hidden="true">{symbol}</span>{app}</button>)}</div> : <KioskApp app={selectedApp} token={token} initialMeal={mealEditTarget} timezone={timezone} onScratchpadSaveReady={registerScratchpadSave} />}
        </div>
      </section>
    </div>}
  </main>;
}

function latestRefresh(resourceUpdatedAt: Record<string, string>, timezone: string) {
  const timestamps = Object.values(resourceUpdatedAt).map(Date.parse).filter(Number.isFinite);
  if (!timestamps.length) return "saved information";
  return new Date(Math.max(...timestamps)).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });
}

function WeatherIcon({ code }: { code: number }) {
  if ([0, 1, 2].includes(code)) return <Sun size={21} strokeWidth={1.7} aria-hidden="true"/>;
  if ([3, 45, 48].includes(code)) return <span className="weather-cloud-icon" aria-hidden="true">☁</span>;
  return <span className="weather-rain-icon" aria-hidden="true">☂</span>;
}

function resourceName(key: string) {
  const names: Record<string, string> = { calendar: "Calendar", weather: "Weather", departures: "Journeys", bins: "Bin collections", people: "Household members", commute: "Commute settings", trains: "Train times" };
  return names[key] ?? key;
}

function ChoreCompletionDialog({ chore, people, token, onComplete }: { chore: DisplayChore; people: DisplayPerson[]; token: string; onComplete: () => void }) {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  async function complete(userID: string) {
    setPending(true);
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/chores/${encodeURIComponent(chore.ID)}/complete`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" }, body: JSON.stringify({ userID }) });
      if (!response.ok) throw new Error(response.status === 409 ? "This chore was already completed." : "The chore could not be completed. Try again.");
      onComplete();
    } catch (reason) { setError(reason instanceof Error ? reason.message : "The chore could not be completed. Try again."); }
    finally { setPending(false); }
  }
  return <section className="person-picker"><h3>Who completed {chore.Name}?</h3>{people.map(person => <button className="person-button" key={person.id} disabled={pending} onClick={() => void complete(person.id)}><span className="person-avatar">{person.name.slice(0, 1).toUpperCase()}</span>{person.name}</button>)}{error && <p role="alert">{error}</p>}</section>;
}

function KioskApp({app,token,initialMeal,timezone,onScratchpadSaveReady}:{app:string;token:string;initialMeal:MealEditTarget|null;timezone:string;onScratchpadSaveReady:(save:(()=>Promise<void>)|null)=>void}){return app==="Notes"?<KioskScratchpad token={token} onSaveReady={onScratchpadSaveReady}/>:<KioskDataApp app={app} token={token} initialMeal={initialMeal} timezone={timezone}/>}
function KioskDataApp({app,token,initialMeal,timezone}:{app:string;token:string;initialMeal:MealEditTarget|null;timezone:string}){
  const [data,setData]=useState<unknown>(null);const [error,setError]=useState("");const [busy,setBusy]=useState("");const [week,setWeek]=useState(()=>weekStartAt(new Date(),timezone));const [selected,setSelected]=useState("");const [mealName,setMealName]=useState("");const [recipeDetails,setRecipeDetails]=useState<Record<string,DisplayRecipe>>({});const [recipeErrors,setRecipeErrors]=useState<Record<string,string>>({});const [newMealDate,setNewMealDate]=useState(()=>dayKeyAt(new Date(),timezone));const [newMealType,setNewMealType]=useState<MealType>("dinner");const [newMealName,setNewMealName]=useState("");const [newShoppingName,setNewShoppingName]=useState("");const initialMealApplied=useRef(false);
  const path=app==="Shopping"?"/api/display/shopping":app==="Calendar"?"/api/display/calendar?view=week&date="+week:app==="Recipes"?"/api/display/recipes":app==="Meal plan"?"/api/display/meals"+(initialMeal?"?week="+initialMeal.date:""):app==="Inventory"?"/api/display/inventory":app==="Journeys"?"/api/display/departures":app==="Trains"?"/api/display/trains":app==="Bins"?"/api/display/bin-collections":"/api/display/weather";
  const load=useCallback(async()=>{setError("");try{setData(await displayRequest<unknown>(path,token))}catch{setError("This family information is unavailable right now.")}},[path,token]);useEffect(()=>{void load()},[load]);
  useEffect(()=>{if(app!=="Meal plan"||!initialMeal||!Array.isArray(data)||initialMealApplied.current)return;initialMealApplied.current=true;setNewMealDate(initialMeal.date);setNewMealType(initialMeal.mealType);const meal=(data as DisplayMeal[]).find(item=>item.Date===initialMeal.date&&item.MealType===initialMeal.mealType);if(meal){setSelected(meal.Date+meal.MealType);setMealName(meal.Name)}},[app,data,initialMeal]);
  async function updateShopping(item:{id:string;name:string;purchased:boolean;version:number}){setBusy(item.id);setError("");try{await fetch(apiBase+"/api/display/shopping/items/"+item.id,{method:"PATCH",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({purchased:!item.purchased,version:item.version})}).then(async response=>{if(!response.ok)throw new Error();return response.json()});await load()}catch{setError("We couldn’t update the shopping list. Please try again.")}finally{setBusy("")}}
  async function addShopping(event:FormEvent<HTMLFormElement>){event.preventDefault();const name=newShoppingName.trim();if(!name)return;setBusy("new-shopping");setError("");try{const response=await fetch(apiBase+"/api/display/shopping/items",{method:"POST",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({name})});if(!response.ok)throw new Error();setNewShoppingName("");await load()}catch{setError("We couldn’t add that item. Please try again.")}finally{setBusy("")}}
  async function changeStock(item:{ID:string;TrackingMode:string;Quantity:number;Level:number},direction:number){setBusy(item.ID);setError("");const change=item.TrackingMode==="level"?{level:Math.max(0,Math.min(100,item.Level+direction*25))}:{quantityDelta:direction};try{const response=await fetch(apiBase+"/api/display/inventory/items/"+encodeURIComponent(item.ID)+"/stock",{method:"PUT",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify(change)});if(!response.ok)throw new Error();await load()}catch{setError("Inventory could not be updated. Please try again.")}finally{setBusy("")}}
  async function addMeal(event:FormEvent<HTMLFormElement>){event.preventDefault();const name=newMealName.trim();if(!name)return;setBusy("new-meal");setError("");try{const response=await fetch(apiBase+"/api/display/meals",{method:"POST",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({date:newMealDate,mealType:newMealType,name})});if(!response.ok)throw new Error();setNewMealName("");await load()}catch{setError("We couldn’t save that meal. Please try again.")}finally{setBusy("")}}
  async function loadRecipe(id:string){try{const recipe=await displayRequest<DisplayRecipe>("/api/display/recipes/"+id,token);setRecipeDetails(current=>({...current,[id]:recipe}))}catch{setRecipeErrors(current=>({...current,[id]:"This recipe is unavailable right now."}))}}
  async function saveMeal(event:FormEvent<HTMLFormElement>,meal:DisplayMeal){event.preventDefault();const name=mealName.trim();if(!name)return;setBusy(meal.Date+meal.MealType);setError("");try{const response=await fetch(apiBase+"/api/display/meals",{method:"POST",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({date:meal.Date,mealType:meal.MealType,name})});if(!response.ok)throw new Error();setSelected("");setMealName("");await load()}catch{setError("We couldn’t save that meal. Please try again.")}finally{setBusy("")}}
  if(error&&!data)return <p role="alert">{error}</p>;if(!data)return <p role="status">Loading…</p>;
  if(app==="Shopping")return <div className="kiosk-list"><form className="inline-form" onSubmit={event=>void addShopping(event)}><label>Item<input aria-label="Shopping item" value={newShoppingName} onChange={event=>setNewShoppingName(event.target.value)} placeholder="Add to shopping" /></label><button className="action-button" disabled={!newShoppingName.trim()||busy==="new-shopping"}>{busy==="new-shopping"?"Adding…":"Add item"}</button></form>{error&&<p role="alert">{error}</p>}{(data as {id:string;name:string;purchased:boolean;version:number}[]).map(item=><label className="shopping-check" key={item.id}><input type="checkbox" checked={item.purchased} disabled={busy===item.id} onChange={()=>void updateShopping(item)}/><span>{item.name}</span></label>)}</div>;
  if(app==="Inventory")return <div className="kiosk-list">{error&&<p role="alert">{error}</p>}{(data as {ID:string;Name:string;Items:{ID:string;Name:string;TrackingMode:string;Quantity:number;Level:number;Unit:string}[]}[]).map(area=><section key={area.ID}><h2>{area.Name}</h2>{area.Items.map(item=><div className="stock-row" key={item.ID}><span><strong>{item.Name}</strong><small>{item.TrackingMode==="level"?`${item.Level}%`: `${item.Quantity} ${item.Unit}`}</small></span><div><button aria-label={`Decrease ${item.Name}`} disabled={busy===item.ID||(item.TrackingMode==="level"?item.Level<=0:item.Quantity<=0)} onClick={()=>void changeStock(item,-1)}>−</button><button aria-label={`Increase ${item.Name}`} disabled={busy===item.ID||(item.TrackingMode==="level"?item.Level>=100:false)} onClick={()=>void changeStock(item,1)}>＋</button></div></div>)}</section>)}</div>;
  if(app==="Recipes")return <div className="kiosk-list">{(data as {ID:string;Title:string}[]).map(recipe=>{const detail=recipeDetails[recipe.ID];return <details key={recipe.ID} onToggle={event=>{if(event.currentTarget.open&&!detail&&!recipeErrors[recipe.ID])void loadRecipe(recipe.ID)}}><summary>{recipe.Title}</summary>{recipeErrors[recipe.ID]?<><p role="alert">{recipeErrors[recipe.ID]}</p><button className="kiosk-back" onClick={()=>{setRecipeErrors(current=>({...current,[recipe.ID]:""}));void loadRecipe(recipe.ID)}}>Try again</button></>:!detail?<p role="status">Loading recipe…</p>:<><h3>Ingredients</h3>{(detail.Ingredients??[]).flatMap(group=>group.items).map((ingredient,index)=><p key={index}>{ingredient}</p>)}<h3>Method</h3>{(detail.Steps??(detail.Instructions?[detail.Instructions]:[])).map((step,index)=><p key={index}>{index+1}. {step}</p>)}</>}</details>})}</div>;
  if(app==="Journeys"){const journeys=data as {status:string;items:DisplayDeparture[]};const items=relevantDepartures(journeys.items??[],new Date());return <div className="kiosk-list">{journeys.status!=="available"&&<p role="status">{journeys.status==="unavailable"?"Journey information is unavailable.":"No journeys are scheduled soon."}</p>}{items.map(item=><section key={item.eventID}><h3>{item.title}</h3><p>{item.destination}</p><p>Leave at {new Date(item.leaveAt).toLocaleTimeString("en-GB",{hour:"2-digit",minute:"2-digit"})}</p><small>Event starts {new Date(item.startTime).toLocaleTimeString("en-GB",{hour:"2-digit",minute:"2-digit"})}{item.drivingSeconds?` · ${Math.round(item.drivingSeconds/60)} min drive`:""}</small></section>)}</div>}
  if(app==="Bins"){const bins=data as BinSchedule;return <div className="kiosk-list">{(bins.collections??[]).length===0?<p>No upcoming bin collections are available.</p>:(bins.collections??[]).map(bin=><section key={`${bin.date}-${bin.type}`}><h3>{bin.name||bin.type}</h3><p>{new Date(`${bin.date}T12:00:00`).toLocaleDateString("en-GB",{weekday:"long",day:"numeric",month:"long"})}</p></section>)}</div>}
  if(app==="Trains"){const commute=data as CommuteResult;return <div className="kiosk-list">{commute.status==="setup"?<p>Choose stations and a shared schedule in display settings.</p>:commute.status==="outside_schedule"?<p>No trains are shown outside the shared commute schedule.</p>:commute.status==="unavailable"?<p role="status">Train times are unavailable. The display will retry.</p>:commute.boards.map(board=><section key={board.origin.code+board.destination.code}><h3>{board.origin.code} to {board.destination.code}</h3>{board.trains.map(train=><div className="train-detail" key={train.serviceId}><strong>{train.expectedDeparture||train.scheduledDeparture||"Time unavailable"}</strong><span>{train.status.replaceAll("_"," ")}</span><small>{train.platform?`Platform ${train.platform}`:train.operator||""}</small></div>)}</section>)}</div>}
  if(app==="Calendar"){
    const calendar=data as {events:DisplayEvent[];chores:DisplayChore[]};
    const dates=Array.from({length:7},(_,index)=>new Date(Date.parse(week+"T12:00:00Z")+index*86_400_000));
    const range=new Intl.DateTimeFormat("en-GB",{day:"numeric",month:"long",timeZone:"UTC"});
    const weekLabel=`${range.format(dates[0])} – ${range.format(dates[6])}`;
    const timeSlots=[8,10,12,14,16,18];
    const dayKey=(date:Date)=>date.toISOString().slice(0,10);
    const eventsForSlot=(date:Date,hour:number)=>calendar.events.filter(event=>{
      const eventDate=event.AllDay?event.StartTime.slice(0,10):dayKeyAt(new Date(event.StartTime),timezone);
      if(eventDate!==dayKey(date))return false;
      return event.AllDay?hour===8:hourAt(event.StartTime,timezone)>=hour&&hourAt(event.StartTime,timezone)<hour+2;
    });
    return <div>
      <div className="calendar-toolbar"><button className="calendar-nav" aria-label="Previous week" onClick={()=>setWeek(weekStartAt(new Date(Date.parse(week+"T12:00:00Z")-7*86_400_000),"UTC"))}>‹</button><strong>{weekLabel}</strong><button className="calendar-nav" aria-label="Next week" onClick={()=>setWeek(weekStartAt(new Date(Date.parse(week+"T12:00:00Z")+7*86_400_000),"UTC"))}>›</button><button className="calendar-today" onClick={()=>setWeek(weekStartAt(new Date(),timezone))} disabled={week===weekStartAt(new Date(),timezone)}>Today</button></div>
      {calendar.events.length===0&&<p className="calendar-empty">No events this week.</p>}
      <div className="week-scroll"><div className="week-grid" aria-label={`Family calendar, ${weekLabel}`}>
        <div className="week-corner"/>{dates.map(date=><div className="week-day" key={dayKey(date)}><strong>{new Intl.DateTimeFormat("en-GB",{weekday:"short",timeZone:"UTC"}).format(date)}</strong><span>{date.getUTCDate()}</span></div>)}
        {timeSlots.map(hour=><Fragment key={hour}><div className="week-hour">{String(hour).padStart(2,"0")}:00</div>{dates.map(date=>{const slotEvents=eventsForSlot(date,hour);return <div className="week-cell" key={`${dayKey(date)}-${hour}`}>{slotEvents.map(event=><div className="week-event" key={event.ID} style={{"--event-color":event.Color ?? "#718da6"} as React.CSSProperties}><small>{event.AllDay?"All day":new Date(event.StartTime).toLocaleTimeString("en-GB",{hour:"2-digit",minute:"2-digit",timeZone:timezone})}</small>{event.Title}{event.Location&&<span>{event.Location}</span>}</div>)}</div>})}</Fragment>)}
      </div></div>
      {calendar.chores.length>0&&<section className="calendar-chores"><h3>Chores this week</h3>{calendar.chores.map(chore=><p key={chore.ID}>{chore.Name}</p>)}</section>}
    </div>;
  }
  if(app==="Weather"){const weather=data as Weather;return <div className="weather-forecast"><p className="dialog-label">Today · {weather.location ?? "At home"}</p><div className="hourly-forecast">{weather.hours?.map((hour,index)=><div className="hour-card" key={hour.time}><span>{index===0?"Now":new Date(hour.time).toLocaleTimeString("en-GB",{hour:"2-digit",timeZone:timezone})}</span><WeatherIcon code={hour.code}/><strong>{Math.round(hour.temperature)}°</strong><small>{hour.rainProbability===0?"Dry":`${Math.round(hour.rainProbability)}%`}</small></div>)}</div><p className="dialog-label forecast-days-label">Next seven days</p><div className="weather-days">{weather.days?.map(day=><div className="forecast-row" key={day.date}><span>{new Date(`${day.date}T12:00:00`).toLocaleDateString("en-GB",{weekday:"short",day:"numeric",timeZone:timezone})}</span><WeatherIcon code={day.code}/><strong>{Math.round(day.high)}° <span className="muted">{Math.round(day.low)}°</span></strong><span className="forecast-rain">☂ {Math.round(day.rainProbability ?? 0)}%</span></div>)}</div>{weather.advice&&<p className="forecast-advice">{weather.advice}</p>}<p className="dialog-label forecast-legend">High and low · Rain chance</p></div>}
  return <div className="kiosk-list"><form className="inline-form" onSubmit={event=>void addMeal(event)}><label>Date<input aria-label="Meal date" type="date" value={newMealDate} onChange={event=>setNewMealDate(event.target.value)}/></label><label>Meal<select aria-label="Meal type" value={newMealType} onChange={event=>setNewMealType(event.target.value as MealType)}><option value="breakfast">Breakfast</option><option value="lunch">Lunch</option><option value="dinner">Dinner</option></select></label><label>Meal idea<input aria-label="Meal idea" value={newMealName} onChange={event=>setNewMealName(event.target.value)} placeholder="Add a meal idea"/></label><button className="action-button" disabled={!newMealName.trim()||busy==="new-meal"}>{busy==="new-meal"?"Saving…":"Add meal"}</button></form>{error&&<p role="alert">{error}</p>}{(data as DisplayMeal[]).map(meal=><section key={meal.Date+meal.MealType}><p>{meal.Date} · {meal.MealType}</p>{selected===meal.Date+meal.MealType?<form className="inline-form" onSubmit={event=>void saveMeal(event,meal)}><input aria-label="Meal name" value={mealName} onChange={event=>setMealName(event.target.value)} /><button className="action-button" disabled={busy===selected}>Save</button><button type="button" className="text-action" onClick={()=>setSelected("")}>Cancel</button></form>:<><strong>{meal.Name}</strong><button className="text-action" onClick={()=>{setSelected(meal.Date+meal.MealType);setMealName(meal.Name)}}>Edit</button></>}</section>)}</div>
}
