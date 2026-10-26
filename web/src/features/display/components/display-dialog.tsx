"use client";

import { useState, type RefObject } from "react";
import { X } from "lucide-react";
import { KioskApp } from "@/features/display/apps/kiosk-app";
import { ChoreCompletionDialog } from "@/features/display/components/chore-completion-dialog";
import type { DisplaySelection, KioskAppName, Snapshot } from "@/features/display/model/display-types";
import { displayDialogTitle } from "@/features/display/model/display-formatters";

type DisplayDialogProps = {
  selection: Exclude<DisplaySelection, null>;
  snapshot: Snapshot | null;
  timezone: string;
  mealLabel: string;
  token: string;
  clock: Date;
  dark: boolean;
  dialogRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  onRefresh: () => void;
  onOpenApp: (app: KioskAppName) => void;
  onScratchpadSaveReady: (save: (() => Promise<void>) | null) => void;
};

export function DisplayDialog(props: DisplayDialogProps) {
  const { selection, snapshot, timezone, mealLabel, token, clock, dark, dialogRef, onClose, onRefresh, onOpenApp, onScratchpadSaveReady } = props;
  const [appDialogTitle, setAppDialogTitle] = useState<string | null>(null);
  const title = getTitle(selection, mealLabel, appDialogTitle);

  return <div className={`display-overlay${dark ? " is-dark" : ""}`} onClick={event => { if (event.target === event.currentTarget) onClose(); }}>
    <section ref={dialogRef} className={`display-dialog${selection.kind === "app" && selection.app === "Weather" ? " weather-dialog" : ""}`} role="dialog" aria-modal="true" aria-labelledby="display-dialog-title" tabIndex={-1}>
      <header className="display-dialog-heading"><div><h2 id="display-dialog-title">{title}</h2></div><button className="close-button" onClick={onClose} aria-label="Back to the household overview"><X size={24} strokeWidth={1.7} aria-hidden="true"/></button></header>
      <div className="display-dialog-content">
        <DialogContent
          selection={selection}
          snapshot={snapshot}
          timezone={timezone}
          token={token}
          clock={clock}
          onClose={onClose}
          onRefresh={onRefresh}
          onOpenApp={onOpenApp}
          onDialogTitleChange={setAppDialogTitle}
          onScratchpadSaveReady={onScratchpadSaveReady}
        />
        <button className="display-dialog-back" onClick={() => void onClose()}>Back to the overview</button>
      </div>
    </section>
  </div>;
}

function getTitle(selection: Exclude<DisplaySelection, null>, mealLabel: string, appDialogTitle: string | null) {
  if (appDialogTitle) return appDialogTitle;
  if (selection.kind === "event") return "Family event";
  if (selection.kind === "chore") return "Who completed this chore?";
  if (selection.app === "launcher") return "Family apps";
  return displayDialogTitle(selection.app, mealLabel);
}

type DialogContentProps = Omit<DisplayDialogProps, "dialogRef" | "mealLabel" | "dark"> & {
  onDialogTitleChange: (title: string | null) => void;
};

function DialogContent(props: DialogContentProps) {
  const { selection, snapshot, timezone, token, clock, onClose, onRefresh, onOpenApp, onDialogTitleChange, onScratchpadSaveReady } = props;
  if (selection.kind === "event") return <FamilyEventDetails event={selection.event} timezone={timezone} />;
  if (selection.kind === "chore") {
    return <ChoreCompletionDialog chore={selection.chore} people={snapshot?.people ?? []} token={token} onComplete={() => { onClose(); onRefresh(); }} />;
  }
  if (selection.app === "launcher") return <Launcher onOpenApp={onOpenApp} />;
  if (selection.app === "Notes") {
    return <KioskApp app="Notes" token={token} onScratchpadSaveReady={onScratchpadSaveReady} />;
  }
  if (selection.app === "Shopping" || selection.app === "Inventory") {
    return <KioskApp app={selection.app} token={token} />;
  }
  if (selection.app === "Calendar") {
    return <KioskApp app="Calendar" token={token} timezone={timezone} clock={clock} />;
  }
  if (selection.app === "Recipes") {
    return <KioskApp app="Recipes" token={token} onDialogTitleChange={onDialogTitleChange} />;
  }
  if (selection.app === "Meal plan") {
    return <KioskApp app="Meal plan" token={token} timezone={timezone} clock={clock} initialMeal={null} onDialogTitleChange={onDialogTitleChange} />;
  }
  return <KioskApp app="Weather" token={token} timezone={timezone} clock={clock} />;
}

function Launcher({ onOpenApp }: { onOpenApp: DisplayDialogProps["onOpenApp"] }) {
  const tiles = [["Shopping", "＋"], ["Notes", "✎"], ["Calendar", "▦"], ["Recipes", "♧"], ["Meal plan", "♨"], ["Inventory", "▤"]] as const;
  return <div className="launcher-grid">{tiles.map(([app, symbol]) => <button className="launcher-tile" key={app} onClick={() => onOpenApp(app)}><span aria-hidden="true">{symbol}</span>{app}</button>)}</div>;
}

function FamilyEventDetails({ event, timezone }: { event: Extract<DisplaySelection, { kind: "event" }>['event']; timezone: string }) {
  const startTime = new Date(event.startTime);
  const endTime = event.endTime ? new Date(event.endTime) : null;
  const leaveAt = event.leaveAt ? new Date(event.leaveAt) : null;

  return <div className="event-detail">
    <p>{startTime.toLocaleDateString("en-GB", { weekday: "long", day: "numeric", month: "long", timeZone: timezone })}</p>
    <h3>{event.title}</h3>
    <p>{startTime.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{endTime ? `–${endTime.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}` : ""}</p>
    {event.location && <p>{event.location}</p>}
    {leaveAt && <p className="event-travel">Leave at {leaveAt.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}{event.drivingSeconds ? ` · ${Math.round(event.drivingSeconds / 60)} min drive` : ""}</p>}
  </div>;
}
