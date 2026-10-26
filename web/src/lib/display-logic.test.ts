import assert from "node:assert/strict";
import { test } from "node:test";
import {
  dayKeyAt,
  displayGreeting,
  displayStoragePrefix,
  householdDocumentTitle,
  isDisplaySnapshot,
  mealWindowAt,
  rankDisplayCards,
  nextDayKey,
  relevantBinDate,
  relevantDepartures,
  usefulCalendarEvents,
  weatherLabel,
} from "./display-logic";

test("meal preview uses the selected household timezone and switches at 20:30", () => {
  assert.deepEqual(mealWindowAt(new Date("2026-10-26T20:29:00Z"), "Europe/London"), {
    date: "2026-10-26", label: "Today", nextMeal: "dinner",
  });
  assert.deepEqual(mealWindowAt(new Date("2026-10-26T20:30:00Z"), "Europe/London"), {
    date: "2026-10-27", label: "Tomorrow", nextMeal: "breakfast",
  });
  assert.equal(mealWindowAt(new Date("2026-03-29T19:29:00Z"), "Europe/London").label, "Today");
  assert.equal(mealWindowAt(new Date("2026-03-29T19:30:00Z"), "Europe/London").label, "Tomorrow");
});

test("overview prioritizes near departures, dinner, and evening bin reminders", () => {
  assert.equal(displayGreeting(8, true), "Time to head out soon.");
  assert.equal(displayGreeting(8, false), "Good morning, everyone.");
  assert.equal(displayGreeting(14, false), "A little room in the day.");
  assert.equal(displayGreeting(19, false), "Home for the evening.");
  const ranked = rankDisplayCards({
    hour: 18,
    departureSoon: true,
    dinnerNext: true,
    hasCalendarEvents: true,
    hasOpenChores: true,
    binsRelevant: true,
    binsTomorrow: true,
    trainsAvailable: true,
    weatherAvailable: true,
  });
  assert.deepEqual(ranked.slice(0, 4), ["departures", "meals", "bins", "calendar"]);
});

test("calendar date keys remain local over daylight saving transitions", () => {
  assert.equal(dayKeyAt(new Date("2026-03-29T00:30:00Z"), "Europe/London"), "2026-03-29");
  assert.equal(dayKeyAt(new Date("2026-10-26T00:30:00Z"), "Europe/London"), "2026-10-26");
  assert.equal(nextDayKey("2026-12-31"), "2027-01-01");
});

test("calendar selection keeps ongoing events and excludes ended events", () => {
  const now = new Date("2026-10-26T12:00:00Z");
  const events = usefulCalendarEvents([
    { ID: "ended", Title: "Ended", StartTime: "2026-10-26T08:00:00Z", EndTime: "2026-10-26T09:00:00Z" },
    { ID: "ongoing", Title: "Ongoing", StartTime: "2026-10-26T11:00:00Z", EndTime: "2026-10-26T13:00:00Z" },
    { ID: "all-day", Title: "All day", StartTime: "2026-10-26T00:00:00Z", AllDay: true },
  ], now, "Europe/London");
  assert.deepEqual(events.map(event => event.ID), ["all-day", "ongoing"]);
});

test("departure cards exclude past and distant journeys and sort by leave time", () => {
  const result = relevantDepartures([
    { eventID: "late", title: "Later", destination: "", startTime: "", leaveAt: "2026-10-26T15:00:00Z", drivingSeconds: 0 },
    { eventID: "past", title: "Past", destination: "", startTime: "", leaveAt: "2026-10-26T09:00:00Z", drivingSeconds: 0 },
    { eventID: "soon", title: "Soon", destination: "", startTime: "", leaveAt: "2026-10-26T12:30:00Z", drivingSeconds: 0 },
  ], new Date("2026-10-26T12:00:00Z"));
  assert.deepEqual(result.map(item => item.eventID), ["soon", "late"]);
});

test("bin relevance includes today and tomorrow only", () => {
  assert.equal(relevantBinDate("2026-10-27", "2026-10-26"), true);
  assert.equal(relevantBinDate("2026-10-28", "2026-10-26"), false);
});

test("weather conditions use the provider code", () => {
  assert.equal(weatherLabel(0), "Clear");
  assert.equal(weatherLabel(61), "Rain");
  assert.equal(weatherLabel(95), "Thunderstorms");
});

test("cache keys include the API origin and old cache schemas are rejected", () => {
  assert.match(displayStoragePrefix("https://family.example/api/"), /family\.example/);
  assert.notEqual(displayStoragePrefix("https://one.example"), displayStoragePrefix("https://two.example"));
  assert.equal(isDisplaySnapshot({ schemaVersion: 1, displayId: "display" }), false);
  assert.equal(isDisplaySnapshot({ schemaVersion: 2, displayId: "display", timezone: "Europe/London", resourceUpdatedAt: {}, availability: {}, events: [], chores: [], meals: [], people: [], enabledSections: [] }), true);
  assert.equal(isDisplaySnapshot({ schemaVersion: 2, displayId: "display", timezone: "Europe/London", resourceUpdatedAt: {}, availability: {}, events: "corrupt", chores: [], meals: [], people: [], enabledSections: [] }), false);
  assert.match(displayStoragePrefix(""), /same-origin/);
  assert.equal(householdDocumentTitle(" Willow "), "Willow household");
  assert.equal(householdDocumentTitle(undefined), "Family household");
});
