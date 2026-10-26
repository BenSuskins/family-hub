"use client";

import { useCallback, useEffect, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { Home } from "lucide-react";
import { apiBase } from "@/features/display/api/display-api";
import type { Pairing } from "@/features/display/model/display-types";

async function requestPairing(): Promise<Pairing> {
  const response = await fetch(apiBase + "/api/display-pairing/challenges", { method: "POST" });
  if (!response.ok) throw new Error("Pairing is unavailable.");
  return response.json() as Promise<Pairing>;
}

export function PairDisplay({ onClaimed }: { onClaimed: (token: string) => void }) {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [error, setError] = useState("");
  const [renewing, setRenewing] = useState(false);
  const [clock, setClock] = useState(() => new Date());
  const create = useCallback(async () => {
    setRenewing(true);
    setError("");
    try {
      setPairing(await requestPairing());
    } catch {
      setError("We couldn’t connect to Family Hub. Check the connection, then try again.");
    } finally { setRenewing(false); }
  }, []);
  useEffect(() => {
    void requestPairing().then(setPairing).catch(() => setError("We couldn’t connect to Family Hub. Check the connection, then try again."));
    const timer = window.setInterval(() => setClock(new Date()), 30_000);
    return () => window.clearInterval(timer);
  }, [create]);
  useEffect(() => {
    if (!pairing) return;
    let stopped = false;
    let requestInFlight = false;
    async function checkClaim() {
      if (requestInFlight || stopped) return;
      requestInFlight = true;
      try {
        const response = await fetch(`${apiBase}/api/display-pairing/challenges/${pairing!.challengeId}/claim`, {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ pairingCode: pairing!.pairingCode }),
        });
        if (response.ok) {
          const body = await response.json() as { token: string };
          if (!stopped && Date.parse(pairing!.expiresAt) > Date.now()) onClaimed(body.token);
        } else if (response.status === 409 && !stopped && Date.parse(pairing!.expiresAt) > Date.now()) {
          setError("");
        } else if (!stopped && response.status !== 409) {
          setError("The display could not check its pairing. It will keep trying.");
        }
      } catch {
        if (!stopped) setError("The display could not check its pairing. It will keep trying.");
      } finally { requestInFlight = false; }
    }
    void checkClaim();
    const timer = window.setInterval(() => {
      if (Date.parse(pairing.expiresAt) <= Date.now()) {
        stopped = true;
        setPairing(null);
        setError("This pairing code expired. Create a new one to continue.");
      } else void checkClaim();
    }, 5_000);
    return () => { stopped = true; window.clearInterval(timer); };
  }, [pairing, onClaimed]);
  const qrValue = pairing ? JSON.stringify({ apiBaseUrl: apiBase, challengeId: pairing.challengeId, pairingCode: pairing.pairingCode }) : "";
  const pairingHour = clock ? Number(new Intl.DateTimeFormat("en-GB", { timeZone: "Europe/London", hour: "2-digit", hourCycle: "h23" }).format(clock)) : 12;
  const isDark = pairingHour >= 20 || pairingHour < 6;
  return <main className={`pair-screen${isDark ? " is-dark" : ""}`}>
    <section className="wall-pairing" aria-label="Family Hub display pairing">
      <span className="wall-pair-mark" aria-hidden="true"><Home size={34} strokeWidth={1.7}/></span>
      <h1>{error.includes("expired") ? "Let’s try that again." : "Bring your household together."}</h1>
      <p>Open Family Hub on your iPhone.<br/>Go to Settings → Wall displays → Scan a display code.</p>
      {pairing ? <div className="wall-qr"><QRCodeSVG value={qrValue} size={180} level="M" includeMargin /></div> : !error && <div className="wall-qr-empty" role="status">{renewing ? "Preparing a pairing code…" : "Pairing code unavailable"}</div>}
      {error && <p className="wall-pair-error" role="alert">{error}</p>}
      {pairing && <small>Single-use code · Expires in five minutes</small>}
      {(!pairing || error) && <button className="action-button" disabled={renewing} onClick={() => void create()}>{pairingActionLabel(renewing, error)}</button>}
      <p className="wall-pair-owner">A shared household screen. No personal login needed here.</p>
    </section>
  </main>;
}

function pairingActionLabel(renewing: boolean, error: string) {
  if (renewing) return "Preparing…";
  if (error.includes("expired")) return "Generate new code";
  if (error) return "Try again";
  return "Create pairing code";
}

