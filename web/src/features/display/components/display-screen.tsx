"use client";

import type { MouseEvent, RefObject } from "react";
import { BookOpen, CalendarDays, Check, CookingPot, ShoppingBag, Trash2, X, type LucideIcon } from "lucide-react";
import { ChoreCompletionDialog } from "@/features/display/components/chore-completion-dialog";
import { KioskApp } from "@/features/display/apps/kiosk-app";
import { WeatherIcon, SectionIcon } from "@/features/display/components/display-icons";
import { latestRefresh, displayDialogTitle, stationName, stationShortName, trainStatusLabel } from "@/features/display/model/display-formatters";
import type { DisplayChore, EventDetails, MealEditTarget, Snapshot } from "@/features/display/model/display-types";
import { dayKeyAt, displayGreeting, mealWindowAt, rankDisplayCards, relevantBinDate, relevantDepartures, weatherLabel } from "@/lib/display-logic";

const overviewSections = ["calendar", "meals", "chores", "weather", "departures", "trains", "bins", "shopping"] as const;
const launcherApps: { name: string; icon: LucideIcon }[] = [
  { name: "Shopping", icon: ShoppingBag }, { name: "Notes", icon: BookOpen },
  { name: "Calendar", icon: CalendarDays }, { name: "Recipes", icon: CookingPot },
  { name: "Meal plan", icon: CookingPot }, { name: "Inventory", icon: Check },
];

type DisplayScreenProps = {
  snapshot: Snapshot | null;
  clock: Date;
  loading: boolean;
  failures: string[];
  selectedApp: string;
  selectedEvent: EventDetails | null;
  selectedChore: DisplayChore | null;
  mealEditTarget: MealEditTarget | null;
  pendingChoreId: string;
  choreFeedback: string;
  token: string;
  dialogRef: RefObject<HTMLElement | null>;
  onOpenApp: (app: string, event?: MouseEvent<HTMLElement>) => void;
  onOpenEvent: (details: EventDetails, event?: MouseEvent<HTMLElement>) => void;
  onSelectChore: (chore: DisplayChore, event: MouseEvent<HTMLElement>) => void;
  onCompleteAssignedChore: (chore: DisplayChore) => void;
  onClose: () => void;
  onRefresh: () => void;
  onScratchpadSaveReady: (save: (() => Promise<void>) | null) => void;
};

export function DisplayScreen(props: DisplayScreenProps) {
  const { snapshot, clock, loading, failures, selectedApp, selectedEvent, selectedChore, mealEditTarget, pendingChoreId, choreFeedback, token, dialogRef, onOpenApp, onOpenEvent, onSelectChore, onCompleteAssignedChore, onClose, onRefresh, onScratchpadSaveReady } = props;
  const current = snapshot;
  const timezone = current?.timezone ?? "Europe/London";
  const dateKey = dayKeyAt(clock, timezone);
  const mealWindow = mealWindowAt(clock, timezone);
  const sections = new Set(current?.enabledSections ?? overviewSections);
  const events = (current?.events ?? []).filter(event => dayKeyAt(new Date(event.StartTime), timezone) === dateKey);
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
  const cardLayoutClass = (key: string) => cardOrder[key] === 0 ? "primary-first" : cardOrder[key] === 1 ? "primary-second" : "secondary-card";
  const isDark = localHour >= 20 || localHour < 6;
  const formattedDate = clock.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone });
  const formattedTime = clock.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone });


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
        <footer className="display-footer"><button aria-label={`Refresh display data${current ? `; last successful refresh ${latestRefresh(current.resourceUpdatedAt, timezone)}` : ""}`} onClick={onRefresh}>Refresh{current && <small> ({latestRefresh(current.resourceUpdatedAt, timezone)})</small>}</button></footer>
      </div>
    </section>
    {choreFeedback && <div className="display-toast" role="status">{choreFeedback}</div>}

    {selectedApp && <div className={`display-overlay${isDark ? " is-dark" : ""}`} onClick={event => { if (event.target === event.currentTarget) onClose(); }}>
      <section ref={dialogRef} className={`display-dialog${selectedApp === "Weather" ? " weather-dialog" : ""}`} role="dialog" aria-modal="true" aria-labelledby="display-dialog-title" tabIndex={-1}>
        <header className="display-dialog-heading"><div><h2 id="display-dialog-title">{displayDialogTitle(selectedApp, mealWindow.label)}</h2></div><button className="close-button" onClick={onClose} aria-label="Back to the household overview"><X size={24} strokeWidth={1.7} aria-hidden="true"/></button></header>
        <div className="display-dialog-content">
          {selectedApp === "Chore completion" && selectedChore ? <ChoreCompletionDialog chore={selectedChore} people={current?.people ?? []} token={token} onComplete={() => { void onClose(); void onRefresh(); }} /> : selectedApp === "Family event" && selectedEvent ? <div className="event-detail"><p>{new Date(selectedEvent.startTime).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</p><h3>{selectedEvent.title}</h3><p>{new Date(selectedEvent.startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{selectedEvent.endTime ? `–${new Date(selectedEvent.endTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}` : ""}</p>{selectedEvent.location && <p>{selectedEvent.location}</p>}{selectedEvent.leaveAt && <p className="event-travel">Leave at {new Date(selectedEvent.leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{selectedEvent.drivingSeconds ? ` · ${Math.round(selectedEvent.drivingSeconds / 60)} min drive` : ""}</p>}</div> : selectedApp === "launcher" ? <div className="launcher-grid">{[["Shopping", "＋"], ["Scratchpad", "✎"], ["Calendar", "▦"], ["Recipes", "♧"], ["Meal plan", "♨"], ["Inventory", "▤"]].map(([app, symbol]) => <button className="launcher-tile" key={app} onClick={event => onOpenApp(app, event)}><span aria-hidden="true">{symbol}</span>{app}</button>)}</div> : <KioskApp app={selectedApp} token={token} initialMeal={mealEditTarget} timezone={timezone} onScratchpadSaveReady={onScratchpadSaveReady} />}
          <button className="display-dialog-back" onClick={() => void onClose()}>Back to the overview</button>
        </div>
      </section>
    </div>}
  </main>;
}
