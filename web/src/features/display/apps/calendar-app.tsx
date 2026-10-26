"use client";

import { Fragment, useState } from "react";
import { Typography } from "@/components/typography";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import { dayKeyAt } from "@/lib/display-logic";
import {
  hourAt,
  weekStartAt,
} from "@/features/display/model/display-formatters";
import type { DisplayChore } from "@/features/display/model/display-types";
import type { DisplayEvent } from "@/lib/display-logic";

type CalendarData = { events: DisplayEvent[]; chores: DisplayChore[] };
const dayMilliseconds = 86_400_000;
const timeSlots = [8, 10, 12, 14, 16, 18];

export function CalendarApp({
  token,
  timezone,
  clock,
}: {
  token: string;
  timezone: string;
  clock: Date;
}) {
  const [week, setWeek] = useState(() => weekStartAt(clock, timezone));
  const { data: calendar, error } = useKioskResource<CalendarData>(
    `/api/display/calendar?view=week&date=${week}`,
    token,
  );
  if (error && !calendar)
    return (
      <Typography as="p" variant="body" role="alert">
        {error}
      </Typography>
    );
  if (!calendar)
    return (
      <Typography as="p" variant="body" role="status">
        Loading…
      </Typography>
    );

  const dates = Array.from(
    { length: 7 },
    (_, index) =>
      new Date(Date.parse(`${week}T12:00:00Z`) + index * dayMilliseconds),
  );
  const range = new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "long",
    timeZone: "UTC",
  });
  const weekLabel = `${range.format(dates[0])} – ${range.format(dates[6])}`;
  const dateKey = (date: Date) => date.toISOString().slice(0, 10);
  const eventsForSlot = (date: Date, hour: number) =>
    calendar.events.filter((event) => {
      const eventDate = event.AllDay
        ? event.StartTime.slice(0, 10)
        : dayKeyAt(new Date(event.StartTime), timezone);
      if (eventDate !== dateKey(date)) return false;
      return event.AllDay
        ? hour === 8
        : hourAt(event.StartTime, timezone) >= hour &&
            hourAt(event.StartTime, timezone) < hour + 2;
    });
  const moveWeek = (offset: number) =>
    setWeek(
      weekStartAt(
        new Date(
          Date.parse(`${week}T12:00:00Z`) + offset * 7 * dayMilliseconds,
        ),
        "UTC",
      ),
    );
  const today = weekStartAt(clock, timezone);

  return (
    <div>
      <div className="calendar-toolbar">
        <button
          className="calendar-nav"
          aria-label="Previous week"
          onClick={() => moveWeek(-1)}
        >
          ‹
        </button>
        <Typography as="strong" variant="caption">
          {weekLabel}
        </Typography>
        <button
          className="calendar-nav"
          aria-label="Next week"
          onClick={() => moveWeek(1)}
        >
          ›
        </button>
        <button
          className="calendar-today"
          onClick={() => setWeek(today)}
          disabled={week === today}
        >
          <Typography as="span" variant="caption">
            Today
          </Typography>
        </button>
      </div>
      {calendar.events.length === 0 && (
        <Typography as="p" variant="body" className="calendar-empty">
          No events this week.
        </Typography>
      )}
      <div className="week-scroll">
        <div className="week-grid" aria-label={`Family calendar, ${weekLabel}`}>
          <div className="week-corner" />
          {dates.map((date) => (
            <div className="week-day" key={dateKey(date)}>
              <Typography as="strong" variant="caption">
                {new Intl.DateTimeFormat("en-GB", {
                  weekday: "short",
                  timeZone: "UTC",
                }).format(date)}
              </Typography>
              <Typography as="span" variant="caption">
                {date.getUTCDate()}
              </Typography>
            </div>
          ))}
          {timeSlots.map((hour) => (
            <Fragment key={hour}>
              <div className="week-hour">
                {String(hour).padStart(2, "0")}:00
              </div>
              {dates.map((date) => (
                <div className="week-cell" key={`${dateKey(date)}-${hour}`}>
                  {eventsForSlot(date, hour).map((event) => (
                    <div
                      className="week-event"
                      key={event.ID}
                      style={
                        {
                          "--event-color": event.Color ?? "#718da6",
                        } as React.CSSProperties
                      }
                    >
                      <Typography as="small" variant="caption">
                        {event.AllDay
                          ? "All day"
                          : new Date(event.StartTime).toLocaleTimeString(
                              "en-GB",
                              {
                                hour: "2-digit",
                                minute: "2-digit",
                                timeZone: timezone,
                              },
                            )}
                      </Typography>
                      <Typography as="span" variant="body">
                        {event.Title}
                      </Typography>
                    </div>
                  ))}
                </div>
              ))}
            </Fragment>
          ))}
        </div>
      </div>
    </div>
  );
}
