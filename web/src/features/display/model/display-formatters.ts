import type { CommuteBoard } from "@/features/display/model/display-types";
import { dayKeyAt } from "@/lib/display-logic";

export function weekStartAt(date: Date, timezone: string) {
  const weekStart = new Date(`${dayKeyAt(date, timezone)}T12:00:00Z`);
  weekStart.setUTCDate(weekStart.getUTCDate() - ((weekStart.getUTCDay() + 6) % 7));
  return weekStart.toISOString().slice(0, 10);
}

export function hourAt(value: string, timezone: string) {
  const hour = new Intl.DateTimeFormat("en-GB", { hour: "2-digit", hourCycle: "h23", timeZone: timezone })
    .formatToParts(new Date(value)).find(part => part.type === "hour")?.value;
  return Number(hour ?? 0);
}

export function trainStatusLabel(train: CommuteBoard["trains"][number]) {
  if (train.status === "on_time") return "On time";
  if (train.status !== "delayed" || !train.expectedDeparture || !train.scheduledDeparture) {
    return train.status.replaceAll("_", " ");
  }
  const minutesAt = (time: string) => {
    const [hours, minutes] = time.split(":").map(Number);
    return hours * 60 + minutes;
  };
  const difference = minutesAt(train.expectedDeparture) - minutesAt(train.scheduledDeparture);
  const delay = difference < 0 ? difference + 24 * 60 : difference;
  return `${Math.max(1, delay)} min late`;
}

export function refreshAge(resourceUpdatedAt: Record<string, string>, now: Date) {
  const timestamps = Object.values(resourceUpdatedAt).map(Date.parse).filter(Number.isFinite);
  if (!timestamps.length) return "saved information";

  const elapsedMinutes = Math.floor(Math.max(0, now.getTime() - Math.max(...timestamps)) / 60_000);
  if (elapsedMinutes === 0) return "just now";
  if (elapsedMinutes < 60) return `${elapsedMinutes} min ago`;

  const elapsedHours = Math.floor(elapsedMinutes / 60);
  if (elapsedHours < 24) return `${elapsedHours} hr ago`;

  const elapsedDays = Math.floor(elapsedHours / 24);
  return `${elapsedDays} day${elapsedDays === 1 ? "" : "s"} ago`;
}

export function displayDialogTitle(app: string, mealLabel: string) {
  const titles: Record<string, string> = {
    Weather: "Weather forecast",
    Shopping: "Shopping list",
    Notes: "Household notes",
    Calendar: "Family calendar",
    Recipes: "Recipe book",
    Inventory: "Household inventory",
    "Family event": "Family event",
    "Chore completion": "Who completed this chore?",
  };
  if (app === "Meal plan") return `${mealLabel}’s meal plan`;
  return titles[app] ?? app;
}

export function stationName(code: string) {
  const names: Record<string, string> = { KEL: "Kelvedon", LST: "London Liverpool Street" };
  return names[code] ?? code;
}

export function stationShortName(code: string) {
  const names: Record<string, string> = { KEL: "Kelvedon", LST: "London Liverpool St." };
  return names[code] ?? code;
}

export function resourceName(key: string) {
  const names: Record<string, string> = { calendar: "Calendar", weather: "Weather", departures: "Journeys", bins: "Bin collections", people: "Household members", commute: "Commute settings", trains: "Train times" };
  return names[key] ?? key;
}
