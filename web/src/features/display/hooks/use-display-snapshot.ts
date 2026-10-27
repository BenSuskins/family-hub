"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type MutableRefObject,
} from "react";
import {
  displayTokenKey,
  snapshotKey,
  localMocksEnabled,
  readStorage,
  writeStorage,
  removeStorage,
  displayRequest,
  DisplayAccessRevoked,
} from "@/features/display/api/display-api";
import type {
  BinSchedule,
  CommuteResult,
  DisplayChore,
  DisplayMeal,
  DisplayPerson,
  DisplaySettings,
  Snapshot,
  Weather,
} from "@/features/display/model/display-types";
import { resourceName } from "@/features/display/model/display-formatters";
import {
  addDaysKey,
  dayKeyAt,
  isDisplaySnapshot,
  mondayKey,
  rollingDateKeys,
  type DisplayEvent,
} from "@/lib/display-logic";

type DisplaySnapshotState = {
  snapshot: Snapshot | null;
  householdName: string;
  loading: boolean;
  failures: string[];
  refresh: () => Promise<void>;
};

export function useDisplaySnapshot(
  token: string,
  clockRef: MutableRefObject<Date | null>,
  onAccessRevoked: () => void,
  onRefreshComplete?: () => void,
): DisplaySnapshotState {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [householdName, setHouseholdName] = useState("Family");
  const [loading, setLoading] = useState(true);
  const [failures, setFailures] = useState<string[]>([]);
  const snapshotRef = useRef<Snapshot | null>(null);
  const refreshInFlight = useRef(false);

  const refresh = useCallback(async () => {
    if (!token || refreshInFlight.current) return;
    refreshInFlight.current = true;
    const successfulResources: string[] = [];
    try {
      const [settingsResult, homeResult] = await Promise.allSettled([
        displayRequest<DisplaySettings>("/api/display/settings", token),
        displayRequest<{ timezone: string; familyName?: string }>(
          "/api/display/home-info",
          token,
        ),
      ]);
      if (settingsResult.status === "rejected") throw settingsResult.reason;
      const displaySettings = settingsResult.value;
      if (!displaySettings.id)
        throw new Error("The server returned incomplete display settings.");
      const storageKey = snapshotKey(displaySettings.id);
      let previous =
        snapshotRef.current?.displayId === displaySettings.id
          ? snapshotRef.current
          : null;
      if (!previous) {
        const stored = readStorage(storageKey);
        if (stored) {
          try {
            const decoded: unknown = JSON.parse(stored);
            if (
              isDisplaySnapshot(decoded) &&
              decoded.displayId === displaySettings.id
            )
              previous = decoded as Snapshot;
            else removeStorage(storageKey);
          } catch {
            removeStorage(storageKey);
          }
        }
      }
      if (previous) {
        snapshotRef.current = previous;
        setSnapshot(previous);
        if (previous.householdName) setHouseholdName(previous.householdName);
      }
      if (
        homeResult.status === "rejected" &&
        homeResult.reason instanceof DisplayAccessRevoked
      )
        throw homeResult.reason;
      const timezone =
        homeResult.status === "fulfilled"
          ? homeResult.value.timezone
          : (previous?.timezone ?? "Europe/London");
      const nextHouseholdName =
        homeResult.status === "fulfilled"
          ? homeResult.value.familyName?.trim() || "Family"
          : previous?.householdName || "Family";
      setHouseholdName(nextHouseholdName);
      if (homeResult.status === "fulfilled") successfulResources.push("home");
      const now = localMocksEnabled
        ? (clockRef.current ?? new Date())
        : new Date();
      const sections = new Set(displaySettings.enabledSections ?? []);
      const today = dayKeyAt(now, timezone);
      const weekDates = rollingDateKeys(today, 7);
      const calendarWeeks = sections.has("calendar")
        ? [...new Set([mondayKey(today), mondayKey(weekDates.at(-1) ?? today)])]
        : [];
      const fetches: [string, Promise<unknown>][] = [];
      for (const week of calendarWeeks) {
        fetches.push([
          `calendar-week-${week}`,
          displayRequest(`/api/display/calendar?view=week&date=${week}`, token),
        ]);
      }
      if (sections.has("meals"))
        fetches.push([
          "meals",
          displayRequest(`/api/display/meals?week=${today}`, token),
        ]);
      if (sections.has("chores"))
        fetches.push([
          "dashboard-chores",
          displayRequest("/api/display/dashboard", token),
        ]);
      if (sections.has("weather"))
        fetches.push([
          "weather",
          displayRequest("/api/display/weather", token),
        ]);
      if (sections.has("departures"))
        fetches.push([
          "departures",
          displayRequest("/api/display/departures", token),
        ]);
      if (sections.has("bins"))
        fetches.push([
          "bins",
          displayRequest("/api/display/bin-collections", token),
        ]);
      if (sections.has("chores"))
        fetches.push(["people", displayRequest("/api/display/people", token)]);
      if (sections.has("trains")) {
        fetches.push([
          "commute",
          displayRequest("/api/display/commute", token),
        ]);
        fetches.push(["trains", displayRequest("/api/display/trains", token)]);
      }
      const results = await Promise.allSettled(
        fetches.map(([, promise]) => promise),
      );
      if (
        results.some(
          (result) =>
            result.status === "rejected" &&
            result.reason instanceof DisplayAccessRevoked,
        )
      )
        throw new DisplayAccessRevoked();
      const values = new Map<string, unknown>();
      const nextFailures: string[] =
        homeResult.status === "rejected" ? ["Home settings"] : [];
      results.forEach((result, index) => {
        const key = fetches[index][0];
        const resource = key.startsWith("calendar-week-") ? "calendar" : key;
        if (result.status === "fulfilled") {
          values.set(key, result.value);
          successfulResources.push(resource);
        } else nextFailures.push(resourceName(resource));
      });
      const dashboardChores = values.get("dashboard-chores") as
        | {
            chores_due_today_list?: DisplayChore[];
            chores_overdue_list?: DisplayChore[];
          }
        | undefined;
      const dashboardChoresAvailable = values.has("dashboard-chores");
      const calendarResponses = calendarWeeks.map(
        (week) =>
          values.get(`calendar-week-${week}`) as
            | { events?: DisplayEvent[]; availability?: Record<string, string> }
            | undefined,
      );
      const calendarEventsAvailable =
        calendarWeeks.length > 0 &&
        calendarResponses.every(
          (response) =>
            response && response.availability?.events !== "unavailable",
        );
      const weekEventIds = new Set<string>();
      const weekEvents = calendarEventsAvailable
        ? calendarResponses
            .flatMap((response) => response?.events ?? [])
            .filter((event) => {
              const eventDate = event.AllDay
                ? event.StartTime.slice(0, 10)
                : dayKeyAt(new Date(event.StartTime), timezone);
              const isInWindow =
                eventDate >= today && eventDate <= addDaysKey(today, 6);
              if (!isInWindow || weekEventIds.has(event.ID)) return false;
              weekEventIds.add(event.ID);
              return true;
            })
        : (previous?.events ?? []);
      const resourceAvailability = { ...previous?.availability };
      for (const [key, result] of results.entries()) {
        const resourceKey = fetches[key][0];
        const resource = resourceKey.startsWith("calendar-week-")
          ? "calendar"
          : resourceKey;
        if (result.status === "fulfilled")
          resourceAvailability[resource] = "available";
        else if (previous?.availability[resource])
          resourceAvailability[resource] = "stale";
        else resourceAvailability[resource] = "unavailable";
      }
      if (calendarWeeks.length) {
        if (calendarEventsAvailable) {
          resourceAvailability.events = "available";
        } else if (previous?.resourceUpdatedAt.events) {
          resourceAvailability.events = "stale";
        } else {
          resourceAvailability.events = "unavailable";
        }
      }
      if (sections.has("meals")) {
        if (!values.has("meals")) {
          resourceAvailability.meals = previous?.resourceUpdatedAt.meals
            ? "stale"
            : "unavailable";
        } else {
          resourceAvailability.meals = "available";
        }
      }
      if (sections.has("chores")) {
        if (dashboardChoresAvailable) {
          resourceAvailability.chores = "available";
        } else if (previous?.resourceUpdatedAt.chores) {
          resourceAvailability.chores = "stale";
        } else {
          resourceAvailability.chores = "unavailable";
        }
      }
      const next: Snapshot = {
        schemaVersion: 2,
        displayId: displaySettings.id,
        householdName: nextHouseholdName,
        events: weekEvents,
        chores: dashboardChoresAvailable
          ? [
              ...(dashboardChores?.chores_overdue_list ?? []),
              ...(dashboardChores?.chores_due_today_list ?? []),
            ]
          : (previous?.chores ?? []).filter(
              (chore) =>
                chore.Status === "overdue" ||
                Boolean(chore.DueDate && chore.DueDate.slice(0, 10) <= today),
            ),
        meals: values.has("meals")
          ? (values.get("meals") as DisplayMeal[])
          : (previous?.meals ?? []),
        weather: values.has("weather")
          ? (values.get("weather") as Weather)
          : previous?.weather,
        departures: values.has("departures")
          ? (values.get("departures") as Snapshot["departures"])
          : previous?.departures,
        bins: values.has("bins")
          ? (values.get("bins") as BinSchedule)
          : previous?.bins,
        commute: values.has("trains")
          ? (values.get("trains") as CommuteResult)
          : previous?.commute,
        people: values.has("people")
          ? (values.get("people") as DisplayPerson[])
          : (previous?.people ?? []),
        enabledSections: displaySettings.enabledSections ?? [],
        timezone,
        resourceUpdatedAt: { ...previous?.resourceUpdatedAt },
        availability: resourceAvailability,
      };
      for (const resource of successfulResources)
        next.resourceUpdatedAt[resource] = now.toISOString();
      if (calendarEventsAvailable)
        next.resourceUpdatedAt.events = now.toISOString();
      if (sections.has("meals") && resourceAvailability.meals === "available")
        next.resourceUpdatedAt.meals = now.toISOString();
      if (dashboardChoresAvailable)
        next.resourceUpdatedAt.chores = now.toISOString();
      snapshotRef.current = next;
      setSnapshot(next);
      setFailures([...new Set(nextFailures)]);
      if (
        (previous || successfulResources.length > 1) &&
        !writeStorage(storageKey, JSON.stringify(next)) &&
        !nextFailures.length
      )
        setFailures([
          "This browser cannot save display data. It will refresh again when reopened.",
        ]);
    } catch (reason) {
      if (reason instanceof DisplayAccessRevoked) {
        removeStorage(displayTokenKey);
        if (snapshotRef.current?.displayId)
          removeStorage(snapshotKey(snapshotRef.current.displayId));
        snapshotRef.current = null;
        setSnapshot(null);
        onAccessRevoked();
        setFailures([]);
      } else {
        setFailures([
          "Display settings could not be reached. The display will retry.",
        ]);
      }
    } finally {
      setLoading(false);
      refreshInFlight.current = false;
      onRefreshComplete?.();
    }
  }, [token, clockRef, onAccessRevoked, onRefreshComplete]);

  useEffect(() => {
    if (!localMocksEnabled || !token) return;
    const refreshForMockClock = () => void refresh();
    window.addEventListener(
      "family-hub:mock-clock-changed",
      refreshForMockClock,
    );
    return () =>
      window.removeEventListener(
        "family-hub:mock-clock-changed",
        refreshForMockClock,
      );
  }, [token, refresh]);

  useEffect(() => {
    if (!token) return;
    void refresh();
    const timer = window.setInterval(() => void refresh(), 60_000);
    const reconnect = () => void refresh();
    window.addEventListener("online", reconnect);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener("online", reconnect);
    };
  }, [token, refresh]);

  return { snapshot, householdName, loading, failures, refresh };
}
