"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Typography } from "@/components/typography";
import { apiBase } from "@/features/display/api/display-api";

type ShoppingItem = {
  id: string;
  name: string;
  purchased: boolean;
  version: number;
};

export function ShoppingApp({
  token,
  items,
  error,
  reload,
}: {
  token: string;
  items: ShoppingItem[] | null;
  error: string;
  reload: () => Promise<void>;
}) {
  const [busy, setBusy] = useState("");
  const [newName, setNewName] = useState("");
  const [mutationError, setMutationError] = useState("");
  const [undoItem, setUndoItem] = useState<ShoppingItem | null>(null);
  const undoTimer = useRef<number | undefined>(undefined);
  useEffect(() => () => window.clearTimeout(undoTimer.current), []);

  async function update(item: ShoppingItem) {
    setBusy(item.id);
    setMutationError("");
    try {
      const response = await fetch(
        `${apiBase}/api/display/shopping/items/${item.id}`,
        {
          method: "PATCH",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            purchased: !item.purchased,
            version: item.version,
          }),
        },
      );
      if (!response.ok) throw new Error();
      const updated = (await response.json()) as ShoppingItem;
      await reload();
      window.clearTimeout(undoTimer.current);
      if (item.purchased) setUndoItem(null);
      else {
        setUndoItem(updated);
        undoTimer.current = window.setTimeout(() => {
          setUndoItem(null);
          void reload();
        }, 5_000);
      }
    } catch {
      setMutationError(
        "We couldn’t update the shopping list. Please try again.",
      );
    } finally {
      setBusy("");
    }
  }

  async function add(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = newName.trim();
    if (!name) return;
    setBusy("new-shopping");
    setMutationError("");
    try {
      const response = await fetch(`${apiBase}/api/display/shopping/items`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ name }),
      });
      if (!response.ok) throw new Error();
      setNewName("");
      await reload();
    } catch {
      setMutationError("We couldn’t add that item. Please try again.");
    } finally {
      setBusy("");
    }
  }

  if ((mutationError || error) && !items)
    return (
      <Typography as="p" variant="body" role="alert">
        {mutationError || error}
      </Typography>
    );
  if (!items)
    return (
      <Typography as="p" variant="body" role="status">
        Loading…
      </Typography>
    );
  return (
    <div className="kiosk-list shopping-app">
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        {items.filter((item) => !item.purchased).length} items to pick up
      </Typography>
      <form className="inline-form" onSubmit={(event) => void add(event)}>
        <label>
          <span className="visually-hidden">Item</span>
          <input
            aria-label="Shopping item"
            value={newName}
            onChange={(event) => setNewName(event.target.value)}
            placeholder="Add an item"
          />
        </label>
        <button
          className="action-button"
          disabled={!newName.trim() || busy === "new-shopping"}
        >
          <Typography as="span" variant="caption">
            {busy === "new-shopping" ? "Adding…" : "Add item"}
          </Typography>
        </button>
      </form>
      {(mutationError || error) && (
        <Typography as="p" variant="body" role="alert">
          {mutationError || error}
        </Typography>
      )}
      {items
        .filter((item) => !item.purchased || item.id === undoItem?.id)
        .map((item) => (
          <button
            className={`shopping-row${item.purchased ? " purchased" : ""}`}
            key={item.id}
            disabled={busy === item.id}
            aria-label={`${item.purchased ? "Undo purchased:" : "Mark purchased:"} ${item.name}`}
            onClick={() => void update(item)}
          >
            <span className="check-ring" aria-hidden="true">
              {item.purchased ? "✓" : ""}
            </span>
            <Typography as="span" variant="shoppingItem">
              {item.name}
            </Typography>
            {busy === item.id && (
              <Typography
                as="small"
                variant="caption"
                className="pending-label"
              >
                Saving…
              </Typography>
            )}
          </button>
        ))}
      {undoItem && (
        <Typography
          as="p"
          variant="body"
          className="shopping-undo"
          role="status"
        >
          {undoItem.name} purchased{" "}
          <button onClick={() => void update(undoItem)}>
            <Typography as="span" variant="caption">
              Undo
            </Typography>
          </button>
        </Typography>
      )}
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        Tap an item when it goes in the basket. Purchased items leave the list
        after five seconds.
      </Typography>
    </div>
  );
}
