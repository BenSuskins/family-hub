export type MealType = "breakfast" | "lunch" | "dinner";

export type MealWindow = {
  date: string;
  label: "Today" | "Tomorrow";
  nextMeal: MealType;
};

export type DisplayCardContext = {
  hour: number;
  departureSoon: boolean;
  hasDeparture: boolean;
  dinnerNext: boolean;
  hasCalendarEvents: boolean;
  hasWeekCalendarEvents: boolean;
  hasWeekMeals: boolean;
  hasOpenChores: boolean;
  binsRelevant: boolean;
  binsTomorrow: boolean;
  trainsAvailable: boolean;
  weatherAvailable: boolean;
  hasShoppingItems: boolean;
  hasLowInventory: boolean;
};

export type DisplayEvent = {
  ID: string;
  Title: string;
  StartTime: string;
  EndTime?: string | null;
  AllDay?: boolean;
  Location?: string;
  Color?: string;
};

export type DisplayDeparture = {
  eventID: string;
  title: string;
  destination: string;
  subtitle?: string;
  startTime: string;
  leaveAt: string;
  drivingSeconds: number;
};

export function dayKeyAt(date: Date, timezone: string) {
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);
  const value = (type: string) =>
    parts.find((part) => part.type === type)?.value ?? "00";
  return `${value("year")}-${value("month")}-${value("day")}`;
}

export function nextDayKey(key: string) {
  return addDaysKey(key, 1);
}

