"use client";

import type { MouseEvent } from "react";
import { Typography } from "@/components/typography";
import {
  BookOpen,
  CalendarDays,
  CookingPot,
  Package,
  ShoppingBag,
  Trash2,
  type LucideIcon,
} from "lucide-react";
import {
  WeatherIcon,
  SectionIcon,
} from "@/features/display/components/display-icons";
import {
  refreshAge,
  stationName,
  stationShortName,
  trainStatusLabel,
} from "@/features/display/model/display-formatters";
import type {
  DisplayChore,
  DisplayEvent,
  DisplayInventoryArea,
  DisplayShoppingItem,
  EventDetails,
  KioskAppName,
  Snapshot,
} from "@/features/display/model/display-types";
import {
  addDaysKey,
  dayKeyAt,
  displayGreeting,
  mealWindowAt,
  usefulCalendarEvents,
  rankDisplayCards,
  relevantBinDate,
  relevantDepartures,
  weatherLabel,
} from "@/lib/display-logic";

const overviewSections = [
  "calendar",
  "meals",
  "chores",
  "weather",
  "departures",
  "trains",
  "bins",
  "shopping",
  "inventory",
] as const;
function eventDateKey(event: DisplayEvent, timezone: string): string {
  if (event.AllDay) return event.StartTime.slice(0, 10);
  return dayKeyAt(new Date(event.StartTime), timezone);
}
const launcherApps: { name: KioskAppName; icon: LucideIcon }[] = [
  { name: "Shopping", icon: ShoppingBag },
  { name: "Notes", icon: BookOpen },
  { name: "Calendar", icon: CalendarDays },
  { name: "Recipes", icon: CookingPot },
  { name: "Meal plan", icon: CookingPot },
  { name: "Inventory", icon: Package },
];
export type DisplayDashboardProps = {
  snapshot: Snapshot | null;
  shoppingItems: DisplayShoppingItem[] | null;
  inventoryAreas: DisplayInventoryArea[] | null;
  clock: Date;
  loading: boolean;
  failures: string[];
  pendingChoreId: string;
  onOpenApp: (
    app: KioskAppName | "launcher",
    event?: MouseEvent<HTMLElement>,
  ) => void;
  onOpenEvent: (details: EventDetails, event?: MouseEvent<HTMLElement>) => void;
  onSelectChore: (chore: DisplayChore, event: MouseEvent<HTMLElement>) => void;
  onCompleteAssignedChore: (chore: DisplayChore) => void;
  onToggleShopping: (item: DisplayShoppingItem) => void;
  pendingShoppingId: string;
  onRefresh: () => void;
};

