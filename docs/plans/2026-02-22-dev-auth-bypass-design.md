# Dev Auth Bypass Design

**Date:** 2026-02-22
**Status:** Approved

## Problem

When `OIDC_ISSUER` is not set, `AuthService` starts in auth-disabled mode. The `RequireAuth` middleware still redirects unauthenticated requests to `/login`, which returns HTTP 503 "OIDC not configured". This creates a dead loop — the app is unusable locally without a real OIDC provider.

## Solution

When OIDC is not configured, hitting `/login` automatically creates (or fetches) a dev admin user, sets a real session cookie, and redirects to `/`. No form, no interaction.

## Approach

**Option B: Auto-login at `/login`** was chosen over:
- Option A (auto-bypass in middleware) — rejected because a synthetic user with no DB record breaks user-dependent features
- Option C (auto-inject session in middleware) — rejected because it mixes DB writes and cookie setting into the auth middleware

## Design

### `AuthService.DevLogin(ctx) (models.User, error)`

New method on `AuthService`:
- Looks up user by OIDC subject `"dev-user"` in the DB
- If not found, creates one: name `"Dev Admin"`, email `"dev@localhost"`, role `admin`
- Logs a warning: `"dev auto-login, do not use in production"`

### `AuthHandler.LoginPage`

Replace the current 503 branch with:
```
DevLogin(ctx) → SetSession(w, user.ID) → redirect /
```

## Scope

- `internal/services/auth.go` — add `DevLogin` method
- `internal/handlers/auth.go` — replace 503 branch in `LoginPage`

No new files, no new config, no new env vars.

## Gate

"OIDC not configured" (`cfg.OIDCIssuer == ""`) is the only gate. Explicit and sufficient.

## Logout

Redirects to `/login`, which re-logs in instantly. Acceptable for dev.
