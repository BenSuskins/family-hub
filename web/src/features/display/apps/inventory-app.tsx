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
  return <div className="kiosk-list inventory-list">
    <p className="dialog-label">Browse by area.</p>
    {error && <p role="alert">{error}</p>}
    {areas.map(area => <section className="inventory-area" key={area.ID}>
      <h3 className="area-heading">{area.Name}</h3>
      {area.Items.map(item => {
        const isLow = item.TrackingMode === "level" ? item.Level <= 25 : item.Quantity <= 1;
        const quantity = item.TrackingMode === "level" ? `${item.Level}%` : item.Quantity;
        const unitLabel = item.Unit ? `${item.Unit} of ` : "";
        return <div className="inventory-row" key={item.ID}>
          <span>{item.Name}<small>{isLow ? "Getting low" : "In the cupboard"}</small></span>
          <span className="inventory-step">
            <button aria-label={`Use one ${unitLabel}${item.Name}`} disabled={busy === item.ID || (item.TrackingMode === "level" ? item.Level <= 0 : item.Quantity <= 0)} onClick={() => void changeStock(item, -1)}>−</button>
            <strong>{quantity}</strong>
            <button aria-label={`Add one ${unitLabel}${item.Name}`} disabled={busy === item.ID || (item.TrackingMode === "level" ? item.Level >= 100 : false)} onClick={() => void changeStock(item, 1)}>+</button>
          </span>
          <small className="inventory-unit">{item.TrackingMode === "level" ? "level" : item.Unit}</small>
        </div>;
      })}
    </section>)}
  </div>;
}
