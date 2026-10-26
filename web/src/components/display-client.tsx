"use client";

import Image from "next/image";
import { FormEvent, MouseEvent, useCallback, useEffect, useRef, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { BookOpen, CalendarDays, CarFront, Check, CookingPot, Monitor, PackageCheck, ShoppingBag, Sun, TrainFront, X, type LucideIcon } from "lucide-react";
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
type Weather = { status: string; fetchedAt?: string; current?: { temperature: number; code: number }; high?: number; low?: number; hours?: { time: string; temperature: number; code: number; rainProbability: number }[]; days?: { date: string; high: number; low: number; code: number; rainProbability?: number }[] };
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

const apiBase = getApiBaseUrl();
const storagePrefix = displayStoragePrefix(apiBase);
const displayTokenKey = `${storagePrefix}.token`;
const snapshotKey = (id: string) => `${storagePrefix}.snapshot.${encodeURIComponent(id)}`;
const overviewSections = ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"] as const;
const sectionIcons: Record<string, LucideIcon> = {
  calendar: CalendarDays, meals: CookingPot, chores: Check, weather: Sun,
  departures: CarFront, trains: TrainFront, bins: PackageCheck, shopping: ShoppingBag,
};
const launcherApps: { name: string; icon: LucideIcon }[] = [
  { name: "Shopping", icon: ShoppingBag }, { name: "Notes", icon: BookOpen },
  { name: "Calendar", icon: CalendarDays }, { name: "Recipes", icon: CookingPot },
  { name: "Meal plan", icon: CookingPot }, { name: "Inventory", icon: PackageCheck },
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
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "We could not start pairing. Check the connection and try again.");
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
      <p>Open Family Hub on your iPhone or iPad. Go to <strong>Settings → Wall displays → Scan a display code</strong>, then approve this screen.</p>
      {pairing ? <div className="qr-wrap"><QRCodeSVG value={qrValue} size={228} level="M" includeMargin /></div> : <div className="qr-empty" role="status">{renewing ? "Preparing a new code…" : "Pairing code is unavailable"}</div>}
      {error && <p className="error-message" role="alert">{error}</p>}
      <small>Each code expires after five minutes and can be used once.</small>
      {(!pairing || error) && <button className="action-button" disabled={renewing} onClick={() => void create()}>{renewing ? "Preparing…" : "Create a new pairing code"}</button>}
      <p className="pair-note">A shared household screen. No personal sign-in is stored here.</p>
    </section>
  </main>;
}

