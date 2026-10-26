"use client";

import { useEffect, useState } from "react";
import { getApiBaseUrl, setToken } from "@/lib/api";

export default function AuthCallbackPage() {
  const [message, setMessage] = useState("Completing sign-in…");
  useEffect(() => {
    async function complete() {
      try {
        const query = new URLSearchParams(location.search);
        const code = query.get("code");
        if (query.get("state") !== sessionStorage.getItem("family-hub-oidc-state") || !code) throw new Error("This sign-in link is invalid or has expired. Please try again.");
        const verifier = sessionStorage.getItem("family-hub-oidc-verifier");
        if (!verifier) throw new Error("Sign-in details are missing. Please try again.");
        const baseUrl = getApiBaseUrl();
        const configResponse = await fetch(`${baseUrl}/api/client-config`);
        if (!configResponse.ok) throw new Error("Could not load sign-in settings.");
        const config = await configResponse.json() as { clientID: string; issuer: string };
        const discoveryResponse = await fetch(`${config.issuer.replace(/\/$/, "")}/.well-known/openid-configuration`);
        if (!discoveryResponse.ok) throw new Error("Could not find the sign-in provider.");
        const discovery = await discoveryResponse.json() as { token_endpoint: string };
        const tokenResponse = await fetch(discovery.token_endpoint, { method: "POST", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body: new URLSearchParams({ grant_type: "authorization_code", client_id: config.clientID, code, redirect_uri: `${location.origin}/auth/callback`, code_verifier: verifier }) });
        if (!tokenResponse.ok) throw new Error("The identity provider rejected sign-in.");
        const tokens = await tokenResponse.json() as { access_token: string };
        const exchange = await fetch(`${baseUrl}/api/auth/exchange?client=web`, { method: "POST", headers: { Authorization: `Bearer ${tokens.access_token}` } });
        if (!exchange.ok) throw new Error("Family Hub could not exchange this sign-in.");
        const result = await exchange.json() as { token: string };
        setToken(result.token);
        sessionStorage.removeItem("family-hub-oidc-state");
        sessionStorage.removeItem("family-hub-oidc-verifier");
        location.replace("/");
      } catch (reason) {
        setMessage(reason instanceof Error ? reason.message : "Sign-in could not be completed.");
      }
    }
    void complete();
  }, []);
  return <main className="login-page"><div className="login-card"><span className="brand-mark login-mark">f</span><h1>Family Hub</h1><p role="status">{message}</p><a className="retry-link" href="/login">Return to sign in</a></div></main>;
}
