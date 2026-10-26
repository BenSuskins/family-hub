"use client";

import { PointerEvent, useCallback, useEffect, useRef, useState } from "react";
import { getApiBaseUrl } from "@/lib/api";

type Point = { x: number; y: number };
type Stroke = { color: string; points: Point[] };
const colors = ["#3c654c", "#8b503c", "#b88642", "#343c31", "#5d7ea2", "#9d3c34"];

export function KioskScratchpad({ token, onSaveReady }: { token: string; onSaveReady: (save: (() => Promise<void>) | null) => void }) {
  const [strokes, setStrokes] = useState<Stroke[]>([]);
  const [color, setColor] = useState(colors[0]);
  const [message, setMessage] = useState("Loading the family drawing…");
  const [saving, setSaving] = useState(false);
  const [isDrawing, setIsDrawing] = useState(false);
  const canvas = useRef<HTMLCanvasElement>(null);
  const liveStrokes = useRef<Stroke[]>([]);
  const drawing = useRef(false);
  const dirty = useRef(false);

  useEffect(() => {
    let active = true;
    void fetch(getApiBaseUrl() + "/api/display/scratchpad", { headers: { Authorization: "Bearer " + token } })
      .then(async response => {
        if (!response.ok) throw new Error("The family drawing could not be loaded.");
        return response.json() as Promise<{ strokes: Stroke[] }>;
      })
      .then(value => {
        if (!active) return;
        const draftText = localStorage.getItem("family-hub-display-scratchpad-draft");
        const draft = draftText ? JSON.parse(draftText) as Stroke[] : null;
        const current = draft ?? value.strokes;
        liveStrokes.current = current; setStrokes(current); dirty.current = Boolean(draft);
        setMessage(draft ? "Unsynced changes restored" : "Shared with everyone at home");
      })
      .catch(() => {
        if (!active) return;
        try {
          const draft=JSON.parse(localStorage.getItem("family-hub-display-scratchpad-draft")??"[]") as Stroke[];
          liveStrokes.current=draft; setStrokes(draft); dirty.current=draft.length>0;
          setMessage(draft.length?"Showing your saved local draft":"The family drawing is unavailable right now.");
        } catch {
          localStorage.removeItem("family-hub-display-scratchpad-draft"); setMessage("The family drawing is unavailable right now.");
        }
      });
    return () => { active = false; };
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
    context.scale(ratio, ratio); context.lineCap = "round"; context.lineJoin = "round";
    for (const stroke of liveStrokes.current) {
      if (!stroke.points.length) continue;
      context.strokeStyle = stroke.color; context.lineWidth = 4; context.beginPath();
      context.moveTo(stroke.points[0].x * bounds.width, stroke.points[0].y * bounds.height);
      for (const point of stroke.points.slice(1)) context.lineTo(point.x * bounds.width, point.y * bounds.height);
      context.stroke();
    }
  }, [strokes]);

  function point(event: PointerEvent<HTMLCanvasElement>): Point {
    const bounds = event.currentTarget.getBoundingClientRect();
    return { x: Math.max(0, Math.min(1, (event.clientX - bounds.left) / bounds.width)), y: Math.max(0, Math.min(1, (event.clientY - bounds.top) / bounds.height)) };
  }
  function start(event: PointerEvent<HTMLCanvasElement>) {
    drawing.current = true; setIsDrawing(true); event.currentTarget.setPointerCapture(event.pointerId);
    const next = [...liveStrokes.current, { color, points: [point(event)] }]; liveStrokes.current = next; dirty.current = true; setStrokes(next);
  }
  function move(event: PointerEvent<HTMLCanvasElement>) {
    if (!drawing.current) return;
    const next = liveStrokes.current.slice(); const last = next[next.length - 1];
    next[next.length - 1] = { ...last, points: [...last.points, point(event)] }; liveStrokes.current = next; dirty.current = true; setStrokes(next);
  }
  const save = useCallback(async () => {
    if (!dirty.current) return;
    setSaving(true); setMessage("Saving the family drawing…");
    try {
      const response = await fetch(getApiBaseUrl() + "/api/display/scratchpad", { method: "PUT", headers: { Authorization: "Bearer " + token, "Content-Type": "application/json" }, body: JSON.stringify({ strokes: liveStrokes.current }) });
      if (!response.ok) throw new Error();
      localStorage.removeItem("family-hub-display-scratchpad-draft");
      dirty.current = false;
      setMessage("Saved for everyone at home");
    } catch {
      localStorage.setItem("family-hub-display-scratchpad-draft", JSON.stringify(liveStrokes.current));
      setMessage("We couldn’t save that. Your drawing is still here. Please try again.");
    } finally { setSaving(false); }
  }, [token]);
  useEffect(() => { onSaveReady(save); return () => onSaveReady(null); }, [onSaveReady, save]);
  function undo() { const next = liveStrokes.current.slice(0, -1); liveStrokes.current = next; dirty.current = true; setStrokes(next); }
  function clear() { liveStrokes.current = []; dirty.current = true; setStrokes([]); }

  return <section className="kiosk-drawing"><div className="kiosk-drawing-tools"><div role="group" aria-label="Pen color">{colors.map(value => <button key={value} aria-label={"Choose pen color " + value} aria-pressed={color === value} className={color === value ? "kiosk-color selected" : "kiosk-color"} style={{ backgroundColor: value }} onClick={() => setColor(value)} />)}</div><button className="kiosk-tool" disabled={!strokes.length||saving} onClick={undo}>Undo</button><button className="kiosk-tool" disabled={!strokes.length||saving} onClick={clear}>Clear</button></div><canvas ref={canvas} data-drawing={isDrawing} aria-label="Draw on the family scratchpad" onPointerDown={start} onPointerMove={move} onPointerUp={() => { drawing.current = false; setIsDrawing(false); }} onPointerCancel={() => { drawing.current = false; setIsDrawing(false); }} /><p role={message.startsWith("We couldn’t save")?"alert":"status"}>{saving ? "Saving changes as you leave…" : message}</p></section>;
}
