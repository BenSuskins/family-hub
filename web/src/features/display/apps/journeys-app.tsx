"use client";

import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { DisplayDeparture } from "@/lib/display-logic";
import { relevantDepartures } from "@/lib/display-logic";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type JourneyData = { status: string; items: DisplayDeparture[] };

export function JourneysApp({ token }: KioskAppProps) {
  const { data, error } = useKioskResource<JourneyData>("/api/display/departures", token);
  if (error && !data) return <p role="alert">{error}</p>;
  if (!data) return <p role="status">Loading…</p>;
  const items = relevantDepartures(data.items ?? [], new Date());
  return <div className="kiosk-list">
    {data.status !== "available" && <p role="status">{data.status === "unavailable" ? "Journey information is unavailable." : "No journeys are scheduled soon."}</p>}
    {items.map(item => <section key={item.eventID}>
      <h3>{item.title}</h3><p>{item.destination}</p>
      <p>Leave at {new Date(item.leaveAt).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" })}</p>
      <small>Event starts {new Date(item.startTime).toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit" })}{item.drivingSeconds ? ` · ${Math.round(item.drivingSeconds / 60)} min drive` : ""}</small>
    </section>)}
  </div>;
}