export function DisplayClient() {
  const [token, setToken] = useState("");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [householdName, setHouseholdName] = useState("Family");
  const snapshotRef = useRef<Snapshot | null>(null);
  const [clock, setClock] = useState<Date | null>(null);
  const [selectedApp, setSelectedApp] = useState("");
  const [mealEditTarget, setMealEditTarget] = useState<MealEditTarget | null>(null);
  const [selectedChore, setSelectedChore] = useState<DisplayChore | null>(null);
  const [loading, setLoading] = useState(true);
  const [failures, setFailures] = useState<string[]>([]);
  const dialogRef = useRef<HTMLElement>(null);
  const refreshInFlight = useRef(false);
  const openerRef = useRef<HTMLElement | null>(null);

  const accept = useCallback((value: string) => {
    if (!writeStorage(displayTokenKey, value)) setFailures(["This browser cannot save the pairing. Keep this tab open or enable site storage."]);
    setToken(value);
  }, []);

  useEffect(() => {
    setToken(readStorage(displayTokenKey) ?? "");
    setClock(new Date());
    const timer = window.setInterval(() => setClock(new Date()), 30_000);
    return () => window.clearInterval(timer);
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
      const now = new Date();
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
      const hasPendingAction = dialogRef.current?.querySelector("button:disabled, [aria-busy='true']");
      const drawing = dialogRef.current?.querySelector("canvas[data-drawing='true']");
      if (hasPendingAction || drawing) { timer = window.setTimeout(closeForIdle, 30_000); return; }
      setSelectedApp("");
      setSelectedChore(null);
      setMealEditTarget(null);
    }
    function keydown(event: KeyboardEvent) {
      resetIdle();
      if (event.key === "Escape") { event.preventDefault(); closeForIdle(); return; }
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
  }, [selectedApp]);

  const open = (app: string, event?: MouseEvent<HTMLElement>) => {
    openerRef.current = event?.currentTarget ?? document.activeElement as HTMLElement;
    setSelectedApp(app);
  };
  const close = () => { setSelectedApp(""); setSelectedChore(null); setMealEditTarget(null); };
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
  });
  const cardOrder = Object.fromEntries(rankedCards.map((key, index) => [key, index]));
  const isDark = localHour >= 20 || localHour < 6;
  const formattedDate = clock.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone });
  const formattedTime = clock.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });

  return <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}>
    <section className={`wall-display${isDark ? " is-dark" : ""}`} aria-label="Family Hub wall display">
      <header className="display-header"><div className="display-brand"><div><p>{formattedDate}</p><h1>{displayGreeting(localHour, departureMinutes <= 45)}</h1></div></div><div className="display-header-right">{sections.has("weather") && weatherReady && weather?.current && <button className="weather-summary" onClick={event => open("Weather", event)} aria-label="Open weather forecast"><Sun size={26} strokeWidth={1.7} aria-hidden="true"/><strong>{Math.round(weather.current.temperature)}°</strong><span>{weatherLabel(weather.current.code)}</span></button>}<time>{formattedTime}</time></div></header>
      {failures.length > 0 && <div className="display-status" role="status">{failures.join(" and ")} {current ? "could not be updated. Showing saved information where available." : "are unavailable. The display will keep trying."}</div>}
      {loading && !current && <p className="display-loading" role="status">Loading household information…</p>}
      <div className="display-grid">
        {sections.has("departures") && departureItems.length > 0 && <section style={{ order: cardOrder.departures }} className="display-card departure-card"><div className="display-card-heading"><span><SectionIcon name="departures"/>Time to head out</span><small>Leave at {new Date(departureItems[0].leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</small></div><Freshness snapshot={current} resource="departures" timezone={timezone}/><button className="display-card-action" onClick={event => open("Journeys", event)}><time>{new Date(departureItems[0].leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><h2>{departureItems[0].title}</h2><p>{departureItems[0].destination}{departureItems[0].drivingSeconds ? ` · ${Math.round(departureItems[0].drivingSeconds / 60)} min drive` : ""}</p><small>Event starts {new Date(departureItems[0].startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</small></button></section>}
        {sections.has("meals") && meals.length > 0 && <section style={{ order: cardOrder.meals }} className="display-card meal-card"><div className="display-card-heading"><span><SectionIcon name="meals"/>{mealWindow.label}’s meals</span><small>{mealWindow.nextMeal} is next</small></div><Freshness snapshot={current} resource="meals" timezone={timezone}/><div className="meal-slots">{meals.map(meal => <button key={`${meal.Date}-${meal.MealType}`} onClick={event => { setMealEditTarget({ date: meal.Date, mealType: meal.MealType }); open("Meal plan", event); }}><small>{meal.MealType}{meal.MealType === mealWindow.nextMeal ? " · Up next" : ""}</small><strong>{meal.Name}</strong></button>)}</div></section>}
        {sections.has("calendar") && events.length > 0 && <section style={{ order: cardOrder.calendar }} className="display-card calendar-card"><div className="display-card-heading"><span><SectionIcon name="calendar"/>Coming up</span><button className="text-action" onClick={event => open("Calendar", event)}>Open calendar</button></div><Freshness snapshot={current} resource="events" timezone={timezone}/>{events.slice(0, 3).map(item => <button className="display-event" key={item.ID} onClick={event => open("Calendar", event)}><time>{item.AllDay ? "All day" : new Date(item.StartTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><span><strong>{item.Title}</strong>{item.Location && <small>{item.Location}</small>}</span></button>)}</section>}
        {sections.has("chores") && chores.length > 0 && <section style={{ order: cardOrder.chores }} className="display-card chores-card"><div className="display-card-heading"><span><SectionIcon name="chores"/>Things to do</span><small>{chores.length} pending</small></div><Freshness snapshot={current} resource="chores" timezone={timezone}/>{chores.slice(0, 4).map(chore => <button className="display-chore" key={chore.ID} onClick={event => { setSelectedChore(chore); open("Chore completion", event); }}><span className="chore-ring" aria-hidden="true"/><span><strong>{chore.Name}</strong><small>{chore.AssignedToUserID ? current?.people.find(person => person.id === chore.AssignedToUserID)?.name ?? "Assigned" : "Choose who completed this"}</small></span></button>)}</section>}
        {sections.has("weather") && weatherReady && weather?.current && <section style={{ order: cardOrder.weather }} className="display-card weather-card"><div className="display-card-heading"><span><SectionIcon name="weather"/>At home</span><small>{weatherLabel(weather.current.code)}</small></div><Freshness snapshot={current} resource="weather" timezone={timezone}/><button className="weather-overview" onClick={event => open("Weather", event)}><strong>{Math.round(weather.current.temperature)}°</strong><span>{weatherLabel(weather.current.code)}{weather.high !== undefined ? ` · H ${Math.round(weather.high)}° L ${Math.round(weather.low ?? 0)}°` : ""}</span></button></section>}
        {sections.has("bins") && collections.length > 0 && <section style={{ order: cardOrder.bins }} className="display-card bins-card"><div className="display-card-heading"><span><SectionIcon name="bins"/>{collections[0].date === dateKey ? "Bins collected today" : "Bins out tonight"}</span><button className="text-action" onClick={event => open("Bins", event)}>Details</button></div><Freshness snapshot={current} resource="bins" timezone={timezone}/>{collections.map(bin => <div className="bin-row" key={`${bin.date}-${bin.type}`}><strong>{bin.name || bin.type}</strong><span>{new Date(`${bin.date}T12:00:00`).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</span></div>)}</section>}
        {sections.has("trains") && trains.length > 0 && <section style={{ order: cardOrder.trains }} className="display-card trains-card"><div className="display-card-heading"><span><SectionIcon name="trains"/>Next trains</span><button className="text-action" onClick={event => open("Trains", event)}>Details</button></div><Freshness snapshot={current} resource="trains" timezone={timezone}/><small className="route-caption">{commute?.boards[0]?.origin.code} to {commute?.boards[0]?.destination.code}</small>{trains.slice(0, 3).map(({ board, train }) => <div className="train-row" key={`${board.origin.code}-${train.serviceId}`}><time>{train.expectedDeparture || train.scheduledDeparture || "—"}</time><strong>{train.platform ? `Platform ${train.platform}` : train.operator || "Train"}</strong><small>{train.status.replaceAll("_", " ")}</small></div>)}</section>}
      </div>
      <nav className="display-launcher" aria-label="Family apps">{launcherApps.map(({ name, icon: Icon }) => <button key={name} onClick={event => open(name, event)}><Icon size={21} strokeWidth={1.7} aria-hidden="true" className="display-icon"/><span>{name}</span></button>)}</nav>
      <footer className="display-footer"><span>{current ? `Last successful data refresh ${latestRefresh(current.resourceUpdatedAt, timezone)}` : "Family Hub"}</span><button onClick={() => void refresh()}>Refresh</button></footer>
    </section>
    {selectedApp && <div className="display-overlay" onClick={event => { if (event.target === event.currentTarget) close(); }}>
      <section ref={dialogRef} className="display-dialog" role="dialog" aria-modal="true" aria-labelledby="display-dialog-title" tabIndex={-1}>
        <header className="display-dialog-heading"><div><Image src="/family-hub-mark.png" alt="" width={36} height={36} /><h2 id="display-dialog-title">{selectedApp}</h2></div><button className="close-button" onClick={close} aria-label="Close"><X size={24} strokeWidth={1.7} aria-hidden="true"/></button></header>
        <div className="display-dialog-content">
          {selectedApp === "Chore completion" && selectedChore ? <ChoreCompletionDialog chore={selectedChore} people={current?.people ?? []} token={token} onComplete={() => { close(); void refresh(); }} /> : selectedApp === "launcher" ? <div className="launcher-grid">{[["Shopping", "＋"], ["Scratchpad", "✎"], ["Calendar", "▦"], ["Recipes", "♧"], ["Meal plan", "♨"], ["Inventory", "▤"]].map(([app, symbol]) => <button className="launcher-tile" key={app} onClick={event => open(app, event)}><span aria-hidden="true">{symbol}</span>{app}</button>)}</div> : <KioskApp app={selectedApp} token={token} initialMeal={mealEditTarget} />}
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

function Freshness({ snapshot, resource, timezone }: { snapshot: Snapshot | null; resource: string; timezone: string }) {
  const status = snapshot?.availability[resource] ?? "unavailable";
  const timestamp = snapshot?.resourceUpdatedAt[resource];
  const updated = timestamp ? new Date(timestamp).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone }) : "";
  const label = status === "available" && updated ? `Updated ${updated}` : status === "stale" && updated ? `Saved · ${updated}` : "Unavailable";
  return <small className={`display-freshness is-${status}`} aria-label={`${resource}: ${label}`}>{label}</small>;
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

function KioskApp({app,token,initialMeal}:{app:string;token:string;initialMeal:MealEditTarget|null}){return app==="Notes"?<KioskScratchpad token={token}/>:<KioskDataApp app={app} token={token} initialMeal={initialMeal}/>}
function KioskDataApp({app,token,initialMeal}:{app:string;token:string;initialMeal:MealEditTarget|null}){
  const [data,setData]=useState<unknown>(null);const [error,setError]=useState("");const [busy,setBusy]=useState("");const [week,setWeek]=useState(()=>dayKeyAt(new Date(),"Europe/London"));const [selected,setSelected]=useState("");const [mealName,setMealName]=useState("");const [recipeDetails,setRecipeDetails]=useState<Record<string,DisplayRecipe>>({});const [recipeErrors,setRecipeErrors]=useState<Record<string,string>>({});const [newMealDate,setNewMealDate]=useState(()=>dayKeyAt(new Date(),"Europe/London"));const [newMealType,setNewMealType]=useState<MealType>("dinner");const [newMealName,setNewMealName]=useState("");const [newShoppingName,setNewShoppingName]=useState("");const initialMealApplied=useRef(false);
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
  if(app==="Calendar"){const calendar=data as {events:DisplayEvent[];chores:DisplayChore[]};return <div className="kiosk-list"><div className="calendar-toolbar"><button className="icon-action" aria-label="Previous week" onClick={()=>setWeek(dayKeyAt(new Date(Date.parse(week+"T12:00:00Z")-7*86400000),"UTC"))}>←</button><strong>Week of {week}</strong><button className="icon-action" aria-label="Next week" onClick={()=>setWeek(dayKeyAt(new Date(Date.parse(week+"T12:00:00Z")+7*86400000),"UTC"))}>→</button><button className="text-action" onClick={()=>setWeek(dayKeyAt(new Date(),"Europe/London"))}>Today</button></div>{calendar.events.map(event=><details key={event.ID}><summary>{new Date(event.StartTime).toLocaleString()} · {event.Title}</summary>{event.Location&&<p>{event.Location}</p>}</details>)}{calendar.chores.map(chore=><p key={chore.ID}>Chore · {chore.Name}</p>)}</div>}
  if(app==="Weather"){const weather=data as Weather;return <div className="kiosk-list"><h2>{weather.current?.temperature}° at home</h2><div className="forecast-week">{weather.days?.map(day=><div key={day.date}><small>{new Date(day.date).toLocaleDateString([], {weekday:"long"})}</small><strong>{Math.round(day.high)}°</strong><span>{Math.round(day.low)}° · {weatherLabel(day.code)}{day.rainProbability===undefined?"":" · "+Math.round(day.rainProbability)+"% rain"}</span></div>)}</div></div>}
  return <div className="kiosk-list"><form className="inline-form" onSubmit={event=>void addMeal(event)}><label>Date<input aria-label="Meal date" type="date" value={newMealDate} onChange={event=>setNewMealDate(event.target.value)}/></label><label>Meal<select aria-label="Meal type" value={newMealType} onChange={event=>setNewMealType(event.target.value as MealType)}><option value="breakfast">Breakfast</option><option value="lunch">Lunch</option><option value="dinner">Dinner</option></select></label><label>Meal idea<input aria-label="Meal idea" value={newMealName} onChange={event=>setNewMealName(event.target.value)} placeholder="Add a meal idea"/></label><button className="action-button" disabled={!newMealName.trim()||busy==="new-meal"}>{busy==="new-meal"?"Saving…":"Add meal"}</button></form>{error&&<p role="alert">{error}</p>}{(data as DisplayMeal[]).map(meal=><section key={meal.Date+meal.MealType}><p>{meal.Date} · {meal.MealType}</p>{selected===meal.Date+meal.MealType?<form className="inline-form" onSubmit={event=>void saveMeal(event,meal)}><input aria-label="Meal name" value={mealName} onChange={event=>setMealName(event.target.value)} /><button className="action-button" disabled={busy===selected}>Save</button><button type="button" className="text-action" onClick={()=>setSelected("")}>Cancel</button></form>:<><strong>{meal.Name}</strong><button className="text-action" onClick={()=>{setSelected(meal.Date+meal.MealType);setMealName(meal.Name)}}>Edit</button></>}</section>)}</div>
}
