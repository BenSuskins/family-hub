import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { Profiler } from "react";
import { useRef } from "react";
import { Window } from "happy-dom";
import { KioskScratchpad } from "@/components/kiosk-scratchpad";
import { useDisplaySnapshot } from "@/features/display/hooks/use-display-snapshot";
import { useKioskResource } from "@/features/display/hooks/use-kiosk-resource";

const browser = new Window({ url: "http://localhost:3000" });
Object.assign(globalThis, {
  window: browser,
  document: browser.document,
  navigator: browser.navigator,
  HTMLElement: browser.HTMLElement,
  Element: browser.Element,
  PointerEvent: browser.PointerEvent,
  localStorage: browser.localStorage,
  IS_REACT_ACT_ENVIRONMENT: true,
});

const { cleanup, fireEvent, render, screen, waitFor } =
  await import("@testing-library/react");
const originalFetch = globalThis.fetch;

afterEach(() => {
  cleanup();
  globalThis.fetch = originalFetch;
});

function ResourceProbe({ path }: { path: string }) {
  const { data, error } = useKioskResource<{ name: string }>(
    path,
    "test-token",
  );
  return <p>{data?.name ?? error ?? "Loading resource"}</p>;
}

function SnapshotProbe() {
  const clock = useRef<Date | null>(null);
  const { snapshot, householdName } = useDisplaySnapshot(
    "test-token",
    clock,
    () => {},
  );
  return (
    <p>
      {snapshot
        ? `${householdName} · ${snapshot.timezone}`
        : "Loading household"}
    </p>
  );
}

test("snapshot settings and home information load concurrently", async () => {
  const requests = new Map<string, (response: Response) => void>();
  globalThis.fetch = (input) =>
    new Promise((resolve) => {
      const path = new URL(String(input), "http://localhost").pathname;
      requests.set(path, (response) => resolve(response));
    });

  render(<SnapshotProbe />);
  await waitFor(() => {
    assert.equal(requests.has("/api/display/settings"), true);
    assert.equal(requests.has("/api/display/home-info"), true);
  });

  requests.get("/api/display/home-info")?.(
    Response.json({
      timezone: "Europe/London",
      familyName: "The Example Family",
    }),
  );
  requests.get("/api/display/settings")?.(
    Response.json({ id: "display-1", name: "Kitchen", enabledSections: [] }),
  );
  await screen.findByText("The Example Family · Europe/London");
});

test("a response for an old kiosk resource path cannot replace the current resource", async () => {
  const requests = new Map<string, (response: Response) => void>();
  globalThis.fetch = (input) =>
    new Promise((resolve) => {
      const path = new URL(String(input), "http://localhost").pathname;
      requests.set(path, (response) => resolve(response));
    });

  const view = render(<ResourceProbe path="/old-resource" />);
  await waitFor(() => assert.equal(requests.has("/old-resource"), true));
  view.rerender(<ResourceProbe path="/new-resource" />);
  await waitFor(() => assert.equal(requests.has("/new-resource"), true));

  requests.get("/new-resource")?.(Response.json({ name: "Current resource" }));
  await screen.findByText("Current resource");
  requests.get("/old-resource")?.(Response.json({ name: "Outdated resource" }));
  await new Promise((resolve) => setTimeout(resolve, 0));

  assert.equal(
    screen.getByText("Current resource").textContent,
    "Current resource",
  );
});

test("scratchpad pointer moves draw on canvas without rerendering React", async () => {
  globalThis.fetch = async () => Response.json({ strokes: [] });
  const context = {
    beginPath() {},
    moveTo() {},
    lineTo() {},
    scale() {},
    stroke() {
      this.strokeCount += 1;
    },
    strokeCount: 0,
  };
  browser.HTMLCanvasElement.prototype.getContext = (() =>
    context) as unknown as typeof browser.HTMLCanvasElement.prototype.getContext;
  browser.HTMLCanvasElement.prototype.getBoundingClientRect = (() => ({
    x: 0,
    y: 0,
    left: 0,
    top: 0,
    right: 100,
    bottom: 100,
    width: 100,
    height: 100,
    toJSON() {},
  })) as typeof browser.HTMLCanvasElement.prototype.getBoundingClientRect;
  browser.HTMLCanvasElement.prototype.setPointerCapture = () => {};

  let renderCount = 0;
  function TrackedScratchpad() {
    return <KioskScratchpad token="test-token" onSaveReady={() => {}} />;
  }

  render(
    <Profiler
      id="scratchpad"
      onRender={() => {
        renderCount += 1;
      }}
    >
      <TrackedScratchpad />
    </Profiler>,
  );
  await screen.findByText("Shared with everyone at home");
  const canvas = screen.getByLabelText("Draw on the family scratchpad");
  fireEvent.pointerDown(canvas, { pointerId: 1, clientX: 10, clientY: 10 });
  const rendersDuringStroke = renderCount;
  const strokesBeforeMoves = context.strokeCount;

  for (let index = 0; index < 8; index += 1) {
    fireEvent.pointerMove(canvas, {
      pointerId: 1,
      clientX: 15 + index * 5,
      clientY: 20 + index * 5,
    });
  }

  assert.equal(renderCount, rendersDuringStroke);
  assert.equal(context.strokeCount - strokesBeforeMoves, 8);
});