export function addDaysKey(key: string, days: number) {
  const [year, month, day] = key.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1, day + days));
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, "0")}-${String(date.getUTCDate()).padStart(2, "0")}`;
}

export function rollingDateKeys(startDate: string, dayCount: number) {
  return Array.from({ length: dayCount }, (_, index) =>
    addDaysKey(startDate, index),
  );
}

export function mondayKey(dateKey: string) {
  const weekday = new Date(`${dateKey}T12:00:00Z`).getUTCDay();
  return addDaysKey(dateKey, -((weekday + 6) % 7));
}

export function previousDayKey(key: string) {
  const [year, month, day] = key.split("-").map(Number);
  const previous = new Date(Date.UTC(year, month - 1, day - 1));
  return `${previous.getUTCFullYear()}-${String(previous.getUTCMonth() + 1).padStart(2, "0")}-${String(previous.getUTCDate()).padStart(2, "0")}`;
}

export function mealWindowAt(date: Date, timezone: string): MealWindow {
  const parts = new Intl.DateTimeFormat("en-GB", {
    timeZone: timezone,
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date);
  const hour = Number(parts.find((part) => part.type === "hour")?.value ?? "0");
  const minute = Number(
    parts.find((part) => part.type === "minute")?.value ?? "0",
  );
  const minutes = hour * 60 + minute;
  const today = dayKeyAt(date, timezone);
  const tomorrow = minutes >= 20 * 60;
  let nextMeal: MealType;
  if (tomorrow || minutes < 10 * 60) nextMeal = "breakfast";
  else if (minutes < 15 * 60) nextMeal = "lunch";
  else nextMeal = "dinner";
  return {
    date: tomorrow ? nextDayKey(today) : today,
    label: tomorrow ? "Tomorrow" : "Today",
    nextMeal,
  };
}

export function weatherLabel(code: number) {
  if (code === 0) return "Clear";
  if ([1, 2].includes(code)) return "Partly cloudy";
  if (code === 3) return "Cloudy";
  if ([45, 48].includes(code)) return "Fog";
  if ([51, 53, 55, 56, 57].includes(code)) return "Drizzle";
  if ([61, 63, 65, 66, 67, 80, 81, 82].includes(code)) return "Rain";
  if ([71, 73, 75, 77, 85, 86].includes(code)) return "Snow";
  if ([95, 96, 99].includes(code)) return "Thunderstorms";
  return "Forecast unavailable";
}

export function usefulCalendarEvents(
  events: DisplayEvent[],
  now: Date,
  timezone: string,
) {
  return events
    .filter((event) => {
      const start = new Date(event.StartTime);
      const end = event.EndTime ? new Date(event.EndTime) : start;
      return (
        event.AllDay ||
        end >= now ||
        (dayKeyAt(start, timezone) === dayKeyAt(now, timezone) && start >= now)
      );
    })
    .sort((first, second) => first.StartTime.localeCompare(second.StartTime));
}

export function relevantDepartures(
  departures: DisplayDeparture[],
  now: Date,
  minutesAhead = 12 * 60,
) {
  const earliestRelevantLeave = now.getTime() - 10 * 60 * 1000;
  const latestRelevantLeave = now.getTime() + minutesAhead * 60 * 1000;
  return departures
    .filter((item) => {
      const leaveAt = Date.parse(item.leaveAt);
      const startTime = Date.parse(item.startTime);
      return (
        Number.isFinite(leaveAt) &&
        Number.isFinite(startTime) &&
        leaveAt >= earliestRelevantLeave &&
        leaveAt <= latestRelevantLeave &&
        startTime > now.getTime()
      );
    })
    .sort(
      (first, second) => Date.parse(first.leaveAt) - Date.parse(second.leaveAt),
    );
}

export function relevantBinDate(collectionDate: string, today: string) {
  return collectionDate === today || collectionDate === nextDayKey(today);
}

export function displayStoragePrefix(apiBase: string) {
  const base = apiBase ? new URL(apiBase) : null;
  const scope = base
    ? base.origin + base.pathname.replace(/\/$/, "")
    : "same-origin";
  return `family-hub.display.v2.${encodeURIComponent(scope)}`;
}

export function householdDocumentTitle(name: string | undefined) {
  return `${name?.trim() || "Family"} household`;
}

export function displayGreeting(hour: number) {
  if (hour < 12) return "Good morning, everyone.";
  if (hour < 17) return "A little room in the day.";
  return "Home for the evening.";
}

export function rankDisplayCards(context: DisplayCardContext) {
  let choreScore = 0;
  if (context.hasOpenChores) choreScore = context.hour >= 16 ? 74 : 48;
  let trainScore = 0;
  if (context.hour >= 7 && context.hour < 10) trainScore = 88;
  else if (context.hour >= 16 && context.hour < 19) trainScore = 45;
  let departureScore = 0;
  if (context.departureSoon) departureScore = 110;
  else if (context.hasDeparture) departureScore = 42;
  const candidates = [
    { key: "calendar", score: context.hasCalendarEvents ? 80 : 0 },
    { key: "meals", score: context.hour >= 16 ? 90 : 65 },
    { key: "week-calendar", score: context.hasWeekCalendarEvents ? 72 : 0 },
    { key: "week-meals", score: context.hasWeekMeals ? 62 : 0 },
    { key: "chores", score: choreScore },
    { key: "departures", score: departureScore },
    { key: "bins", score: context.binsRelevant && context.hour >= 17 ? 85 : 0 },
    { key: "trains", score: context.trainsAvailable ? trainScore : 0 },
    { key: "shopping", score: context.hasShoppingItems ? 70 : 0 },
    { key: "inventory", score: context.hasLowInventory ? 76 : 0 },
  ];
  return candidates
    .filter((candidate) => candidate.score > 0)
    .sort((first, second) => second.score - first.score)
    .map((candidate) => candidate.key);
}

export function isDisplaySnapshot(
  value: unknown,
): value is { schemaVersion: 2; displayId: string; [key: string]: unknown } {
  if (!value || typeof value !== "object") return false;
  const snapshot = value as Record<string, unknown>;
  const object = (candidate: unknown) =>
    candidate !== null &&
    typeof candidate === "object" &&
    !Array.isArray(candidate);
  return (
    snapshot.schemaVersion === 2 &&
    typeof snapshot.displayId === "string" &&
    typeof snapshot.timezone === "string" &&
    Array.isArray(snapshot.events) &&
    Array.isArray(snapshot.chores) &&
    Array.isArray(snapshot.meals) &&
    Array.isArray(snapshot.people) &&
    Array.isArray(snapshot.enabledSections) &&
    object(snapshot.resourceUpdatedAt) &&
    object(snapshot.availability) &&
    (snapshot.weather === undefined || object(snapshot.weather)) &&
    (snapshot.departures === undefined ||
      (object(snapshot.departures) &&
        Array.isArray(
          (snapshot.departures as Record<string, unknown>).items,
        ))) &&
    (snapshot.bins === undefined ||
      (object(snapshot.bins) &&
        Array.isArray(
          (snapshot.bins as Record<string, unknown>).collections,
        ))) &&
    (snapshot.commute === undefined ||
      (object(snapshot.commute) &&
        Array.isArray((snapshot.commute as Record<string, unknown>).boards)))
  );
}
