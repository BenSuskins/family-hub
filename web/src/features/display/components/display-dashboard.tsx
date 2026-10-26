"use client";

import type { MouseEvent } from "react";
import { BookOpen, CalendarDays, Check, CookingPot, ShoppingBag, Trash2, type LucideIcon } from "lucide-react";
import { WeatherIcon, SectionIcon } from "@/features/display/components/display-icons";
import { refreshAge, stationName, stationShortName, trainStatusLabel } from "@/features/display/model/display-formatters";
import type { DisplayChore, EventDetails, KioskAppName, Snapshot } from "@/features/display/model/display-types";
import { dayKeyAt, displayGreeting, mealWindowAt, rankDisplayCards, relevantBinDate, relevantDepartures, usefulCalendarEvents, weatherLabel } from "@/lib/display-logic";

const overviewSections = ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"] as const;
const launcherApps: { name: KioskAppName; icon: LucideIcon }[] = [
  { name: "Shopping", icon: ShoppingBag }, { name: "Notes", icon: BookOpen },
  { name: "Calendar", icon: CalendarDays }, { name: "Recipes", icon: CookingPot },
  { name: "Meal plan", icon: CookingPot }, { name: "Inventory", icon: Check },
];
export type DisplayDashboardProps = {
  snapshot: Snapshot | null;
  clock: Date;
  loading: boolean;
  failures: string[];
  pendingChoreId: string;
  onOpenApp: (app: KioskAppName | "launcher", event?: MouseEvent<HTMLElement>) => void;
  onOpenEvent: (details: EventDetails, event?: MouseEvent<HTMLElement>) => void;
  onSelectChore: (chore: DisplayChore, event: MouseEvent<HTMLElement>) => void;
  onCompleteAssignedChore: (chore: DisplayChore) => void;
  onRefresh: () => void;
};

