"use client";

import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { CommuteResult } from "@/features/display/model/display-types";
import { trainStatusLabel } from "@/features/display/model/display-formatters";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

export function TrainsApp({ token }: KioskAppProps) {
  const { data, error } = useKioskResource<CommuteResult>("/api/display/trains", token);
  if (error && !data) return <p role="alert">{error}</p>;
  if (!data) return <p role="status">Loading…</p>;
  return <div className="kiosk-list">
    {data.status === "setup" ? <p>Choose stations and a shared schedule in display settings.</p>
      : data.status === "outside_schedule" ? <p>No trains are shown outside the shared commute schedule.</p>
        : data.status === "unavailable" ? <p role="status">Train times are unavailable. The display will retry.</p>
          : data.boards.map(board => <section key={board.origin.code + board.destination.code}>
              <h3>{board.origin.code} to {board.destination.code}</h3>
              {board.trains.map(train => <div className="train-detail" key={train.serviceId}>
                <strong>{train.expectedDeparture || train.scheduledDeparture || "Time unavailable"}</strong>
                <span>{trainStatusLabel(train)}</span><small>{train.platform ? `Platform ${train.platform}` : train.operator || ""}</small>
              </div>)}
            </section>)}
  </div>;
}
