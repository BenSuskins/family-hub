"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { apiBase } from "@/features/display/api/display-api";
import type { DisplayShoppingItem } from "@/features/display/model/display-types";

export type DisplayShoppingFeedback = {
  message: string;
  undoItem?: DisplayShoppingItem;
};

export function useDisplayShoppingActions(
  token: string,
  reloadShopping: () => Promise<void>,
) {
  const [pendingShoppingId, setPendingShoppingId] = useState("");
  const [feedback, setFeedback] = useState<DisplayShoppingFeedback | null>(
    null,
  );
  const feedbackTimer = useRef<number | undefined>(undefined);

  useEffect(() => () => window.clearTimeout(feedbackTimer.current), []);

  const showFeedback = useCallback(
    (message: string, undoItem?: DisplayShoppingItem) => {
      window.clearTimeout(feedbackTimer.current);
      setFeedback({ message, undoItem });
      feedbackTimer.current = window.setTimeout(
        () => setFeedback(null),
        undoItem ? 5_000 : 4_500,
      );
    },
    [],
  );

  const setPurchased = useCallback(
    async (item: DisplayShoppingItem, purchased: boolean) => {
      if (pendingShoppingId) return;
      setPendingShoppingId(item.id);
      setFeedback(null);
      window.clearTimeout(feedbackTimer.current);
      try {
        const response = await fetch(
          `${apiBase}/api/display/shopping/items/${encodeURIComponent(item.id)}`,
          {
            method: "PATCH",
            headers: {
              Authorization: `Bearer ${token}`,
              "Content-Type": "application/json",
            },
            body: JSON.stringify({ purchased, version: item.version }),
          },
        );
        if (!response.ok) throw new Error();
        const updated = (await response.json()) as DisplayShoppingItem;
        await reloadShopping();
        showFeedback(
          purchased
            ? `${item.name} purchased`
            : `${item.name} returned to the list`,
          purchased ? updated : undefined,
        );
      } catch {
        showFeedback(
          purchased
            ? "We couldn’t update the shopping list. Please try again."
            : "We couldn’t undo that purchase. Please refresh the list.",
        );
      } finally {
        setPendingShoppingId("");
      }
    },
    [pendingShoppingId, reloadShopping, showFeedback, token],
  );

  const toggleShopping = useCallback(
    (item: DisplayShoppingItem) => void setPurchased(item, !item.purchased),
    [setPurchased],
  );
  const undoShopping = useCallback(() => {
    if (feedback?.undoItem) void setPurchased(feedback.undoItem, false);
  }, [feedback, setPurchased]);

  return {
    pendingShoppingId,
    feedback,
    toggleShopping,
    undoShopping,
  };
}
