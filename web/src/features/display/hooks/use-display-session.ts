"use client";

import { useCallback, useEffect, useRef, useState, type MutableRefObject } from "react";
import { displayTokenKey, localMocksEnabled, readStorage, writeStorage } from "@/features/display/api/display-api";

type DisplaySession = {
  mockReady: boolean;
  token: string;
  setToken: (token: string) => void;
  clock: Date | null;
  clockRef: MutableRefObject<Date | null>;
  accept: (token: string) => void;
};

export function useDisplaySession(onTokenStorageFailure: (message: string) => void): DisplaySession {
  const [mockReady, setMockReady] = useState(!localMocksEnabled);
  const [token, setToken] = useState("");
  const [clock, setClock] = useState<Date | null>(null);
  const clockRef = useRef<Date | null>(null);

  const accept = useCallback((value: string) => {
    if (!writeStorage(displayTokenKey, value)) onTokenStorageFailure("This browser cannot save the pairing. Keep this tab open or enable site storage.");
    setToken(value);
  }, [onTokenStorageFailure]);

  useEffect(() => {
    let stopped = false;
    let timer: number | undefined;
    async function initialise() {
      if (localMocksEnabled) {
        const { startLocalMocks } = await import("@/mocks/browser");
        await startLocalMocks();
      }
      if (stopped) return;
      const storedToken = readStorage(displayTokenKey) ?? "";
      let initialClock = new Date();
      if (localMocksEnabled) {
        try {
          const response = await fetch("/api/local-mocks/clock");
          if (response.ok) initialClock = new Date((await response.json() as { now: string }).now);
        } catch { /* Use the system clock if the local mock clock is unavailable. */ }
      }
      clockRef.current = initialClock;
      setClock(initialClock);
      setToken(storedToken);
      setMockReady(true);
      if (localMocksEnabled) {
        let lastMockClock = initialClock;
        let lastReadAt = Date.now();
        timer = window.setInterval(() => {
          void fetch("/api/local-mocks/clock").then(async response => {
            if (!response.ok) return;
            const nextClock = new Date((await response.json() as { now: string }).now);
            const expected = lastMockClock.getTime() + (Date.now() - lastReadAt);
            const jumped = Math.abs(nextClock.getTime() - expected) > 30_000;
            lastMockClock = nextClock;
            lastReadAt = Date.now();
            clockRef.current = nextClock;
            setClock(nextClock);
            if (jumped) window.dispatchEvent(new Event("family-hub:mock-clock-changed"));
          }).catch(() => undefined);
        }, 5_000);
      } else {
        timer = window.setInterval(() => { const nextClock = new Date(); clockRef.current = nextClock; setClock(nextClock); }, 30_000);
      }
    }
    void initialise();
    return () => { stopped = true; if (timer) window.clearInterval(timer); };
  }, []);

  return { mockReady, token, setToken, clock, clockRef, accept };
}
