"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type PointerEvent,
} from "react";
import { getApiBaseUrl } from "@/lib/api";
import { Typography } from "@/components/typography";

type Point = { x: number; y: number };
type Stroke = { color: string; points: Point[] };
const colors = [
  { value: "#3c654c", name: "Sage" },
  { value: "#8b503c", name: "Clay" },
  { value: "#617b98", name: "Blue" },
  { value: "#5c594d", name: "Charcoal" },
  { value: "#d2a82e", name: "Sunshine" },
];

export function KioskScratchpad({
  token,
  onSaveReady,
}: {
  token: string;
  onSaveReady: (save: (() => Promise<void>) | null) => void;
}) {
  const [strokeCount, setStrokeCount] = useState(0);
  const [redrawRevision, setRedrawRevision] = useState(0);
  const [color, setColor] = useState(colors[0].value);
  const [message, setMessage] = useState("Loading the family drawing…");
  const [saving, setSaving] = useState(false);
  const [isDrawing, setIsDrawing] = useState(false);
  const canvas = useRef<HTMLCanvasElement>(null);
  const liveStrokes = useRef<Stroke[]>([]);
  const drawing = useRef(false);
  const dirty = useRef(false);

  useEffect(() => {
    let active = true;
    void fetch(getApiBaseUrl() + "/api/display/scratchpad", {
      headers: { Authorization: "Bearer " + token },
    })
      .then(async (response) => {
        if (!response.ok)
          throw new Error("The family drawing could not be loaded.");
        return response.json() as Promise<{ strokes: Stroke[] }>;
      })
      .then((value) => {
        if (!active) return;
        const draftText = localStorage.getItem(
          "family-hub-display-scratchpad-draft",
        );
        const draft = draftText ? (JSON.parse(draftText) as Stroke[]) : null;
        const current = draft ?? value.strokes;
        liveStrokes.current = current;
        setStrokeCount(current.length);
        setRedrawRevision((revision) => revision + 1);
        dirty.current = Boolean(draft);
        setMessage(
          draft ? "Unsynced changes restored" : "Shared with everyone at home",
        );
      })
      .catch(() => {
        if (!active) return;
        try {
          const draft = JSON.parse(
            localStorage.getItem("family-hub-display-scratchpad-draft") ?? "[]",
          ) as Stroke[];
          liveStrokes.current = draft;
          setStrokeCount(draft.length);
          setRedrawRevision((revision) => revision + 1);
          dirty.current = draft.length > 0;
          setMessage(
            draft.length
              ? "Showing your saved local draft"
              : "The family drawing is unavailable right now.",
          );
        } catch {
          localStorage.removeItem("family-hub-display-scratchpad-draft");
          setMessage("The family drawing is unavailable right now.");
        }
      });
    return () => {
      active = false;
    };
  }, [token]);

  useEffect(() => {
    const element = canvas.current;
    if (!element) return;
    const bounds = element.getBoundingClientRect();
    const ratio = window.devicePixelRatio || 1;
    element.width = Math.round(bounds.width * ratio);
    element.height = Math.round(bounds.height * ratio);
    const context = element.getContext("2d");
    if (!context) return;
    context.scale(ratio, ratio);
    context.lineCap = "round";
    context.lineJoin = "round";
    for (const stroke of liveStrokes.current) {
      if (!stroke.points.length) continue;
      context.strokeStyle = stroke.color;
      context.lineWidth = 4;
      context.beginPath();
      context.moveTo(
        stroke.points[0].x * bounds.width,
        stroke.points[0].y * bounds.height,
      );
      for (const point of stroke.points.slice(1))
        context.lineTo(point.x * bounds.width, point.y * bounds.height);
      context.stroke();
    }
  }, [redrawRevision]);

  function point(event: PointerEvent<HTMLCanvasElement>): Point {
    const bounds = event.currentTarget.getBoundingClientRect();
    return {
      x: Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width)),
      y: Math.max(0, Math.min(1, (event.clientY - bounds.top) / bounds.height)),
    };
  }
  function start(event: PointerEvent<HTMLCanvasElement>) {
    drawing.current = true;
    setIsDrawing(true);
    event.currentTarget.setPointerCapture(event.pointerId);
    const next = [...liveStrokes.current, { color, points: [point(event)] }];
    liveStrokes.current = next;
    dirty.current = true;
    setStrokeCount(next.length);
    setRedrawRevision((revision) => revision + 1);
  }
  function move(event: PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return;
    const last = liveStrokes.current.at(-1);
    const previousPoint = last?.points.at(-1);
    const nextPoint = point(event);
    if (!last || !previousPoint) return;
    last.points.push(nextPoint);
    const element = event.currentTarget;
    const bounds = element.getBoundingClientRect();
    const context = element.getContext("2d");
    if (!context) return;
    context.strokeStyle = last.color;
    context.lineWidth = 4;
    context.lineCap = "round";
    context.lineJoin = "round";
    context.beginPath();
    context.moveTo(
      previousPoint.x * bounds.width,
      previousPoint.y * bounds.height,
    );
    context.lineTo(nextPoint.x * bounds.width, nextPoint.y * bounds.height);
    context.stroke();
    dirty.current = true;
  }
  const save = useCallback(async () => {
    if (!dirty.current) return;
    setSaving(true);
    setMessage("Saving the family drawing…");
    try {
      const response = await fetch(
        getApiBaseUrl() + "/api/display/scratchpad",
        {
          method: "PUT",
          headers: {
            Authorization: "Bearer " + token,
            "Content-Type": "application/json",
          },
          body: JSON.stringify({ strokes: liveStrokes.current }),
        },
      );
      if (!response.ok) throw new Error();
      localStorage.removeItem("family-hub-display-scratchpad-draft");
      dirty.current = false;
      setMessage("Saved for everyone at home");
    } catch {
      localStorage.setItem(
        "family-hub-display-scratchpad-draft",
        JSON.stringify(liveStrokes.current),
      );
      setMessage(
        "We couldn’t save that. Your drawing is still here. Please try again.",
      );
    } finally {
      setSaving(false);
    }
  }, [token]);
  useEffect(() => {
    onSaveReady(save);
    return () => onSaveReady(null);
  }, [onSaveReady, save]);
  function undo() {
    const next = liveStrokes.current.slice(0, -1);
    liveStrokes.current = next;
    dirty.current = true;
    setStrokeCount(next.length);
    setRedrawRevision((revision) => revision + 1);
  }
  function clear() {
    liveStrokes.current = [];
    dirty.current = true;
    setStrokeCount(0);
    setRedrawRevision((revision) => revision + 1);
  }

  return (
    <section className="kiosk-drawing">
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        A shared place for the little things.
      </Typography>
      <div className="kiosk-drawing-tools">
        <Typography as="span" variant="caption">
          Pick a pen colour
        </Typography>
        <div role="group" aria-label="Pen colour">
          {colors.map((pen) => (
            <button
              key={pen.name}
              aria-label={`${pen.name} pen`}
              aria-pressed={color === pen.value}
              className={
                color === pen.value ? "kiosk-color selected" : "kiosk-color"
              }
              onClick={() => setColor(pen.value)}
            >
              <span
                className="kiosk-color-swatch"
                style={{ "--ink-color": pen.value } as CSSProperties}
              />
            </button>
          ))}
        </div>
        <button
          className="kiosk-tool"
          disabled={!strokeCount || saving}
          onClick={undo}
        >
          <Typography as="span" variant="caption">
            Undo line
          </Typography>
        </button>
        <button
          className="kiosk-tool"
          disabled={!strokeCount || saving}
          onClick={clear}
        >
          <Typography as="span" variant="caption">
            Clear page
          </Typography>
        </button>
      </div>
      <canvas
        ref={canvas}
        data-drawing={isDrawing}
        aria-label="Draw on the family scratchpad"
        onPointerDown={start}
        onPointerMove={move}
        onPointerUp={() => {
          drawing.current = false;
          setIsDrawing(false);
        }}
        onPointerCancel={() => {
          drawing.current = false;
          setIsDrawing(false);
        }}
      />
      <Typography as="p" variant="dialogLabel" className="dialog-label">
        Draw with a finger or stylus. Your drawing is shared with everyone at
        home.
      </Typography>
      <Typography
        as="p"
        variant="body"
        role={message.startsWith("We couldn’t save") ? "alert" : "status"}
      >
        {saving ? "Saving changes as you leave…" : message}
      </Typography>
    </section>
  );
}