export function DisplayDashboard(props: DisplayDashboardProps) {
  const {
    snapshot,
    shoppingItems,
    inventoryAreas,
    clock,
    loading,
    failures,
    pendingChoreId,
    onToggleShopping,
    pendingShoppingId,
    onOpenApp,
    onOpenEvent,
    onSelectChore,
    onCompleteAssignedChore,
    onRefresh,
  } = props;
  const current = snapshot;
  const timezone = current?.timezone ?? "Europe/London";
  const dateKey = dayKeyAt(clock, timezone);
  const finalDayKey = addDaysKey(dateKey, 6);
  const mealWindow = mealWindowAt(clock, timezone);
  const sections = new Set(current?.enabledSections ?? overviewSections);
  const weekEvents = usefulCalendarEvents(
    current?.events ?? [],
    clock,
    timezone,
  ).filter((event) => {
    const eventDate = eventDateKey(event, timezone);
    return eventDate >= dateKey && eventDate <= finalDayKey;
  });
  const todayEvents = weekEvents.filter((event) => {
    const eventDate = eventDateKey(event, timezone);
    return eventDate === dateKey;
  });
  const events = todayEvents;
  const chores = (current?.chores ?? [])
    .filter((item) => item.Status === "pending" || item.Status === "overdue")
    .sort((first, second) =>
      (first.DueDate ?? "9999").localeCompare(second.DueDate ?? "9999"),
    );
  const meals = (current?.meals ?? []).filter(
    (item) => item.Date === mealWindow.date,
  );
  const weekMeals = (current?.meals ?? [])
    .filter((item) => item.Date >= dateKey && item.Date <= finalDayKey)
    .sort(
      (first, second) =>
        first.Date.localeCompare(second.Date) ||
        first.MealType.localeCompare(second.MealType),
    );
  const openShoppingItems = (shoppingItems ?? []).filter(
    (item) => !item.purchased,
  );
  const lowInventoryItems = (inventoryAreas ?? []).flatMap((area) =>
    area.Items.filter((item) =>
      item.TrackingMode === "level"
        ? item.Level <= item.LowAt
        : item.Quantity <= item.LowAt,
    ).map((item) => ({ ...item, areaName: area.Name })),
  );
  const departureItems = relevantDepartures(
    current?.departures?.items ?? [],
    clock,
  );
  const collections = (current?.bins?.collections ?? []).filter((item) =>
    relevantBinDate(item.date, dateKey),
  );
  const weather = current?.weather;
  const weatherReady =
    weather?.status === "available" || weather?.status === "ok";
  const commute = current?.commute;
  const firstBoard = commute?.boards[0];
  const trains =
    (commute?.status === "available" || commute?.status === "partial") &&
    firstBoard
      ? firstBoard.trains.map((train) => ({ board: firstBoard, train }))
      : [];
  const departureMinutes = departureItems[0]
    ? Math.round(
        (Date.parse(departureItems[0].leaveAt) - clock.getTime()) / 60_000,
      )
    : Infinity;
  const localHour = Number(
    new Intl.DateTimeFormat("en-GB", {
      timeZone: timezone,
      hour: "2-digit",
      hourCycle: "h23",
    }).format(clock),
  );
  const availableCards = new Set([
    ...(sections.has("calendar") && events.length ? ["calendar"] : []),
    ...(sections.has("calendar") && weekEvents.length ? ["week-calendar"] : []),
    ...(sections.has("meals") && meals.length ? ["meals"] : []),
    ...(sections.has("meals") && weekMeals.length ? ["week-meals"] : []),
    ...(sections.has("chores") && chores.length ? ["chores"] : []),
    ...(sections.has("departures") && departureItems.length
      ? ["departures"]
      : []),
    ...(sections.has("bins") && collections.length ? ["bins"] : []),
    ...(sections.has("trains") && trains.length ? ["trains"] : []),
    ...(sections.has("shopping") && openShoppingItems.length
      ? ["shopping"]
      : []),
    ...(sections.has("inventory") && lowInventoryItems.length
      ? ["inventory"]
      : []),
  ]);
  const rankedCards = rankDisplayCards({
    hour: localHour,
    departureSoon: departureMinutes <= 45,
    dinnerNext:
      mealWindow.label === "Today" && mealWindow.nextMeal === "dinner",
    hasCalendarEvents: events.length > 0,
    hasWeekCalendarEvents: weekEvents.length > 0,
    hasWeekMeals: weekMeals.length > 0,
    hasOpenChores: chores.length > 0,
    binsRelevant: collections.length > 0,
    binsTomorrow: collections.some((item) => item.date !== dateKey),
    trainsAvailable: trains.length > 0,
    weatherAvailable: weatherReady && Boolean(weather?.current),
    hasShoppingItems: openShoppingItems.length > 0,
    hasLowInventory: lowInventoryItems.length > 0,
  })
    .filter((key) => availableCards.has(key))
    .slice(0, 5);
  const cardOrder = Object.fromEntries(
    rankedCards.map((key, index) => [key, index]),
  );
  const cardSections: Record<string, string> = {
    "week-calendar": "calendar",
    "week-meals": "meals",
  };
  const isCardVisible = (key: string) =>
    sections.has(cardSections[key] ?? key) && cardOrder[key] !== undefined;
  const cardStyle = (key: string) => ({ order: cardOrder[key] });
  const cardLayoutClass = (key: string) => {
    if (cardOrder[key] === 0) return "primary-first";
    if (cardOrder[key] === 1) return "primary-second";
    return "secondary-card";
  };
  const isDark = localHour >= 20 || localHour < 6;
  const formattedDate = clock.toLocaleDateString("en-GB", {
    weekday: "long",
    day: "numeric",
    month: "long",
    timeZone: timezone,
  });
  const formattedTime = clock.toLocaleTimeString("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    timeZone: timezone,
  });
  const refreshAgeLabel = current
    ? refreshAge(current.resourceUpdatedAt, clock)
    : null;

  return (
    <main className={`wall-display-shell${isDark ? " is-dark" : ""}`}>
      <section
        className={`wall-display${isDark ? " is-dark" : ""}`}
        aria-label="Family Hub wall display"
      >
        <header className="display-header">
          <div className="display-brand">
            <div>
              <Typography as="p" variant="date">
                {formattedDate}
              </Typography>
              <Typography as="h1" variant="greeting">
                {displayGreeting(localHour)}
              </Typography>
            </div>
          </div>
          <div className="display-header-right">
            {sections.has("weather") && weatherReady && weather?.current && (
              <button
                className="weather-summary"
                onClick={(event) => onOpenApp("Weather", event)}
                aria-label="Open weather forecast"
              >
                <WeatherIcon code={weather.current.code} size={41} />
                <span>
                  <Typography as="strong" variant="cardTitle">
                    {Math.round(weather.current.temperature)}°
                  </Typography>
                  <Typography as="small" variant="cardHint">
                    {weather.current.summary ??
                      weatherLabel(weather.current.code)}{" "}
                    · H{" "}
                    {Math.round(weather.high ?? weather.current.temperature)}° L{" "}
                    {Math.round(weather.low ?? weather.current.temperature)}°
                  </Typography>
                </span>
              </button>
            )}
            <Typography as="time" variant="clock">
              {formattedTime}
            </Typography>
          </div>
        </header>
        {failures.length > 0 && (
          <Typography
            as="div"
            variant="body"
            className="display-status"
            role="status"
          >
            {failures.join(" and ")}{" "}
            {current
              ? "could not be updated. Showing saved information where available."
              : "are unavailable. The display will keep trying."}
          </Typography>
        )}
        {loading && !current && (
          <Typography
            as="p"
            variant="body"
            className="display-loading"
            role="status"
          >
            Loading household information…
          </Typography>
        )}
        <div
          className="display-grid"
          data-card-count={Object.keys(cardOrder).length}
        >
          {rankedCards.length === 0 && (
            <section className="display-card empty-wall" aria-live="polite">
              <div>
                <Typography as="h2" variant="cardTitle">
                  A quiet moment at home.
                </Typography>
                <Typography as="p" variant="body">
                  Enabled sections appear when there is something useful to
                  show.
                </Typography>
              </div>
            </section>
          )}
          {isCardVisible("departures") && departureItems.length > 0 && (
            <section
              style={cardStyle("departures")}
              className={`display-card departure-card ${cardLayoutClass("departures")}`}
              aria-label="Time to head out"
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="departures" />
                  <Typography as="span" variant="cardTitle">
                    Time to head out
                  </Typography>
                </span>
                <Typography
                  as="small"
                  variant="cardHint"
                  className="leave-pill"
                >
                  {departureMinutes > 0
                    ? `Leave in ${departureMinutes} min`
                    : "Time to leave"}
                </Typography>
              </div>
              <Typography
                as="div"
                variant="departureTime"
                className="departure-time"
              >
                {new Date(departureItems[0].leaveAt).toLocaleTimeString(
                  "en-GB",
                  { hour: "2-digit", minute: "2-digit", timeZone: timezone },
                )}
              </Typography>
              <Typography as="h2" variant="departureTitle">
                {departureItems[0].title}
              </Typography>
              <Typography as="p" variant="subtitle">
                {[
                  departureItems[0].subtitle,
                  departureItems[0].destination,
                  departureItems[0].drivingSeconds
                    ? `${Math.round(departureItems[0].drivingSeconds / 60)} min drive`
                    : "",
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </Typography>
              <div className="departure-footer">
                <Typography as="small" variant="caption">
                  Starts at{" "}
                  {new Date(departureItems[0].startTime).toLocaleTimeString(
                    "en-GB",
                    { hour: "2-digit", minute: "2-digit", timeZone: timezone },
                  )}
                </Typography>
                <button
                  className="display-card-cta"
                  onClick={(event) => {
                    const item = current?.events.find(
                      (calendarEvent) =>
                        calendarEvent.ID === departureItems[0].eventID,
                    );
                    onOpenEvent(
                      {
                        title: departureItems[0].title,
                        startTime: departureItems[0].startTime,
                        endTime: item?.EndTime,
                        location:
                          item?.Location ?? departureItems[0].destination,
                        leaveAt: departureItems[0].leaveAt,
                        drivingSeconds: departureItems[0].drivingSeconds,
                      },
                      event,
                    );
                  }}
                >
                  <Typography as="span" variant="caption">
                    Event details
                  </Typography>{" "}
                  <span aria-hidden="true">→</span>
                </button>
              </div>
            </section>
          )}
          {isCardVisible("meals") && meals.length > 0 && (
            <section
              style={cardStyle("meals")}
              className={`display-card meal-card ${cardLayoutClass("meals")}`}
              role="group"
              aria-label={`${mealWindow.label}’s meals`}
              tabIndex={0}
              onClick={(event) => {
                if (
                  !(event.target instanceof Element) ||
                  !event.target.closest("button")
                )
                  onOpenApp("Meal plan", event);
              }}
              onKeyDown={(event) => {
                if (
                  (event.key === "Enter" || event.key === " ") &&
                  event.target === event.currentTarget
                ) {
                  event.preventDefault();
                  onOpenApp("Meal plan");
                }
              }}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="meals" />
                  <Typography as="span" variant="cardTitle">
                    {mealWindow.label}’s meals
                  </Typography>
                </span>
              </div>
              <div className="meal-agenda">
                {(["breakfast", "lunch", "dinner"] as const).map((type) => {
                  const meal = meals.find((item) => item.MealType === type);
                  return (
                    meal && (
                      <div
                        className={`meal-agenda-row${type === mealWindow.nextMeal ? " meal-current" : ""}`}
                        key={`${meal.Date}-${meal.MealType}`}
                      >
                        <Typography as="span" variant="caption">
                          {type}
                        </Typography>
                        <Typography as="strong" variant="caption">
                          {meal.Name}
                        </Typography>
                        {type === mealWindow.nextMeal && (
                          <Typography as="small" variant="cardHint">
                            Up next
                          </Typography>
                        )}
                      </div>
                    )
                  );
                })}
              </div>
              <div className="meal-card-footer">
                <button
                  className="display-card-cta"
                  onClick={(event) => {
                    event.stopPropagation();
                    onOpenApp("Meal plan", event);
                  }}
                >
                  <Typography as="span" variant="caption">
                    Edit meals
                  </Typography>{" "}
                  <span aria-hidden="true">→</span>
                </button>
              </div>
            </section>
          )}
          {isCardVisible("calendar") && events.length > 0 && (
            <section
              style={cardStyle("calendar")}
              className={`display-card calendar-card ${cardLayoutClass("calendar")}`}
              aria-label="Today’s calendar events"
              tabIndex={0}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="calendar" />
                  <Typography as="span" variant="cardTitle">
                    Coming up
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {events.length} {events.length === 1 ? "event" : "events"}
                </Typography>
              </div>
              {events.slice(0, 3).map((item) => {
                const relatedDeparture = departureItems.find(
                  (departure) => departure.eventID === item.ID,
                );
                const eventSubtitle = [
                  relatedDeparture?.subtitle,
                  item.Location,
                ]
                  .filter(Boolean)
                  .join(" · ");
                return (
                  <button
                    className="display-event"
                    key={item.ID}
                    onClick={(event) =>
                      onOpenEvent(
                        {
                          title: item.Title,
                          startTime: item.StartTime,
                          endTime: item.EndTime,
                          location: item.Location,
                        },
                        event,
                      )
                    }
                  >
                    <time>
                      <Typography as="span" variant="displayEventTime">
                        {item.AllDay
                          ? "All day"
                          : new Date(item.StartTime).toLocaleTimeString(
                              "en-GB",
                              {
                                hour: "2-digit",
                                minute: "2-digit",
                                timeZone: timezone,
                              },
                            )}
                      </Typography>
                    </time>
                    <span
                      className="event-line"
                      style={
                        {
                          "--event-color": item.Color ?? "#718da6",
                        } as React.CSSProperties
                      }
                      aria-hidden="true"
                    />
                    <span>
                      <Typography as="strong" variant="displayEventTitle">
                        {item.Title}
                      </Typography>
                      {eventSubtitle && (
                        <Typography as="small" variant="displayEventMeta">
                          {eventSubtitle}
                        </Typography>
                      )}
                    </span>
                  </button>
                );
              })}
              <div className="calendar-card-footer">Shared family calendar</div>
            </section>
          )}
          {isCardVisible("week-calendar") && weekEvents.length > 0 && (
            <section
              style={cardStyle("week-calendar")}
              className={`display-card calendar-card week-calendar-card ${cardLayoutClass("week-calendar")}`}
              aria-label="Calendar for the next seven days"
              tabIndex={0}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="calendar" />
                  <Typography as="span" variant="cardTitle">
                    Next seven days
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {weekEvents.length}{" "}
                  {weekEvents.length === 1 ? "event" : "events"}
                </Typography>
              </div>
              {weekEvents.map((item) => {
                const eventDay = item.AllDay
                  ? item.StartTime.slice(0, 10)
                  : dayKeyAt(new Date(item.StartTime), timezone);
                return (
                  <button
                    className="display-event display-week-event"
                    key={item.ID}
                    onClick={(event) =>
                      onOpenEvent(
                        {
                          title: item.Title,
                          startTime: item.StartTime,
                          endTime: item.EndTime,
                          location: item.Location,
                        },
                        event,
                      )
                    }
                  >
                    <time>
                      <Typography as="span" variant="displayEventDate">
                        {new Date(`${eventDay}T12:00:00Z`).toLocaleDateString(
                          "en-GB",
                          {
                            weekday: "short",
                            day: "numeric",
                            month: "short",
                            timeZone: "UTC",
                          },
                        )}
                      </Typography>
                      <Typography as="small" variant="displayEventTime">
                        {item.AllDay
                          ? "All day"
                          : new Date(item.StartTime).toLocaleTimeString(
                              "en-GB",
                              {
                                hour: "2-digit",
                                minute: "2-digit",
                                timeZone: timezone,
                              },
                            )}
                      </Typography>
                    </time>
                    <span
                      className="event-line"
                      style={
                        {
                          "--event-color": item.Color ?? "#718da6",
                        } as React.CSSProperties
                      }
                      aria-hidden="true"
                    />
                    <span>
                      <Typography as="strong" variant="displayEventTitle">
                        {item.Title}
                      </Typography>
                      {item.Location && (
                        <Typography as="small" variant="displayEventMeta">
                          {item.Location}
                        </Typography>
                      )}
                    </span>
                  </button>
                );
              })}
              <div className="calendar-card-footer">Shared family calendar</div>
            </section>
          )}
          {isCardVisible("week-meals") && weekMeals.length > 0 && (
            <section
              style={cardStyle("week-meals")}
              className={`display-card meal-card week-meal-card ${cardLayoutClass("week-meals")}`}
              aria-label="Meals for the next seven days"
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="meals" />
                  <Typography as="span" variant="cardTitle">
                    Meals this week
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {weekMeals.length} planned
                </Typography>
              </div>
              <div className="week-meal-list">
                {weekMeals.map((meal) => (
                  <div
                    className="week-meal-row"
                    key={`${meal.Date}-${meal.MealType}`}
                  >
                    <Typography as="span" variant="caption">
                      {new Date(`${meal.Date}T12:00:00Z`).toLocaleDateString(
                        "en-GB",
                        {
                          weekday: "short",
                          day: "numeric",
                          month: "short",
                          timeZone: "UTC",
                        },
                      )}
                    </Typography>
                    <Typography as="small" variant="cardHint">
                      {meal.MealType}
                    </Typography>
                    <Typography as="strong" variant="caption">
                      {meal.Name}
                    </Typography>
                  </div>
                ))}
              </div>
              <div className="meal-card-footer">
                <button
                  className="display-card-cta"
                  onClick={() => onOpenApp("Meal plan")}
                >
                  <Typography as="span" variant="caption">
                    Open meal plan
                  </Typography>
                  <span aria-hidden="true">→</span>
                </button>
              </div>
            </section>
          )}
          {isCardVisible("shopping") && openShoppingItems.length > 0 && (
            <section
              style={cardStyle("shopping")}
              className={`display-card shopping-card ${cardLayoutClass("shopping")}`}
              aria-label="Shopping list"
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="shopping" />
                  <Typography as="span" variant="cardTitle">
                    Shopping list
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {openShoppingItems.length} to pick up
                </Typography>
              </div>
              {openShoppingItems.map((item) => (
                <button
                  className="display-shopping-row"
                  key={item.id}
                  disabled={pendingShoppingId === item.id}
                  aria-label={`Mark purchased: ${item.name}`}
                  onClick={() => onToggleShopping(item)}
                >
                  <span className="check-ring" aria-hidden="true" />
                  <Typography as="span" variant="caption">
                    {item.name}
                  </Typography>
                  {pendingShoppingId === item.id && (
                    <Typography as="small" variant="cardHint">
                      Saving…
                    </Typography>
                  )}
                </button>
              ))}
              <div className="meal-card-footer">
                <button
                  className="display-card-cta"
                  onClick={() => onOpenApp("Shopping")}
                >
                  <Typography as="span" variant="caption">
                    Open shopping
                  </Typography>
                  <span aria-hidden="true">→</span>
                </button>
              </div>
            </section>
          )}
          {isCardVisible("inventory") && lowInventoryItems.length > 0 && (
            <section
              style={cardStyle("inventory")}
              className={`display-card inventory-card ${cardLayoutClass("inventory")}`}
              aria-label="Low inventory"
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="inventory" />
                  <Typography as="span" variant="cardTitle">
                    Getting low
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {lowInventoryItems.length}{" "}
                  {lowInventoryItems.length === 1 ? "item" : "items"}
                </Typography>
              </div>
              {lowInventoryItems.map((item) => (
                <button
                  className="display-inventory-row"
                  key={item.ID}
                  onClick={() => onOpenApp("Inventory")}
                  aria-label={`Open Inventory for ${item.Name}`}
                >
                  <span>
                    <Typography as="strong" variant="caption">
                      {item.Name}
                    </Typography>
                    <Typography as="small" variant="cardHint">
                      {item.areaName}
                    </Typography>
                  </span>
                  <Typography as="small" variant="cardHint">
                    {item.TrackingMode === "level"
                      ? `${item.Level}%`
                      : `${item.Quantity} ${item.Unit}`}
                  </Typography>
                </button>
              ))}
            </section>
          )}
          {isCardVisible("chores") && chores.length > 0 && (
            <section
              style={cardStyle("chores")}
              className={`display-card chores-card ${cardLayoutClass("chores")}`}
              aria-label="Household chores"
              tabIndex={0}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="chores" />
                  <Typography as="span" variant="cardTitle">
                    A few things to do
                  </Typography>
                </span>
                <Typography as="small" variant="cardHint">
                  {chores.length} left
                </Typography>
              </div>
              {chores.map((chore) => {
                const person = chore.AssignedToUserID
                  ? current?.people.find(
                      (item) => item.id === chore.AssignedToUserID,
                    )?.name
                  : undefined;
                const pending = pendingChoreId === chore.ID;
                return (
                  <button
                    className="display-chore"
                    key={chore.ID}
                    disabled={pending}
                    onClick={(event) => {
                      if (chore.AssignedToUserID)
                        void onCompleteAssignedChore(chore);
                      else {
                        onSelectChore(chore, event);
                      }
                    }}
                  >
                    <span className="chore-ring" aria-hidden="true" />
                    <span>
                      <Typography as="strong" variant="caption">
                        {chore.Name}
                      </Typography>
                      <Typography as="small" variant="cardHint">
                        {pending ? "Saving…" : (person ?? "Anyone")}
                      </Typography>
                    </span>
                    {person && (
                      <span className="chore-avatar" aria-hidden="true">
                        {person[0]}
                      </span>
                    )}
                  </button>
                );
              })}
            </section>
          )}
          {isCardVisible("bins") && collections.length > 0 && (
            <section
              style={cardStyle("bins")}
              className={`display-card bins-card ${cardLayoutClass("bins")}`}
              aria-label="Bin collections"
              tabIndex={0}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="bins" />
                  <Typography as="span" variant="cardTitle">
                    {collections[0].date === dateKey
                      ? "Bins collected today"
                      : "Bins out tonight"}
                  </Typography>
                </span>
              </div>
              {collections.map((bin) => (
                <div className="bin-content" key={`${bin.date}-${bin.type}`}>
                  <Trash2 size={45} strokeWidth={1.7} aria-hidden="true" />
                  <div>
                    <Typography as="strong" variant="cardTitle">
                      {bin.name || bin.type}
                    </Typography>
                    <Typography as="p" variant="body">
                      Collection {bin.date === dateKey ? "today" : "tomorrow"}
                      <br />
                      {new Date(`${bin.date}T12:00:00`).toLocaleDateString(
                        "en-GB",
                        {
                          weekday: "long",
                          day: "numeric",
                          month: "long",
                          timeZone: timezone,
                        },
                      )}
                    </Typography>
                  </div>
                </div>
              ))}
              <Typography
                as="div"
                variant="cardHint"
                className="bin-card-footer"
              >
                Leave them out before bedtime
              </Typography>
            </section>
          )}
          {isCardVisible("trains") && trains.length > 0 && (
            <section
              style={cardStyle("trains")}
              className={`display-card trains-card ${cardLayoutClass("trains")}`}
              aria-label="Next trains"
              tabIndex={0}
            >
              <div className="display-card-heading">
                <span>
                  <SectionIcon name="trains" />
                  <Typography as="span" variant="cardTitle">
                    Next trains
                  </Typography>
                </span>
              </div>
              <Typography
                as="small"
                variant="cardHint"
                className="route-caption"
              >
                {stationName(firstBoard!.origin.code)} →{" "}
                {stationName(firstBoard!.destination.code)}
              </Typography>
              {trains.slice(0, 3).map(({ board, train }) => (
                <div
                  className="train-row"
                  key={`${board.origin.code}-${train.serviceId}`}
                >
                  <Typography as="time" variant="caption">
                    {train.expectedDeparture || train.scheduledDeparture || "—"}
                  </Typography>
                  <Typography as="strong" variant="caption">
                    {stationShortName(board.destination.code)}
                  </Typography>
                  <Typography as="small" variant="cardHint">
                    {trainStatusLabel(train)}
                  </Typography>
                </div>
              ))}
              <Typography
                as="div"
                variant="cardHint"
                className="train-card-footer"
              >
                {current?.availability.trains === "available"
                  ? "Updated just now"
                  : "Saved train times"}
              </Typography>
            </section>
          )}
        </div>
        <div className="display-controls">
          <nav className="display-launcher" aria-label="Family apps">
            {launcherApps.map(({ name, icon: Icon }) => (
              <button key={name} onClick={(event) => onOpenApp(name, event)}>
                <Icon
                  size={21}
                  strokeWidth={1.7}
                  aria-hidden="true"
                  className="display-icon"
                />
                <Typography as="span" variant="caption">
                  {name === "Shopping" && shoppingItems
                    ? `${name} (${shoppingItems.filter((item) => !item.purchased).length})`
                    : name}
                </Typography>
              </button>
            ))}
          </nav>
          <footer className="display-footer">
            <button
              aria-label={`Refresh display data${refreshAgeLabel ? `; last successful refresh ${refreshAgeLabel}` : ""}`}
              onClick={onRefresh}
            >
              <Typography as="span" variant="caption">
                Refresh
              </Typography>
              {refreshAgeLabel && (
                <Typography as="small" variant="cardHint">
                  {" "}
                  ({refreshAgeLabel})
                </Typography>
              )}
            </button>
          </footer>
        </div>
      </section>
    </main>
  );
}