export function DisplayDashboard(props: DisplayDashboardProps) {
  const { snapshot, clock, loading, failures, pendingChoreId, onOpenApp, onOpenEvent, onSelectChore, onCompleteAssignedChore, onRefresh } = props;
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
  const firstBoard = commute?.boards[0];
  const trains = (commute?.status === "available" || commute?.status === "partial") && firstBoard
    ? firstBoard.trains.map(train => ({ board: firstBoard, train }))
    : [];
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
  const cardLayoutClass = (key: string) => {
    if (cardOrder[key] === 0) return "primary-first";
    if (cardOrder[key] === 1) return "primary-second";
    return "secondary-card";
  };
  const isDark = localHour >= 20 || localHour < 6;
  const formattedDate = clock.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone });
  const formattedTime = clock.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });
  const refreshAgeLabel = current ? refreshAge(current.resourceUpdatedAt, clock) : null;

  return <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}>
    <section className={`wall-display${isDark ? " is-dark" : ""}`} aria-label="Family Hub wall display">
      <header className="display-header"><div className="display-brand"><div><p>{formattedDate}</p><h1>{displayGreeting(localHour)}</h1></div></div><div className="display-header-right">{sections.has("weather") && weatherReady && weather?.current && <button className="weather-summary" onClick={event => onOpenApp("Weather", event)} aria-label="Open weather forecast"><WeatherIcon code={weather.current.code} size={41}/><span><strong>{Math.round(weather.current.temperature)}°</strong><small>{weather.current.summary ?? weatherLabel(weather.current.code)} · H {Math.round(weather.high ?? weather.current.temperature)}° L {Math.round(weather.low ?? weather.current.temperature)}°</small></span></button>}<time>{formattedTime}</time></div></header>
      {failures.length > 0 && <div className="display-status" role="status">{failures.join(" and ")} {current ? "could not be updated. Showing saved information where available." : "are unavailable. The display will keep trying."}</div>}
      {loading && !current && <p className="display-loading" role="status">Loading household information…</p>}
      <div className="display-grid" data-card-count={Object.keys(cardOrder).length}>
        {rankedCards.length === 0 && <section className="display-card empty-wall" aria-live="polite"><div><h2>A quiet moment at home.</h2><p>Enabled sections appear when there is something useful to show.</p></div></section>}
        {isCardVisible("departures") && departureItems.length > 0 && <section style={cardStyle("departures")} className={`display-card departure-card ${cardLayoutClass("departures")}`}><div className="display-card-heading"><span><SectionIcon name="departures"/>Time to head out</span><small className="leave-pill">{departureMinutes > 0 ? `Leave in ${departureMinutes} min` : "Time to leave"}</small></div><div className="departure-time">{new Date(departureItems[0].leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</div><h2>{departureItems[0].title}</h2><p>{[departureItems[0].subtitle, departureItems[0].destination, departureItems[0].drivingSeconds ? `${Math.round(departureItems[0].drivingSeconds / 60)} min drive` : ""].filter(Boolean).join(" · ")}</p><div className="departure-footer"><small>Starts at {new Date(departureItems[0].startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</small><button className="display-card-cta" onClick={event => { const item = current?.events.find(calendarEvent => calendarEvent.ID === departureItems[0].eventID); onOpenEvent({ title: departureItems[0].title, startTime: departureItems[0].startTime, endTime: item?.EndTime, location: item?.Location ?? departureItems[0].destination, leaveAt: departureItems[0].leaveAt, drivingSeconds: departureItems[0].drivingSeconds }, event); }}>Event details <span aria-hidden="true">→</span></button></div></section>}
        {isCardVisible("meals") && meals.length > 0 && <section style={cardStyle("meals")} className={`display-card meal-card ${cardLayoutClass("meals")}`} role="group" aria-label={`${mealWindow.label}’s meals`} tabIndex={0} onClick={event => { if (!(event.target instanceof Element) || !event.target.closest("button")) onOpenApp("Meal plan", event); }} onKeyDown={event => { if ((event.key === "Enter" || event.key === " ") && event.target === event.currentTarget) { event.preventDefault(); onOpenApp("Meal plan"); } }}><div className="display-card-heading"><span><SectionIcon name="meals"/>{mealWindow.label}’s meals</span></div><div className="meal-agenda">{(["breakfast", "lunch", "dinner"] as const).map(type => { const meal = meals.find(item => item.MealType === type); return meal && <div className={`meal-agenda-row${type === mealWindow.nextMeal ? " meal-current" : ""}`} key={`${meal.Date}-${meal.MealType}`}><span>{type}</span><strong>{meal.Name}</strong>{type === mealWindow.nextMeal && <small>Up next</small>}</div>; })}</div><div className="meal-card-footer"><button className="display-card-cta" onClick={event => { event.stopPropagation(); onOpenApp("Meal plan", event); }}>Edit meals <span aria-hidden="true">→</span></button></div></section>}
        {isCardVisible("calendar") && events.length > 0 && <section style={cardStyle("calendar")} className={`display-card calendar-card ${cardLayoutClass("calendar")}`} aria-label="Upcoming calendar events" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="calendar"/>Coming up</span><small>{events.length} {events.length === 1 ? "event" : "events"}</small></div>{events.slice(0, 3).map(item => { const relatedDeparture = departureItems.find(departure => departure.eventID === item.ID); const eventSubtitle = [relatedDeparture?.subtitle, item.Location].filter(Boolean).join(" · "); return <button className="display-event" key={item.ID} onClick={event => onOpenEvent({ title: item.Title, startTime: item.StartTime, endTime: item.EndTime, location: item.Location }, event)}><time>{item.AllDay ? "All day" : new Date(item.StartTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}</time><span className="event-line" style={{"--event-color":item.Color ?? "#718da6"} as React.CSSProperties} aria-hidden="true"/><span><strong>{item.Title}</strong>{eventSubtitle && <small>{eventSubtitle}</small>}</span></button>; })}<div className="calendar-card-footer">Shared family calendar</div></section>}
        {isCardVisible("chores") && chores.length > 0 && <section style={cardStyle("chores")} className={`display-card chores-card ${cardLayoutClass("chores")}`} aria-label="Household chores" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="chores"/>A few things to do</span><small>{chores.length} left</small></div>{chores.map(chore => { const person = chore.AssignedToUserID ? current?.people.find(item => item.id === chore.AssignedToUserID)?.name : undefined; const pending = pendingChoreId === chore.ID; return <button className="display-chore" key={chore.ID} disabled={pending} onClick={event => { if (chore.AssignedToUserID) void onCompleteAssignedChore(chore); else { onSelectChore(chore, event); } }}><span className="chore-ring" aria-hidden="true"/><span><strong>{chore.Name}</strong><small>{pending ? "Saving…" : person ?? "Anyone"}</small></span>{person && <span className="chore-avatar" aria-hidden="true">{person[0]}</span>}</button>; })}</section>}
        {isCardVisible("bins") && collections.length > 0 && <section style={cardStyle("bins")} className={`display-card bins-card ${cardLayoutClass("bins")}`} aria-label="Bin collections" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="bins"/>{collections[0].date === dateKey ? "Bins collected today" : "Bins out tonight"}</span></div>{collections.map(bin => <div className="bin-content" key={`${bin.date}-${bin.type}`}><Trash2 size={45} strokeWidth={1.7} aria-hidden="true"/><div><strong>{bin.name || bin.type}</strong><p>Collection {bin.date === dateKey ? "today" : "tomorrow"}<br/>{new Date(`${bin.date}T12:00:00`).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</p></div></div>)}<div className="bin-card-footer">Leave them out before bedtime</div></section>}
        {isCardVisible("trains") && trains.length > 0 && <section style={cardStyle("trains")} className={`display-card trains-card ${cardLayoutClass("trains")}`} aria-label="Next trains" tabIndex={0}><div className="display-card-heading"><span><SectionIcon name="trains"/>Next trains</span></div><small className="route-caption">{stationName(firstBoard!.origin.code)} → {stationName(firstBoard!.destination.code)}</small>{trains.slice(0, 3).map(({ board, train }) => <div className="train-row" key={`${board.origin.code}-${train.serviceId}`}><time>{train.expectedDeparture || train.scheduledDeparture || "—"}</time><strong>{stationShortName(board.destination.code)}</strong><small>{trainStatusLabel(train)}</small></div>)}<div className="train-card-footer">{current?.availability.trains === "available" ? "Updated just now" : "Saved train times"}</div></section>}
      </div>
      <div className="display-controls"><nav className="display-launcher" aria-label="Family apps">{launcherApps.map(({ name, icon: Icon }) => <button key={name} onClick={event => onOpenApp(name, event)}><Icon size={21} strokeWidth={1.7} aria-hidden="true" className="display-icon"/><span>{name}</span></button>)}</nav>
        <footer className="display-footer"><button aria-label={`Refresh display data${refreshAgeLabel ? `; last successful refresh ${refreshAgeLabel}` : ""}`} onClick={onRefresh}>Refresh{refreshAgeLabel && <small> ({refreshAgeLabel})</small>}</button></footer>
      </div>
    </section>
  </main>;
}
