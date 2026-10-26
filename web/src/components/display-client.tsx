"use client";

import { QRCodeSVG } from "qrcode.react";
import { useCallback, useEffect, useRef, useState } from "react";
import { getApiBaseUrl } from "@/lib/api";

type DisplayEvent = { ID: string; Title: string; StartTime: string; Location: string };
type DisplayChore = { ID: string; Name: string; Status: string; DueDate?: string | null; AssignedToUserID?: string | null };
type DisplayMeal = { Date: string; MealType: string; Name: string };
type Weather = { status: string; current?: { temperature: number; code: number }; high?: number; low?: number; hours?: { time: string; temperature: number; code: number; rainProbability: number }[]; days?: { date: string; high: number; low: number; code: number; rainProbability?: number }[] };
type Departure = { eventID: string; title: string; destination: string; startTime: string; leaveAt: string; drivingSeconds: number };
type BinSchedule = { collections?: { date: string; type: string; name: string }[]; enabled?: boolean };
type Snapshot = { events: DisplayEvent[]; chores: DisplayChore[]; meals: DisplayMeal[]; weather: Weather; departures: Departure[]; bins: BinSchedule; enabledSections: string[]; timezone: string; updatedAt: string };
type Pairing = { challengeId: string; pairingCode: string; expiresAt: string };
type MealType = "breakfast" | "lunch" | "dinner";
const displayTokenKey = "family-hub-display-token";
const snapshotKey = "family-hub-display-snapshot";
const apiBase = getApiBaseUrl();
class DisplayAccessRevoked extends Error {}

async function displayRequest<T>(path: string, token: string): Promise<T> {
  const response = await fetch(apiBase + path, { headers: { Authorization: "Bearer " + token } });
  if (response.status === 401 || response.status === 403) throw new DisplayAccessRevoked();
  if (!response.ok) throw new Error("Display data is temporarily unavailable.");
  return response.json() as Promise<T>;
}

function dayKeyAt(date: Date, timezone: string) {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: timezone, year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(date);
  const value = (type: string) => parts.find(part => part.type === type)?.value ?? "00";
  return `${value("year")}-${value("month")}-${value("day")}`;
}
function nextDayKey(key: string) { const [year, month, day] = key.split("-").map(Number); const next = new Date(Date.UTC(year, month - 1, day + 1)); return `${next.getUTCFullYear()}-${String(next.getUTCMonth() + 1).padStart(2, "0")}-${String(next.getUTCDate()).padStart(2, "0")}`; }

