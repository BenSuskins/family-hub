"use client";

import type { RefObject } from "react";
import {
  DisplayDashboard,
  type DisplayDashboardProps,
} from "@/features/display/components/display-dashboard";
import { DisplayDialog } from "@/features/display/components/display-dialog";
import type { DisplaySelection } from "@/features/display/model/display-types";
import { mealWindowAt } from "@/lib/display-logic";

type DisplayScreenProps = DisplayDashboardProps & {
  selection: DisplaySelection;
  choreFeedback: string;
  token: string;
  dialogRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  onScratchpadSaveReady: (save: (() => Promise<void>) | null) => void;
};

export function DisplayScreen(props: DisplayScreenProps) {
  const {
    snapshot,
    clock,
    selection,
    choreFeedback,
    token,
    dialogRef,
    onClose,
    onRefresh,
    onOpenApp,
    onScratchpadSaveReady,
  } = props;
  const timezone = snapshot?.timezone ?? "Europe/London";
  const localHour = Number(
    new Intl.DateTimeFormat("en-GB", {
      timeZone: timezone,
      hour: "2-digit",
      hourCycle: "h23",
    }).format(clock),
  );
  const dark = localHour >= 20 || localHour < 6;
  const mealLabel = mealWindowAt(clock, timezone).label;

  return (
    <>
      <DisplayDashboard {...props} />
      {choreFeedback && (
        <div className="display-toast" role="status">
          {choreFeedback}
        </div>
      )}
      {selection && (
        <DisplayDialog
          key={selection.kind === "app" ? selection.app : selection.kind}
          selection={selection}
          snapshot={snapshot}
          timezone={timezone}
          mealLabel={mealLabel}
          token={token}
          clock={clock}
          dark={dark}
          dialogRef={dialogRef}
          onClose={onClose}
          onRefresh={onRefresh}
          onOpenApp={(app) => onOpenApp(app)}
          onScratchpadSaveReady={onScratchpadSaveReady}
        />
      )}
    </>
  );
}
