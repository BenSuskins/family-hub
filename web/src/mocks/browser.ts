import { setupWorker } from "msw/browser";
import { handlers, setMockAutoApproval } from "./handlers";

export const worker = setupWorker(...handlers);

let startPromise: Promise<void> | undefined;

export function startLocalMocks() {
  setMockAutoApproval(new URLSearchParams(window.location.search).get("mock") === "paired");
  startPromise ??= worker.start({ serviceWorker: { url: "/mockServiceWorker.js" }, onUnhandledFrame: "bypass", quiet: true }).then(() => undefined);
  return startPromise;
}