function PairDisplay({ onClaimed }: { onClaimed: (token: string) => void }) {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [error, setError] = useState("");
  const [renewing, setRenewing] = useState(false);
  const [clock, setClock] = useState<Date | null>(null);
  const create = useCallback(async () => {
    setRenewing(true); setError("");
    try { const response = await fetch(apiBase + "/api/display-pairing/challenges", { method: "POST" }); if (!response.ok) throw new Error("We could not start pairing. Please try again."); setPairing(await response.json() as Pairing); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "We could not start pairing. Please try again."); }
    finally { setRenewing(false); }
  }, []);
  useEffect(() => { setClock(new Date()); const timer = window.setInterval(() => setClock(new Date()), 30_000); void create(); return () => window.clearInterval(timer); }, [create]);
  useEffect(() => {
    if (!pairing) return;
    let stopped = false;
    async function checkClaim() {
      try {
        const response = await fetch(apiBase + "/api/display-pairing/challenges/" + pairing!.challengeId + "/claim", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ pairingCode: pairing!.pairingCode }) });
        if (response.ok) { const body = await response.json() as { token: string }; if (!stopped) onClaimed(body.token); }
        else if (response.status !== 409 && !stopped) setError("We lost the pairing connection. Please try again.");
      } catch { if (!stopped) setError("We lost the pairing connection. Please try again."); }
    }
    const timer = window.setInterval(() => { if (Date.parse(pairing.expiresAt) <= Date.now()) { setPairing(null); setError("This pairing code expired. Create a new one to continue."); } else void checkClaim(); }, 8000);
    return () => { stopped = true; window.clearInterval(timer); };
  }, [pairing, onClaimed]);
  const qrValue = pairing ? JSON.stringify({ apiBaseUrl: apiBase, challengeId: pairing.challengeId, pairingCode: pairing.pairingCode }) : "";
  const pairingHour = clock ? Number(new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/London", hour: "2-digit", hourCycle: "h23" }).format(clock)) : 12;
  const isDark = pairingHour >= 20 || pairingHour < 6;
  return <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}><section className={`wall-display${isDark ? " is-dark" : ""}`} aria-label="Family Hub display pairing"><div className="wall-pairing"><span className="wall-pair-mark" aria-hidden="true">f</span><h1>{error.includes("expired") ? "Let’s try that again." : "Bring your household together."}</h1><p>Open Family Hub on your iPhone.<br/>Go to Settings → Displays → Scan QR code.</p>{pairing ? <div className="wall-qr"><QRCodeSVG value={qrValue} size={170} level="M" includeMargin /></div> : <div className="wall-qr wall-qr-empty" role="status">{renewing ? "Preparing a new code…" : "Pairing code is unavailable"}</div>}<small>Single-use code · Expires in five minutes</small>{error && <p className="wall-pair-error" role="alert">{error}</p>}<div className="wall-pair-owner">A shared household screen. No personal login needed here.</div>{(!pairing || error) && <button className="action-button" disabled={renewing} onClick={() => void create()}>{renewing ? "Preparing…" : "Create a new pairing code"}</button>}</div></section></main>;
}

