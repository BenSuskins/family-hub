import { http, HttpResponse } from "msw";
import fixture from "../../mocks/display.json";

type Fixture = typeof fixture;
type MockState = { data: Fixture; approved: boolean; autoApprove: boolean; challengeId: string; challengeNumber: number; clockTime: string | null };

const state: MockState = {
  data: structuredClone(fixture),
  approved: false,
  autoApprove: false,
  challengeId: "",
  challengeNumber: 0,
  clockTime: null,
};
const displayToken = "local-display-token";

export function setMockAutoApproval(enabled: boolean) {
  state.autoApprove = enabled;
}

export function setMockClock(time: string | null) {
  state.clockTime = time && /^([01]\d|2[0-3]):[0-5]\d$/.test(time) ? time : null;
}

function mockNow() {
  const now = new Date();
  if (state.clockTime) {
    const [hours, minutes] = state.clockTime.split(":").map(Number);
    now.setHours(hours, minutes, 0, 0);
  }
  return now;
}

function dateKey(date: Date) {
  const parts = new Intl.DateTimeFormat("en-CA", { timeZone: "Europe/London", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(date);
  const value = (type: string) => parts.find(part => part.type === type)?.value ?? "00";
  return `${value("year")}-${value("month")}-${value("day")}`;
}

function dateAtOffset(offset: number) {
  const [year, month, day] = dateKey(mockNow()).split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, day + offset)).toISOString().slice(0, 10);
}

function resolveFixture(value: unknown): unknown {
  if (typeof value === "string") {
    if (value === "{{today}}") return dateAtOffset(0);
    if (value === "{{tomorrow}}") return dateAtOffset(1);
    if (value === "{{now}}") return mockNow().toISOString();
    const hourOffset = value.match(/^\{\{hour:(\d+)\}\}$/);
    if (hourOffset) {
      const date = mockNow();
      date.setHours(date.getHours() + Number(hourOffset[1]), 0, 0, 0);
      return date.toISOString();
    }
    const dateOffset = value.match(/^\{\{date:(-?\d+)\}\}$/);
    if (dateOffset) return dateAtOffset(Number(dateOffset[1]));
    const dateTime = value.match(/^\{\{dateAt:(-?\d+):([01]\d|2[0-3]):([0-5]\d)\}\}$/);
    if (dateTime) {
      const date = new Date(`${dateAtOffset(Number(dateTime[1]))}T12:00:00`);
      date.setHours(Number(dateTime[2]), Number(dateTime[3]), 0, 0);
      return date.toISOString();
    }
    const timeOffset = value.match(/^\{\{(today|tomorrow)At:([01]\d|2[0-3]):([0-5]\d)\}\}$/);
    if (timeOffset) {
      const date = new Date(`${dateAtOffset(timeOffset[1] === "today" ? 0 : 1)}T12:00:00`);
      date.setHours(Number(timeOffset[2]), Number(timeOffset[3]), 0, 0);
      return date.toISOString();
    }
    const minuteOffset = value.match(/^\{\{in:(\d+)\}\}$/);
    if (minuteOffset) return new Date(mockNow().getTime() + Number(minuteOffset[1]) * 60_000).toISOString();
    const clockOffset = value.match(/^\{\{clock:(\d+)\}\}$/);
    if (clockOffset) {
      return new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/London", hour: "2-digit", minute: "2-digit", hourCycle: "h23" })
        .format(new Date(mockNow().getTime() + Number(clockOffset[1]) * 60_000));
    }
    return value;
  }
  if (Array.isArray(value)) return value.map(resolveFixture);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([key, child]) => [key, resolveFixture(child)]));
  return value;
}

function calendarWeek(date: string) {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) return null;
  const selected = new Date(`${date}T12:00:00Z`);
  if (Number.isNaN(selected.getTime())) return null;
  const mondayOffset = (selected.getUTCDay() + 6) % 7;
  const start = dateAtOffsetFrom(date, -mondayOffset);
  const end = dateAtOffsetFrom(start, 7);
  return { start, end };
}

function dateAtOffsetFrom(date: string, offset: number) {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(Date.UTC(year, month - 1, day + offset)).toISOString().slice(0, 10);
}

function json(value: unknown, status = 200) {
  return HttpResponse.json(value as never, { status, headers: { "Cache-Control": "no-store" } });
}

function isDisplayRequest(request: Request) {
  return request.headers.get("authorization") === `Bearer ${displayToken}`;
}

