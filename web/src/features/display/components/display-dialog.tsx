"use client";

import { useState, type RefObject } from "react";
import { X } from "lucide-react";
import { Typography } from "@/components/typography";
import { KioskApp } from "@/features/display/apps/kiosk-app";
import { ChoreCompletionDialog } from "@/features/display/components/chore-completion-dialog";
import type {
  DisplaySelection,
  KioskAppName,
  Snapshot,
} from "@/features/display/model/display-types";
import { displayDialogTitle } from "@/features/display/model/display-formatters";

type DisplayDialogProps = {
  selection: Exclude<DisplaySelection, null>;
  snapshot: Snapshot | null;
  shoppingItems: ShoppingItem[] | null;
  shoppingError: string;
  reloadShopping: () => Promise<void>;
  reloadInventory: () => Promise<void>;
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

type ShoppingItem = {
  id: string;
  name: string;
  purchased: boolean;
  version: number;
};

export function DisplayDialog(props: DisplayDialogProps) {
  const {
    selection,
    snapshot,
    shoppingItems,
    shoppingError,
    reloadShopping,
    reloadInventory,
    timezone,
    mealLabel,
    token,
    clock,
    dark,
    dialogRef,
    onClose,
    onRefresh,
    onOpenApp,
    onScratchpadSaveReady,
  } = props;
  const [appDialogTitle, setAppDialogTitle] = useState<string | null>(null);
  const title = getTitle(selection, mealLabel, appDialogTitle);

  return (
    <div
      className={`display-overlay${dark ? " is-dark" : ""}`}
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <section
        ref={dialogRef}
        className={`display-dialog${selection.kind === "app" && selection.app === "Weather" ? " weather-dialog" : ""}${selection.kind === "app" && selection.app === "Recipes" ? " recipe-dialog" : ""}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="display-dialog-title"
        tabIndex={-1}
      >
        <header className="display-dialog-heading">
          <div>
            <Typography as="h2" variant="dialogTitle" id="display-dialog-title">
              {title}
            </Typography>
          </div>
          <button
            className="close-button"
            onClick={onClose}
            aria-label="Close dialog"
          >
            <X size={24} strokeWidth={1.7} aria-hidden="true" />
          </button>
        </header>
        <div className="display-dialog-content">
          <DialogContent
            selection={selection}
            snapshot={snapshot}
            shoppingItems={shoppingItems}
            shoppingError={shoppingError}
            reloadShopping={reloadShopping}
            reloadInventory={reloadInventory}
            timezone={timezone}
            token={token}
            clock={clock}
            onClose={onClose}
            onRefresh={onRefresh}
            onOpenApp={onOpenApp}
            onDialogTitleChange={setAppDialogTitle}
            onScratchpadSaveReady={onScratchpadSaveReady}
          />
        </div>
      </section>
    </div>
  );
}

function getTitle(
  selection: Exclude<DisplaySelection, null>,
  mealLabel: string,
  appDialogTitle: string | null,
) {
  if (appDialogTitle) return appDialogTitle;
  if (selection.kind === "event") return "Family event";
  if (selection.kind === "chore") return "Who completed this chore?";
  if (selection.app === "launcher") return "Family apps";
  return displayDialogTitle(selection.app, mealLabel);
}

type DialogContentProps = Omit<
  DisplayDialogProps,
  "dialogRef" | "mealLabel" | "dark"
> & {
  onDialogTitleChange: (title: string | null) => void;
};

function DialogContent(props: DialogContentProps) {
  const {
    selection,
    snapshot,
    shoppingItems,
    shoppingError,
    reloadShopping,
    reloadInventory,
    timezone,
    token,
    clock,
    onClose,
    onRefresh,
    onOpenApp,
    onDialogTitleChange,
    onScratchpadSaveReady,
  } = props;
  if (selection.kind === "event")
    return <FamilyEventDetails event={selection.event} timezone={timezone} />;
  if (selection.kind === "chore") {
    return (
      <ChoreCompletionDialog
        chore={selection.chore}
        people={snapshot?.people ?? []}
        token={token}
        onComplete={() => {
          onClose();
          onRefresh();
        }}
      />
    );
  }
  if (selection.app === "launcher") return <Launcher onOpenApp={onOpenApp} />;
  if (selection.app === "Notes") {
    return (
      <KioskApp
        app="Notes"
        token={token}
        onScratchpadSaveReady={onScratchpadSaveReady}
      />
    );
  }
  if (selection.app === "Shopping" || selection.app === "Inventory") {
    if (selection.app === "Shopping")
      return (
        <KioskApp
          app="Shopping"
          token={token}
          items={shoppingItems}
          error={shoppingError}
          reload={reloadShopping}
        />
      );
    return (
      <KioskApp
        app="Inventory"
        token={token}
        onInventoryChanged={reloadInventory}
      />
    );
  }
  if (selection.app === "Calendar") {
    return (
      <KioskApp
        app="Calendar"
        token={token}
        timezone={timezone}
        clock={clock}
      />
    );
  }
  if (selection.app === "Recipes") {
    return <KioskApp app="Recipes" token={token} />;
  }
  if (selection.app === "Meal plan") {
    return (
      <KioskApp
        app="Meal plan"
        token={token}
        timezone={timezone}
        clock={clock}
        initialMeal={null}
        onDialogTitleChange={onDialogTitleChange}
      />
    );
  }
  return (
    <KioskApp app="Weather" token={token} timezone={timezone} clock={clock} />
  );
}

function Launcher({
  onOpenApp,
}: {
  onOpenApp: DisplayDialogProps["onOpenApp"];
}) {
  const tiles = [
    ["Shopping", "＋"],
    ["Notes", "✎"],
    ["Calendar", "▦"],
    ["Recipes", "♧"],
    ["Meal plan", "♨"],
    ["Inventory", "▤"],
  ] as const;
  return (
    <div className="launcher-grid">
      {tiles.map(([app, symbol]) => (
        <button
          className="launcher-tile"
          key={app}
          onClick={() => onOpenApp(app)}
        >
          <span aria-hidden="true">{symbol}</span>
          <Typography as="span" variant="cardTitle">
            {app}
          </Typography>
        </button>
      ))}
    </div>
  );
}

function FamilyEventDetails({
  event,
  timezone,
}: {
  event: Extract<DisplaySelection, { kind: "event" }>["event"];
  timezone: string;
}) {
  const startTime = new Date(event.startTime);
  const endTime = event.endTime ? new Date(event.endTime) : null;
  const leaveAt = event.leaveAt ? new Date(event.leaveAt) : null;

  return (
    <div className="event-detail">
      <Typography as="p" variant="dialogLabel">
        {startTime.toLocaleDateString("en-GB", {
          weekday: "long",
          day: "numeric",
          month: "long",
          timeZone: timezone,
        })}
      </Typography>
      <Typography as="h3" variant="detailTitle">
        {event.title}
      </Typography>
      <Typography as="p" variant="detailBody">
        {startTime.toLocaleTimeString("en-GB", {
          hour: "2-digit",
          minute: "2-digit",
          timeZone: timezone,
        })}
        {endTime
          ? `–${endTime.toLocaleTimeString("en-GB", { hour: "2-digit", minute: "2-digit", timeZone: timezone })}`
          : ""}
      </Typography>
      {event.location && (
        <Typography as="p" variant="detailBody">
          {event.location}
        </Typography>
      )}
      {leaveAt && (
        <Typography as="p" variant="detailBody" className="event-travel">
          Leave at{" "}
          {leaveAt.toLocaleTimeString("en-GB", {
            hour: "2-digit",
            minute: "2-digit",
            timeZone: timezone,
          })}
          {event.drivingSeconds
            ? ` · ${Math.round(event.drivingSeconds / 60)} min drive`
            : ""}
        </Typography>
      )}
    </div>
  );
}