export function DisplayClient() {
  const [token, setToken] = useState("");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const snapshotRef = useRef<Snapshot | null>(null);
  const [offline, setOffline] = useState(false);
  const [clock, setClock] = useState<Date | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const accept = useCallback((value: string) => { localStorage.setItem(displayTokenKey, value); setToken(value); }, []);
  useEffect(() => {
    const saved = localStorage.getItem(displayTokenKey) ?? "";
    setToken(saved);
    const cached = localStorage.getItem(snapshotKey);
    if (cached) { try { const value = JSON.parse(cached) as Snapshot; snapshotRef.current = value; setSnapshot(value); } catch { localStorage.removeItem(snapshotKey); } }
  }, []);
  useEffect(() => { setClock(new Date()); const timer = window.setInterval(() => setClock(new Date()), 30_000); return () => window.clearInterval(timer); }, []);
  const refresh = useCallback(async () => {
    if (!token) return;
    const timezone = snapshotRef.current?.timezone ?? "Europe/London";
    const date = dayKeyAt(new Date(), timezone);
    const results = await Promise.allSettled([
      displayRequest<{ timezone: string }>("/api/display/home-info", token),
      displayRequest<{ events: DisplayEvent[]; chores: DisplayChore[]; meals: DisplayMeal[] }>("/api/display/calendar?view=week&date=" + date, token),
      displayRequest<Weather>("/api/display/weather", token),
      displayRequest<{ items?: Departure[] }>("/api/display/departures", token),
      displayRequest<BinSchedule>("/api/display/bin-collections", token),
      displayRequest<{ enabledSections: string[] }>("/api/display/settings", token),
    ]);
    const revoked = results.some(result => result.status === "rejected" && result.reason instanceof DisplayAccessRevoked);
    if (revoked) { localStorage.removeItem(displayTokenKey); localStorage.removeItem(snapshotKey); snapshotRef.current = null; setSnapshot(null); setToken(""); setOffline(false); setLoading(false); return; }
    const previous = snapshotRef.current;
    const value: Snapshot = {
      timezone: results[0].status === "fulfilled" ? results[0].value.timezone : timezone,
      events: results[1].status === "fulfilled" ? results[1].value.events ?? [] : previous?.events ?? [],
      chores: results[1].status === "fulfilled" ? results[1].value.chores ?? [] : previous?.chores ?? [],
      meals: results[1].status === "fulfilled" ? results[1].value.meals ?? [] : previous?.meals ?? [],
      weather: results[2].status === "fulfilled" ? results[2].value : previous?.weather ?? { status: "unavailable" },
      departures: results[3].status === "fulfilled" ? results[3].value.items ?? [] : previous?.departures ?? [],
      bins: results[4].status === "fulfilled" ? results[4].value : previous?.bins ?? {},
      enabledSections: results[5].status === "fulfilled" ? results[5].value.enabledSections : previous?.enabledSections ?? ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"],
      updatedAt: new Date().toISOString(),
    };
    const failed = results.some(result => result.status === "rejected");
    snapshotRef.current = value;
    setSnapshot(value);
    localStorage.setItem(snapshotKey, JSON.stringify(value));
    setOffline(failed);
    setError(failed && !previous ? "We’re having trouble connecting. The display will keep trying." : "");
    setLoading(false);
  }, [token]);
  useEffect(() => { if (!token) return; void refresh(); const timer = window.setInterval(() => void refresh(), 60_000); return () => window.clearInterval(timer); }, [token, refresh]);
  if (!token) return <PairDisplay onClaimed={accept} />;
  if (!clock) return <main className="wall-display-shell" aria-busy="true" />;
  const timezone = snapshot?.timezone ?? "Europe/London";
  const dateKey = dayKeyAt(clock, timezone);
  const hour = Number(new Intl.DateTimeFormat("en-GB", { timeZone: timezone, hour: "2-digit", hourCycle: "h23" }).format(clock));
  const tomorrow = hour >= 18;
  const mealDate = tomorrow ? nextDayKey(dateKey) : dateKey;
  const nextMeal: MealType = tomorrow || hour < 11 ? "breakfast" : hour < 16 ? "lunch" : "dinner";
  const events = (snapshot?.events ?? []).filter(item => dayKeyAt(new Date(item.StartTime), timezone) === dateKey).sort((first, second) => first.StartTime.localeCompare(second.StartTime));
  const chores = (snapshot?.chores ?? []).filter(item => item.Status === "pending" || item.Status === "overdue").slice(0, 3);
  const departure = (snapshot?.departures ?? []).find(item => !item.startTime || new Date(item.startTime) >= clock) ?? snapshot?.departures[0];
  const bins = (snapshot?.bins.collections ?? []).filter(item => item.date >= dateKey).slice(0, 1);
  const meals = snapshot?.meals ?? [];
  const sections = new Set(snapshot?.enabledSections ?? ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"]);
  const date = clock.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone });
  const time = clock.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });
  const currentWeather = snapshot?.weather.current;
  const todaysMeals = meals.filter(item => item.Date === mealDate);
  const appearanceHour = Number(new Intl.DateTimeFormat("en-GB", { timeZone: timezone, hour: "2-digit", hourCycle: "h23" }).format(clock));
  const isDark = appearanceHour >= 20 || appearanceHour < 6;
  return <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}><section className={`wall-display${isDark ? " is-dark" : ""}`} aria-label="Family Hub wall display">
    <header className="wall-header"><div><p>{date} · Willow household</p><h1>{departure ? (hour >= 16 ? "Home for the evening." : "Home for the day.") : "A little room for a good day."}</h1></div><div className="wall-weather">{sections.has("weather") && <><span aria-hidden="true">☀</span><div><strong>{currentWeather ? `${Math.round(currentWeather.temperature)}°` : "—"}</strong><small>{currentWeather ? "Mostly sunny" : loading ? "Loading forecast" : "Forecast unavailable"}{snapshot?.weather.high !== undefined ? ` · H ${Math.round(snapshot.weather.high)}° L ${Math.round(snapshot.weather.low ?? 0)}°` : ""}</small></div></>}<time>{time}</time></div></header>
    {offline && <div className="wall-status" role="status">Connection lost · Showing saved information · Trying to reconnect</div>}{error && <div className="wall-status" role="alert">{error}</div>}
    <div className="wall-primary-grid">
      {sections.has("departures") && <section className="wall-card departure-card"><div className="wall-card-heading"><span aria-hidden="true">⌑</span><strong>Time to head out</strong>{departure && <small>Leave in {Math.max(0, Math.round((new Date(departure.leaveAt).getTime() - clock.getTime()) / 60_000))} min</small>}</div>{departure ? <><time className="departure-time">{new Date(departure.startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><h2>{departure.title}</h2><p>{departure.destination}{departure.drivingSeconds ? ` · ${Math.round(departure.drivingSeconds / 60)} min drive` : ""}</p><small className="wall-departure-foot">Starts at {new Date(departure.startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</small></> : <div className="wall-empty"><h2>No journeys coming up</h2><p>Your family's next departure will appear here.</p></div>}</section>}
      {sections.has("meals") && <section className="wall-card meals-card"><div className="wall-card-heading"><span aria-hidden="true">♧</span><strong>Today’s meals</strong></div><div className="wall-meal-list">{(["breakfast", "lunch", "dinner"] as MealType[]).map(type => { const meal = todaysMeals.find(item => item.MealType === type); return <div className={type === nextMeal ? "wall-meal-row next" : "wall-meal-row"} key={type}><span>{type}</span><strong>{meal?.Name ?? "Not planned"}</strong>{type === nextMeal && <small>Up next</small>}</div>; })}</div><div className="wall-meal-footer">Edit meals <span>→</span></div></section>}
    </div>
    <div className="wall-secondary-grid">
      {sections.has("bins") && <section className="wall-card wall-small-card"><div className="wall-card-heading"><span aria-hidden="true">♲</span><strong>Bins out tonight</strong></div>{bins.length ? <><h2>{bins[0].name || bins[0].type}</h2><p>Collection {new Date(bins[0].date + "T12:00:00").toLocaleDateString("en-GB", { weekday: "long" })}<br/>{new Date(bins[0].date + "T12:00:00").toLocaleDateString("en-GB", { day: "numeric", month: "long" })}</p><small>Leave them out before bedtime</small></> : <div className="wall-empty"><h2>No collection scheduled</h2><p>Upcoming bin collections will appear here.</p></div>}</section>}
      {sections.has("calendar") && <section className="wall-card wall-small-card"><div className="wall-card-heading"><span aria-hidden="true">▦</span><strong>Coming up</strong><small>{events.length} {events.length === 1 ? "event" : "events"}</small></div>{events.length ? <>{events.slice(0, 2).map(event => <div className="wall-event-row" key={event.ID}><time>{new Date(event.StartTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><div><strong>{event.Title}</strong><small>{event.Location || "Shared family calendar"}</small></div></div>)}<small className="wall-card-foot">Shared family calendar</small></> : <div className="wall-empty"><h2>Space in the calendar</h2><p>Nothing else planned today.</p></div>}</section>}
      {sections.has("chores") && <section className="wall-card wall-small-card"><div className="wall-card-heading"><span aria-hidden="true">☷</span><strong>A few things to do</strong><small>{chores.length} left</small></div>{chores.length ? chores.map(chore => <div className="wall-chore-row" key={chore.ID}><i aria-hidden="true"/><span>{chore.Name}<small>{chore.AssignedToUserID ? "Assigned" : "Anyone"}</small></span></div>) : <div className="wall-empty"><h2>All caught up</h2><p>No chores are waiting right now.</p></div>}</section>}
    </div>
    <nav className="wall-launcher" aria-label="Family apps">{sections.has("shopping") && <span><b aria-hidden="true">♧</b>Shopping</span>}<span><b aria-hidden="true">▤</b>Notes</span>{sections.has("calendar") && <span><b aria-hidden="true">▦</b>Calendar</span>}<span><b aria-hidden="true">♨</b>Recipes</span>{sections.has("meals") && <span><b aria-hidden="true">♧</b>Meal plan</span>}<span><b aria-hidden="true">✓</b>Inventory</span></nav>
  </section></main>;
}
