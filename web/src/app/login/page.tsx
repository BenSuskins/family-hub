"use client";

import { useState } from "react";
import { getApiBaseUrl } from "@/lib/api";

const verifierKey = "family-hub-oidc-verifier";
const stateKey = "family-hub-oidc-state";

function encode(value: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(value))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export default function LoginPage() {
  const [error, setError] = useState("");
  async function signIn() {
    try {
      const baseUrl = getApiBaseUrl();
      const configResponse = await fetch(`${baseUrl}/api/client-config`);
      if (!configResponse.ok) throw new Error("Could not load sign-in settings.");
      const config = await configResponse.json() as { clientID: string; issuer: string };
      const discoveryResponse = await fetch(`${config.issuer.replace(/\/$/, "")}/.well-known/openid-configuration`);
      if (!discoveryResponse.ok) throw new Error("Could not find the sign-in provider.");
      const discovery = await discoveryResponse.json() as { authorization_endpoint: string };
      const randomBytes = crypto.getRandomValues(new Uint8Array(32));
      const verifier = encode(randomBytes.buffer);
      const state = encode(crypto.getRandomValues(new Uint8Array(24)).buffer);
      const challenge = encode(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier)));
      sessionStorage.setItem(verifierKey, verifier);
      sessionStorage.setItem(stateKey, state);
      const url = new URL(discovery.authorization_endpoint);
      url.search = new URLSearchParams({ client_id: config.clientID, redirect_uri: `${location.origin}/auth/callback`, response_type: "code", scope: "openid profile email", state, code_challenge: challenge, code_challenge_method: "S256" }).toString();
      location.assign(url);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Sign-in could not start.");
    }
  }
  return <main className="login-page"><div className="login-card"><span className="brand-mark login-mark">f</span><p className="kicker">YOUR HOME, TOGETHER</p><h1>Sign in to Family Hub</h1><p>Use your household account to see what’s happening at home.</p><button className="sign-in-button" onClick={signIn}>Continue with household account <span>→</span></button>{error && <p role="alert" className="error-message">{error}</p>}<small>Your sign-in is handled by your configured identity provider.</small></div></main>;
}
