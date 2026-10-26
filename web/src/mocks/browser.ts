import { setupWorker } from "msw/browser";
import { handlers, setMockAutoApproval, setMockClock } from "./handlers";

export const worker = setupWorker(...handlers);

let startPromise: Promise<void> | undefined;

export function startLocalMocks() {
  const search = new URLSearchParams(window.location.search);
  setMockAutoApproval(search.get("mock") === "paired");
  setMockClock(search.get("mockTime"));
  startPromise ??= worker.start({ serviceWorker: { url: "/mockServiceWorker.js" }, onUnhandledFrame: "bypass", quiet: true }).then(() => undefined);
  return startPromise;
}
