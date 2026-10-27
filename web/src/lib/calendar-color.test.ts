import assert from "node:assert/strict";
import { describe, test } from "node:test";
import { calendarEventColor } from "@/lib/calendar-color";

describe("calendarEventColor", () => {
  test("maps every subscription color to a CSS color", () => {
    assert.equal(calendarEventColor("indigo"), "#6366F1");
    assert.equal(calendarEventColor("violet"), "#8B5CF6");
    assert.equal(calendarEventColor("rose"), "#F43F5E");
    assert.equal(calendarEventColor("red"), "#EF4444");
    assert.equal(calendarEventColor("orange"), "#F97316");
    assert.equal(calendarEventColor("amber"), "#F59E0B");
    assert.equal(calendarEventColor("emerald"), "#10B981");
    assert.equal(calendarEventColor("teal"), "#14B8A6");
    assert.equal(calendarEventColor("sky"), "#0EA5E9");
    assert.equal(calendarEventColor("blue"), "#3B82F6");
  });

  test("preserves hex event colors and safely defaults unknown values", () => {
    assert.equal(calendarEventColor("#FF0088"), "#FF0088");
    assert.equal(calendarEventColor("Rose"), "#F43F5E");
    assert.equal(calendarEventColor("unknown"), "#6366F1");
  });
});
