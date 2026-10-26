"use client";

import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { BinSchedule } from "@/features/display/model/display-types";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

export function BinsApp({ token }: KioskAppProps) {
  const { data, error } = useKioskResource<BinSchedule>("/api/display/bin-collections", token);
  if (error && !data) return <p role="alert">{error}</p>;
  if (!data) return <p role="status">Loading…</p>;
  const collections = data.collections ?? [];
  return <div className="kiosk-list">
    {collections.length === 0 ? <p>No upcoming bin collections are available.</p> : collections.map(bin => <section key={`${bin.date}-${bin.type}`}>
      <h3>{bin.name || bin.type}</h3>
      <p>{new Date(`${bin.date}T12:00:00`).toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long" })}</p>
    </section>)}
  </div>;
}