export const handlers = [
  http.get("*/api/local-mocks/clock", () => json({ now: mockNow().toISOString(), time: state.clockTime })),
  http.post("*/api/local-mocks/clock", async ({ request }) => {
    const body = await request.json() as { time?: string | null };
    if (body.time !== null && (typeof body.time !== "string" || !/^([01]\d|2[0-3]):[0-5]\d$/.test(body.time))) {
      return json({ error: "Use a 24-hour time such as 07:30." }, 400);
    }
    setMockClock(body.time ?? null);
    return json({ now: mockNow().toISOString(), time: state.clockTime });
  }),
  http.post("*/api/local-mocks/reset", () => {
    state.data = structuredClone(fixture);
    state.approved = false;
    state.autoApprove = false;
    state.challengeId = "";
    state.challengeNumber = 0;
    state.clockTime = null;
    return json({ reset: true });
  }),
  http.post("*/api/local-mocks/pairing/approve", () => {
    if (!state.challengeId) return json({ error: "Create a display pairing code first." }, 409);
    state.approved = true;
    return json({ approved: true });
  }),
  http.post("*/api/display-pairing/challenges", () => {
    state.challengeNumber += 1;
    state.challengeId = `local-challenge-${state.challengeNumber}`;
    state.approved = state.autoApprove;
    return json({ challengeId: state.challengeId, pairingCode: "123456", expiresAt: new Date(Date.now() + 300_000).toISOString() });
  }),
  http.post("*/api/display-pairing/challenges/:challengeId/claim", async ({ params, request }) => {
    const body = await request.json() as { pairingCode?: string };
    if (params.challengeId !== state.challengeId || body.pairingCode !== "123456") return json({ error: "Pairing code is invalid." }, 400);
    if (!state.approved) return json({ error: "Waiting for local approval." }, 409);
    return json({ token: displayToken });
  }),
  http.get("*/api/display/:resource", ({ params, request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const data = state.data;
    const resource = String(params.resource ?? "");
    const calendar = resolveFixture(data.calendar) as Fixture["calendar"];
    const query = new URL(request.url).searchParams;
    const date = query.get("date") ?? dateAtOffset(0);
    if (resource === "calendar" && query.get("view") === "week") {
      const prototypeEventIds = new Set(["event-school-dropoff", "event-library", "event-football", "event-thursday-swim", "event-pizza"]);
      calendar.events = calendar.events.filter(event => prototypeEventIds.has(event.ID));
    } else if (resource === "calendar" && query.get("view") === "dashboard") {
      calendar.events = calendar.events.filter(event => event.StartTime.slice(0, 10) === dateAtOffset(0));
    }
    const week = calendarWeek(date);
    if (!week) return json({ error: "Invalid date format, use YYYY-MM-DD." }, 400);
    const inWeek = (value: string) => value >= week.start && value < week.end;
    calendar.events = calendar.events.filter(event => inWeek(event.StartTime.slice(0, 10)));
    calendar.chores = calendar.chores.filter(chore => !chore.DueDate || inWeek(chore.DueDate));
    calendar.meals = calendar.meals.filter(meal => inWeek(meal.Date));
    const resources: Record<string, unknown> = {
      settings: data.settings,
      "home-info": data.home,
      people: data.people,
      calendar,
      chores: resolveFixture(data.calendar.chores),
      meals: resolveFixture(data.calendar.meals),
      weather: resolveFixture(data.weather),
      departures: resolveFixture(data.departures),
      "bin-collections": resolveFixture(data.bins),
      commute: resolveFixture(data.commute),
      trains: resolveFixture(data.commute),
      shopping: data.shopping,
      inventory: data.inventory,
      recipes: data.recipes,
      scratchpad: data.scratchpad,
    };
    return resource in resources ? json(resources[resource]) : json({ error: "Mock route not found." }, 404);
  }),
  http.get("*/api/display/recipes/:recipeId", ({ params, request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const recipes = state.data.recipeDetails as Record<string, unknown>;
    const recipeId = String(params.recipeId ?? "");
    return recipeId in recipes ? json(recipes[recipeId]) : json({ error: "Recipe not found." }, 404);
  }),
  http.put("*/api/display/scratchpad", async ({ request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    state.data.scratchpad = await request.json() as typeof state.data.scratchpad;
    return json(state.data.scratchpad);
  }),
  http.post("*/api/display/meals", async ({ request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const body = await request.json() as { date: string; mealType: string; name: string };
    const meals = state.data.calendar.meals as { Date: string; MealType: string; Name: string }[];
    const templateDate = body.date === dateAtOffset(0) ? "{{today}}" : "{{tomorrow}}";
    const index = meals.findIndex(meal => (meal.Date === body.date || meal.Date === templateDate) && meal.MealType === body.mealType);
    const saved = { Date: body.date, MealType: body.mealType, Name: body.name };
    if (index >= 0) meals[index] = saved;
    else meals.push(saved);
    return json(saved);
  }),
  http.post("*/api/display/chores/:choreId/complete", async ({ params, request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const body = await request.json() as { userID?: string };
    const chore = state.data.calendar.chores.find(item => item.ID === params.choreId);
    if (!chore || !body.userID) return json({ error: "Chore or household member not found." }, 404);
    chore.Status = "completed";
    chore.AssignedToUserID = body.userID;
    return new HttpResponse(null, { status: 204 });
  }),
  http.patch("*/api/display/shopping/items/:itemId", async ({ params, request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const item = state.data.shopping.find(value => value.id === params.itemId);
    if (!item) return json({ error: "Shopping item not found." }, 404);
    const body = await request.json() as { purchased?: boolean };
    item.purchased = Boolean(body.purchased);
    item.version += 1;
    return json(item);
  }),
  http.post("*/api/display/shopping/items", async ({ request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const body = await request.json() as { name?: string };
    if (!body.name?.trim()) return json({ error: "Item name is required." }, 400);
    const item = { id: `shopping-local-${Date.now()}`, name: body.name.trim(), purchased: false, version: 1 };
    state.data.shopping.push(item);
    return json(item, 201);
  }),
  http.put("*/api/display/inventory/items/:itemId/stock", async ({ params, request }) => {
    if (!isDisplayRequest(request)) return json({ error: "Display token is invalid." }, 401);
    const item = state.data.inventory.flatMap(area => area.Items).find(value => value.ID === params.itemId);
    if (!item) return json({ error: "Inventory item not found." }, 404);
    const body = await request.json() as { quantityDelta?: number; level?: number };
    if (item.TrackingMode === "level" && body.level !== undefined) item.Level = Math.max(0, Math.min(100, body.level));
    if (item.TrackingMode !== "level" && body.quantityDelta !== undefined) item.Quantity = Math.max(0, item.Quantity + body.quantityDelta);
    return json(item);
  }),
];
