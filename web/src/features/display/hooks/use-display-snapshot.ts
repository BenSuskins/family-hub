"use client";

import { useCallback, useEffect, useRef, useState, type MutableRefObject } from "react";
import { displayTokenKey, snapshotKey, localMocksEnabled, readStorage, writeStorage, removeStorage, displayRequest, DisplayAccessRevoked } from "@/features/display/api/display-api";
import type { BinSchedule, CommuteResult, DisplayChore, DisplayMeal, DisplayPerson, DisplaySettings, Snapshot, Weather } from "@/features/display/model/display-types";
import { resourceName } from "@/features/display/model/display-formatters";
import { dayKeyAt, isDisplaySnapshot, mealWindowAt, type DisplayEvent } from "@/lib/display-logic";

type DisplaySnapshotState = {
  snapshot: Snapshot | null;
  householdName: string;
  loading: boolean;
  failures: string[];
  refresh: () => Promise<void>;
};

export function useDisplaySnapshot(token: string, clockRef: MutableRefObject<Date | null>, onAccessRevoked: () => void): DisplaySnapshotState {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [householdName, setHouseholdName] = useState("Family");
  const [loading, setLoading] = useState(true);
  const [failures, setFailures] = useState<string[]>([]);
  const snapshotRef = useRef<Snapshot | null>(null);
  const refreshInFlight = useRef(false);

  const refresh = useCallback(async () => {
    if (!token || refreshInFlight.current) return;
    refreshInFlight.current = true;
    let successfulResources: string[] = [];
    try {
      const displaySettings = await displayRequest<DisplaySettings>("/api/display/settings", token);
      if (!displaySettings.id) throw new Error("The server returned incomplete display settings.");
      const storageKey = snapshotKey(displaySettings.id);
      let previous = snapshotRef.current?.displayId === displaySettings.id ? snapshotRef.current : null;
      if (!previous) {
        const stored = readStorage(storageKey);
        if (stored) {
          try {
            const decoded: unknown = JSON.parse(stored);
            if (isDisplaySnapshot(decoded) && decoded.displayId === displaySettings.id) previous = decoded as Snapshot;
            else removeStorage(storageKey);
          } catch { removeStorage(storageKey); }
        }
      }
      if (previous) {
        snapshotRef.current = previous;
        setSnapshot(previous);
        if (previous.householdName) setHouseholdName(previous.householdName);
      }
      const homeResult = await Promise.allSettled([displayRequest<{ timezone: string; familyName?: string }>("/api/display/home-info", token)]);
      if (homeResult[0].status === "rejected" && homeResult[0].reason instanceof DisplayAccessRevoked) throw homeResult[0].reason;
      const timezone = homeResult[0].status === "fulfilled" ? homeResult[0].value.timezone : previous?.timezone ?? "Europe/London";
      const nextHouseholdName = homeResult[0].status === "fulfilled" ? homeResult[0].value.familyName?.trim() || "Family" : previous?.householdName || "Family";
      setHouseholdName(nextHouseholdName);
      if (homeResult[0].status === "fulfilled") successfulResources.push("home");
      const now = localMocksEnabled ? clockRef.current ?? new Date() : new Date();
      const mealWindow = mealWindowAt(now, timezone);
      const sections = new Set(displaySettings.enabledSections ?? []);
      const fetches: [string, Promise<unknown>][] = [];
      if (sections.has("meals")) {
        fetches.push(["calendar", displayRequest(`/api/display/calendar?view=week&date=${mealWindow.date}`, token)]);
      }
      if (sections.has("calendar")) {
        fetches.push(["dashboard-calendar", displayRequest(`/api/display/calendar?view=day&date=${dayKeyAt(now, timezone)}`, token)]);
      }
      if (sections.has("chores")) fetches.push(["dashboard-chores", displayRequest("/api/display/dashboard", token)]);
      if (sections.has("weather")) fetches.push(["weather", displayRequest("/api/display/weather", token)]);
      if (sections.has("departures")) fetches.push(["departures", displayRequest("/api/display/departures", token)]);
      if (sections.has("bins")) fetches.push(["bins", displayRequest("/api/display/bin-collections", token)]);
      if (sections.has("chores")) fetches.push(["people", displayRequest("/api/display/people", token)]);
      if (sections.has("trains")) {
        fetches.push(["commute", displayRequest("/api/display/commute", token)]);
        fetches.push(["trains", displayRequest("/api/display/trains", token)]);
      }
      const results = await Promise.allSettled(fetches.map(([, promise]) => promise));
      if (results.some(result => result.status === "rejected" && result.reason instanceof DisplayAccessRevoked)) throw new DisplayAccessRevoked();
      const values = new Map<string, unknown>();
      const nextFailures: string[] = homeResult[0].status === "rejected" ? ["Home settings"] : [];
      results.forEach((result, index) => {
        const key = fetches[index][0];
        if (result.status === "fulfilled") { values.set(key, result.value); successfulResources.push(key); }
        else nextFailures.push(resourceName(key));
      });
      const calendar = values.get("calendar") as { events?: DisplayEvent[]; chores?: DisplayChore[]; meals?: DisplayMeal[]; availability?: Record<string, string> } | undefined;
      const dashboardCalendar = values.get("dashboard-calendar") as { events?: DisplayEvent[]; availability?: Record<string, string> } | undefined;
      const dashboardChores = values.get("dashboard-chores") as { chores_due_today_list?: DisplayChore[]; chores_overdue_list?: DisplayChore[] } | undefined;
      const calendarAvailability = calendar?.availability ?? {};
      const dashboardChoresAvailable = values.has("dashboard-chores");
      const today = dayKeyAt(now, timezone);
      const resourceAvailability = { ...previous?.availability };
      for (const [key, result] of results.entries()) {
        const resource = fetches[key][0];
        resourceAvailability[resource] = result.status === "fulfilled" ? "available" : previous?.availability[resource] ? "stale" : "unavailable";
      }
      if (sections.has("calendar")) {
        resourceAvailability.events = !dashboardCalendar || dashboardCalendar.availability?.events === "unavailable"
          ? previous?.resourceUpdatedAt.events ? "stale" : "unavailable"
          : "available";
      }
      if (sections.has("meals")) {
        resourceAvailability.meals = calendarAvailability.meals === "unavailable"
          ? previous?.resourceUpdatedAt.meals ? "stale" : "unavailable"
          : calendar ? "available" : previous?.resourceUpdatedAt.meals ? "stale" : "unavailable";
      }
      if (sections.has("chores")) {
        resourceAvailability.chores = dashboardChoresAvailable
          ? "available"
          : previous?.resourceUpdatedAt.chores ? "stale" : "unavailable";
      }
      const next: Snapshot = {
        schemaVersion: 2,
        displayId: displaySettings.id,
        householdName: nextHouseholdName,
        events: dashboardCalendar && dashboardCalendar.availability?.events !== "unavailable" ? dashboardCalendar.events ?? [] : previous?.events ?? [],
        chores: dashboardChoresAvailable
          ? [...(dashboardChores?.chores_overdue_list ?? []), ...(dashboardChores?.chores_due_today_list ?? [])]
          : (previous?.chores ?? []).filter(chore => chore.Status === "overdue" || Boolean(chore.DueDate && chore.DueDate.slice(0, 10) <= today)),
        meals: calendar && calendarAvailability.meals !== "unavailable" ? calendar.meals ?? [] : previous?.meals ?? [],
        weather: values.has("weather") ? values.get("weather") as Weather : previous?.weather,
        departures: values.has("departures") ? values.get("departures") as Snapshot["departures"] : previous?.departures,
        bins: values.has("bins") ? values.get("bins") as BinSchedule : previous?.bins,
        commute: values.has("trains") ? values.get("trains") as CommuteResult : previous?.commute,
        people: values.has("people") ? values.get("people") as DisplayPerson[] : previous?.people ?? [],
        enabledSections: displaySettings.enabledSections ?? [],
        timezone,
        resourceUpdatedAt: { ...previous?.resourceUpdatedAt },
        availability: resourceAvailability,
      };
      for (const resource of successfulResources) next.resourceUpdatedAt[resource] = now.toISOString();
      if (sections.has("calendar") && resourceAvailability.events === "available") next.resourceUpdatedAt.events = now.toISOString();
      if (sections.has("meals") && resourceAvailability.meals === "available") next.resourceUpdatedAt.meals = now.toISOString();
      if (dashboardChoresAvailable) next.resourceUpdatedAt.chores = now.toISOString();
      snapshotRef.current = next;
      setSnapshot(next);
      setFailures(nextFailures);
      if ((previous || successfulResources.length > 1) && !writeStorage(storageKey, JSON.stringify(next)) && !nextFailures.length) setFailures(["This browser cannot save display data. It will refresh again when reopened."]);
    } catch (reason) {
      if (reason instanceof DisplayAccessRevoked) {
        removeStorage(displayTokenKey);
        if (snapshotRef.current?.displayId) removeStorage(snapshotKey(snapshotRef.current.displayId));
        snapshotRef.current = null;
        setSnapshot(null);
        onAccessRevoked();
        setFailures([]);
      } else {
        setFailures(["Display settings could not be reached. The display will retry."]);
      }
    } finally {
      setLoading(false);
      refreshInFlight.current = false;
    }
  }, [token, onAccessRevoked]);

  useEffect(() => {
    if (!localMocksEnabled || !token) return;
    const refreshForMockClock = () => void refresh();
    window.addEventListener("family-hub:mock-clock-changed", refreshForMockClock);
    return () => window.removeEventListener("family-hub:mock-clock-changed", refreshForMockClock);
  }, [token, refresh]);

  useEffect(() => {
    if (!token) return;
    void refresh();
    const timer = window.setInterval(() => void refresh(), 60_000);
    const reconnect = () => void refresh();
    window.addEventListener("online", reconnect);
    return () => { window.clearInterval(timer); window.removeEventListener("online", reconnect); };
  }, [token, refresh]);

  return { snapshot, householdName, loading, failures, refresh };
}
