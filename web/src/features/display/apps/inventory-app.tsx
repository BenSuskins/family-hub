"use client";

import { useState } from "react";
import { apiBase } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type InventoryItem = { ID: string; Name: string; TrackingMode: string; Quantity: number; Level: number; Unit: string };
type InventoryArea = { ID: string; Name: string; Items: InventoryItem[] };

export function InventoryApp({ token }: KioskAppProps) {
  const { data: areas, error, setError, reload } = useKioskResource<InventoryArea[]>("/api/display/inventory", token);
  const [busy, setBusy] = useState("");

  async function changeStock(item: InventoryItem, direction: number) {
    setBusy(item.ID);
    setError("");
    const change = item.TrackingMode === "level"
      ? { level: Math.max(0, Math.min(100, item.Level + direction * 25)) }
      : { quantityDelta: direction };
    try {
      const response = await fetch(`${apiBase}/api/display/inventory/items/${encodeURIComponent(item.ID)}/stock`, {
        method: "PUT",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify(change),
      });
      if (!response.ok) throw new Error();
      await reload();
    } catch {
      setError("Inventory could not be updated. Please try again.");
    } finally { setBusy(""); }
  }

  if (error && !areas) return <p role="alert">{error}</p>;
  if (!areas) return <p role="status">Loading…</p>;
  return <div className="kiosk-list">
    <p className="dialog-label">Browse by area.</p>
    {error && <p role="alert">{error}</p>}
    {areas.map(area => <section key={area.ID}>
      <h2>{area.Name}</h2>
      {area.Items.map(item => <div className="stock-row" key={item.ID}>
        <span><strong>{item.Name}</strong><small>{item.TrackingMode === "level" ? `${item.Level}%` : `${item.Quantity} ${item.Unit}`}</small></span>
        <div>
          <button aria-label={`Decrease ${item.Name}`} disabled={busy === item.ID || (item.TrackingMode === "level" ? item.Level <= 0 : item.Quantity <= 0)} onClick={() => void changeStock(item, -1)}>−</button>
          <button aria-label={`Increase ${item.Name}`} disabled={busy === item.ID || (item.TrackingMode === "level" ? item.Level >= 100 : false)} onClick={() => void changeStock(item, 1)}>＋</button>
        </div>
      </div>)}
    </section>)}
  </div>;
}
