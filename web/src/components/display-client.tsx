"use client";

import { MouseEvent, useCallback, useEffect, useRef, useState } from "react";
import { PairDisplay } from "@/features/display/components/pair-display";
import { apiBase } from "@/features/display/api/display-api";
import type { DisplayChore, EventDetails, MealEditTarget } from "@/features/display/model/display-types";
import { householdDocumentTitle } from "@/lib/display-logic";

import { DisplayScreen } from "@/features/display/components/display-screen";
import { useDisplaySession } from "@/features/display/hooks/use-display-session";
import { useDisplaySnapshot } from "@/features/display/hooks/use-display-snapshot";


export function DisplayClient() {
  const [sessionFailures, setSessionFailures] = useState<string[]>([]);
  const reportTokenStorageFailure = useCallback((message: string) => setSessionFailures([message]), []);
  const session = useDisplaySession(reportTokenStorageFailure);
  const { mockReady, token, setToken, clock, clockRef, accept } = session;
  const clearToken = useCallback(() => { setSessionFailures([]); setToken(""); }, []);
  const snapshotState = useDisplaySnapshot(token, clockRef, clearToken);
  const { snapshot, householdName, loading, failures: snapshotFailures, refresh } = snapshotState;
  const failures = [...sessionFailures, ...snapshotFailures];
  const [selectedApp, setSelectedApp] = useState("");
  const [selectedEvent, setSelectedEvent] = useState<EventDetails | null>(null);
  const [mealEditTarget, setMealEditTarget] = useState<MealEditTarget | null>(null);
  const [selectedChore, setSelectedChore] = useState<DisplayChore | null>(null);
  const [pendingChoreId, setPendingChoreId] = useState("");
  const [choreFeedback, setChoreFeedback] = useState("");
  const dialogRef = useRef<HTMLElement>(null);
  const scratchpadSaveRef = useRef<(() => Promise<void>) | null>(null);
  const closeInFlightRef = useRef(false);

  const registerScratchpadSave = useCallback((save: (() => Promise<void>) | null) => {
    scratchpadSaveRef.current = save;
  }, []);
  const close = useCallback(async () => {
    if (closeInFlightRef.current || dialogRef.current?.querySelector("canvas[data-drawing='true']")) return;
    closeInFlightRef.current = true;
    try {
      if (selectedApp === "Notes") await scratchpadSaveRef.current?.();
    } finally {
      closeInFlightRef.current = false;
      setSelectedApp(""); setSelectedEvent(null); setSelectedChore(null); setMealEditTarget(null);
    }
  }, [selectedApp]);
  useEffect(() => {
    document.title = householdDocumentTitle(householdName);
  }, [householdName]);

  useEffect(() => {
    if (!selectedApp) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    dialogRef.current?.focus();
    let timer = window.setTimeout(closeForIdle, 5 * 60_000);
    const resetIdle = () => { window.clearTimeout(timer); timer = window.setTimeout(closeForIdle, 5 * 60_000); };
    function closeForIdle() {
      const drawing = dialogRef.current?.querySelector("canvas[data-drawing='true']");
      if (drawing) { timer = window.setTimeout(closeForIdle, 30_000); return; }
      void close();
    }
    function keydown(event: KeyboardEvent) {
      resetIdle();
      if (event.key === "Escape") { event.preventDefault(); void close(); return; }
      if (event.key !== "Tab" || !dialogRef.current) return;
      const controls = Array.from(dialogRef.current.querySelectorAll<HTMLElement>("button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex='0']"));
      if (!controls.length) return;
      if (event.shiftKey && document.activeElement === controls[0]) { event.preventDefault(); controls.at(-1)?.focus(); }
      else if (!event.shiftKey && document.activeElement === controls.at(-1)) { event.preventDefault(); controls[0].focus(); }
    }
    const dialog = dialogRef.current;
    dialog?.addEventListener("pointerdown", resetIdle);
    dialog?.addEventListener("input", resetIdle);
    window.addEventListener("keydown", keydown);
    return () => {
      window.clearTimeout(timer);
      dialog?.removeEventListener("pointerdown", resetIdle);
      dialog?.removeEventListener("input", resetIdle);
      window.removeEventListener("keydown", keydown);
      previousFocus?.focus();
    };
  }, [selectedApp, close]);

  const open = (app: string, _event?: MouseEvent<HTMLElement>) => {
    setSelectedApp(app);
  };
  const openEvent = (details: EventDetails, event?: MouseEvent<HTMLElement>) => {
    setSelectedEvent(details);
    open("Family event", event);
  };
  const completeAssignedChore = async (chore: DisplayChore) => {
    const userID = chore.AssignedToUserID;
    if (!userID || pendingChoreId) return;
    setPendingChoreId(chore.ID);
    setChoreFeedback("");
    try {
      const response = await fetch(`${apiBase}/api/display/chores/${encodeURIComponent(chore.ID)}/complete`, { method: "POST", headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" }, body: JSON.stringify({ userID }) });
      if (!response.ok) throw new Error(response.status === 409 ? "This chore was already completed." : "The chore could not be completed. Try again.");
      const person = snapshot?.people.find(item => item.id === userID)?.name ?? "Assigned member";
      setChoreFeedback(`Completed for ${person}`);
      window.setTimeout(() => setChoreFeedback(""), 4_500);
      void refresh();
    } catch (reason) {
      setChoreFeedback(reason instanceof Error ? reason.message : "The chore could not be completed. Try again.");
      window.setTimeout(() => setChoreFeedback(""), 4_500);
    } finally { setPendingChoreId(""); }
  };
  if (!mockReady) return <main className="pair-screen"><p role="status">Starting the local display…</p></main>;
  if (!token) return <PairDisplay onClaimed={accept} />;
  if (!clock) return <main className="wall-display-shell" aria-busy="true" />;

  return <DisplayScreen
    snapshot={snapshot}
    clock={clock}
    loading={loading}
    failures={failures}
    selectedApp={selectedApp}
    selectedEvent={selectedEvent}
    selectedChore={selectedChore}
    mealEditTarget={mealEditTarget}
    pendingChoreId={pendingChoreId}
    choreFeedback={choreFeedback}
    token={token}
    dialogRef={dialogRef}
    onOpenApp={open}
    onOpenEvent={openEvent}
    onSelectChore={(chore, event) => { setSelectedChore(chore); open("Chore completion", event); }}
    onCompleteAssignedChore={completeAssignedChore}
    onClose={() => { void close(); }}
    onRefresh={() => { void refresh(); }}
    onScratchpadSaveReady={registerScratchpadSave}
  />;
}
