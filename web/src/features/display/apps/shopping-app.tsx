"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { apiBase } from "@/features/display/api/display-api";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";
import type { KioskAppProps } from "@/features/display/apps/kiosk-app";

type ShoppingItem = { id: string; name: string; purchased: boolean; version: number };

export function ShoppingApp({ token }: KioskAppProps) {
  const { data: items, error, setError, reload } = useKioskResource<ShoppingItem[]>("/api/display/shopping", token);
  const [busy, setBusy] = useState("");
  const [newName, setNewName] = useState("");
  const [undoItem, setUndoItem] = useState<ShoppingItem | null>(null);
  const undoTimer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(undoTimer.current), []);

  async function update(item: ShoppingItem) {
    setBusy(item.id);
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/shopping/items/${item.id}`, {
        method: "PATCH",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ purchased: !item.purchased, version: item.version }),
      });
      if (!response.ok) throw new Error();
      const updated = await response.json() as ShoppingItem;
      await reload();
      window.clearTimeout(undoTimer.current);
      if (item.purchased) setUndoItem(null);
      else {
        setUndoItem(updated);
        undoTimer.current = window.setTimeout(() => { setUndoItem(null); void reload(); }, 5_000);
      }
    } catch {
      setError("We couldn’t update the shopping list. Please try again.");
    } finally { setBusy(""); }
  }

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = newName.trim();
    if (!name) return;
    setBusy("new-shopping");
    setError("");
    try {
      const response = await fetch(`${apiBase}/api/display/shopping/items`, {
        method: "POST",
        headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      if (!response.ok) throw new Error();
      setNewName("");
      await reload();
    } catch {
      setError("We couldn’t add that item. Please try again.");
    } finally { setBusy(""); }
  }

  if (error && !items) return <p role="alert">{error}</p>;
  if (!items) return <p role="status">Loading…</p>;
  return <div className="kiosk-list shopping-app">
    <p className="dialog-label">{items.filter(item => !item.purchased).length} items to pick up</p>
    <form className="inline-form" onSubmit={event => void add(event)}>
      <label><span className="visually-hidden">Item</span><input aria-label="Shopping item" value={newName} onChange={event => setNewName(event.target.value)} placeholder="Add an item" /></label>
      <button className="action-button" disabled={!newName.trim() || busy === "new-shopping"}>{busy === "new-shopping" ? "Adding…" : "Add item"}</button>
    </form>
    {error && <p role="alert">{error}</p>}
    {items.filter(item => !item.purchased || item.id === undoItem?.id).map(item => <button
      className={`shopping-row${item.purchased ? " purchased" : ""}`}
      key={item.id}
      disabled={busy === item.id}
      aria-label={`${item.purchased ? "Undo purchased:" : "Mark purchased:"} ${item.name}`}
      onClick={() => void update(item)}>
      <span className="check-ring" aria-hidden="true">{item.purchased ? "✓" : ""}</span>
      <span>{item.name}</span>
      {busy === item.id && <small className="pending-label">Saving…</small>}
    </button>)}
    {undoItem && <p className="shopping-undo" role="status">{undoItem.name} purchased <button onClick={() => void update(undoItem)}>Undo</button></p>}
    <p className="dialog-label">Tap an item when it goes in the basket. Purchased items leave the list after five seconds.</p>
  </div>;
}
