# Troubleshooting

> Common issues and fixes for Family Hub. Organised by symptom.

## How to use this doc

- Search by symptom (the bold line) first
- If you fix something not listed here, add it — bar is "would future-me have wanted this written down?"
- Keep entries short: symptom → cause → fix

---

## Setup

**`templ: command not found` during local build**

Cause: the `templ` CLI is not on your `PATH`.
Fix:
```bash
go install github.com/a-h/templ/cmd/templ@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Auth

**`redirect_uri_mismatch` or login loop**

Cause: `OIDC_REDIRECT_URL` does not exactly match the URI registered with your OIDC provider (scheme, host, port, and path all matter).
Fix: set it to the exact value the provider expects, e.g. `OIDC_REDIRECT_URL=https://hub.example.com/auth/callback`.

**Blank page or 500 after OIDC callback**

Cause: `SESSION_SECRET` is empty or unset. Without it the server cannot encode session cookies and rejects every login.
Fix: set `SESSION_SECRET` to a strong random string (e.g. `openssl rand -hex 32`).

**First user is not admin**

Cause: only the very first OIDC login is auto-promoted to admin. A different account beat you to it.
Fix: either manually update the `role` column in SQLite, or temporarily set `DEV_MODE=true` to log in as a dev admin, then promote your real account from the Admin panel and unset `DEV_MODE`.

## iOS

**iOS app shows "Unable to connect"**

Cause: the app fetches `GET /api/client-config` to discover the OIDC issuer and client ID. If that endpoint isn't reachable from the device, login fails.
Fix: check `BASE_URL` matches the publicly reachable URL of your server and that `/api/client-config` is not blocked by a firewall or reverse proxy.
