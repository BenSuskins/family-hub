# OIDC Client Unification — One Public PKCE Client

## Context

Family Hub currently registers **two** OIDC clients in Authelia:

| Client | Type | Used by | Env var |
|---|---|---|---|
| `family-hub` | confidential (client_secret) | Go web server | `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET` |
| `family-hub-ios` | public (PKCE) | iOS app | `IOS_OIDC_CLIENT_ID` |

For an open-source self-hosted app where setup friction is the top concern,
this doubles the Authelia configuration, doubles the env vars, and adds a
`client_secret` that users must generate, distribute via their env files, and
never commit.

**Goal:** collapse to **one public PKCE client** with two redirect URIs (web
callback + iOS custom scheme). Both surfaces become public clients, PKCE
is required on the token exchange, no shared secret anywhere.

## What changes for users setting up Family Hub

**Before** — two client blocks in Authelia config, two env vars to set
(`OIDC_CLIENT_ID` + `OIDC_CLIENT_SECRET`), plus `IOS_OIDC_CLIENT_ID`.

**After** — one Authelia client block:

```yaml
identity_providers:
  oidc:
    clients:
      - id: family-hub
        public: true
        token_endpoint_auth_method: none
        require_pkce: true
        pkce_challenge_method: S256
        redirect_uris:
          - https://familyhub.example.com/auth/callback
          - familyhub://auth/callback
        scopes: [openid, profile, email]
```

One env var on the server: `OIDC_CLIENT_ID`. No secret.

## Approach

### Phase 1 — Server: switch web flow to PKCE

**`server/internal/services/auth.go`:**

- On `LoginURL(state)`: generate a random 32-byte `code_verifier`, compute
  `code_challenge = base64url(sha256(verifier))`, append
  `code_challenge` + `code_challenge_method=S256` to the auth URL via
  `oauth2.S256ChallengeOption`.
- Store the `code_verifier` keyed by `state` so `HandleCallback` can retrieve
  it. Simplest storage: a short-lived encrypted cookie (reuses the existing
  `securecookie` infra). TTL ~10 min.
- On `HandleCallback(ctx, code, state)`: look up the verifier by state, pass
  `oauth2.VerifierOption(verifier)` to `oauthConfig.Exchange`.
- Drop `ClientSecret` from the `oauth2.Config`.

The `golang.org/x/oauth2` library has native PKCE helpers since v0.22:
- `oauth2.GenerateVerifier() string`
- `oauth2.S256ChallengeOption(verifier) AuthCodeOption`
- `oauth2.VerifierOption(verifier) AuthCodeOption`

**`server/internal/handlers/auth.go`:**

- `LoginPage`: after generating state, generate verifier, set a short-lived
  `oidc_verifier` cookie (encrypted, HttpOnly, SameSite=Lax, MaxAge=600).
- `Callback`: read verifier cookie, clear it, pass to `HandleCallback`.

### Phase 2 — Server: drop client secret config

**`server/internal/config/config.go`:**

- Remove `OIDCClientSecret` field + `OIDC_CLIENT_SECRET` env parsing.
- Remove `IOSClientID` field + `IOS_OIDC_CLIENT_ID` env parsing — one client
  now, so the iOS client config endpoint returns `OIDCClientID` directly.

**`server/internal/server/server.go`:**

- `NewAPIHandler(…, cfg.IOSClientID, …)` becomes `cfg.OIDCClientID`.

**`server/internal/handlers/api.go`:**

- `ClientConfig` handler returns `{clientID: cfg.OIDCClientID, issuer: …}` —
  one source of truth.
- `ExchangeToken`: the bearer token audience validation against the iOS
  client ID becomes validation against the (single) `OIDCClientID`.

### Phase 3 — iOS: point at the unified client

iOS app fetches `/api/client-config` on launch to get the client ID and
issuer, so no code change required beyond re-running against the new
Authelia config. The client ID it receives will now be `family-hub`
instead of `family-hub-ios`.

**But** — the iOS redirect URI remains `familyhub://auth/callback` and
the one in Authelia must match exactly. Verify the iOS app still uses
this scheme.

### Phase 4 — Docs

- `README.md` / setup guide: one Authelia client block, no secret.
- `docs/endpoints.md`: nothing to change (public surface identical).
- `CLAUDE.md`: update Auth paragraph — "single public PKCE client,
  web + iOS share it."
- Delete `plans/2026-04-04-security-hardening.md` reference to client
  secret if applicable.

## Files touched

| File | Change |
|---|---|
| `server/internal/services/auth.go` | Add PKCE verifier generation + cookie storage + verifier passing to Exchange; drop ClientSecret from oauth2.Config |
| `server/internal/handlers/auth.go` | Set/read `oidc_verifier` cookie around login/callback |
| `server/internal/config/config.go` | Drop `OIDCClientSecret` + `IOSClientID` fields |
| `server/internal/server/server.go` | Pass `cfg.OIDCClientID` where `cfg.IOSClientID` was |
| `server/internal/handlers/api.go` | `ClientConfig` + `ExchangeToken` audience use `OIDCClientID` |
| `README.md` | Rewrite Authelia setup section |
| `CLAUDE.md` | Update Auth paragraph |
| `.env.example` (if present) | Remove `OIDC_CLIENT_SECRET`, `IOS_OIDC_CLIENT_ID` |

## Migration notes for existing deployments

Existing self-hosters need to:

1. In Authelia config, delete the `family-hub-ios` client block, mark
   `family-hub` as `public: true` with `token_endpoint_auth_method: none`,
   remove its `secret` field, and add the iOS redirect URI to its
   `redirect_uris` list.
2. Unset `OIDC_CLIENT_SECRET` and `IOS_OIDC_CLIENT_ID` env vars on the
   server.
3. Restart the server.

No database migration. No data loss. Existing sessions and API tokens
remain valid (they're not tied to the OIDC client ID).

## Verification

1. **Unit:** `auth.go` PKCE roundtrip — verifier generated, cookie set,
   verifier retrieved from cookie on callback, passed to Exchange.
2. **Manual web login:** log out, `/login` → Authelia → redirected back
   with `?code=…&state=…`, session cookie set, land on dashboard. Inspect
   the `/authorize` URL to confirm `code_challenge` + `code_challenge_method=S256`
   are present.
3. **Manual iOS login:** clean install iOS app, complete OIDC flow, check
   that `/api/client-config` returns the unified client ID, PKCE flow
   completes, `/api/auth/exchange` returns an API token.
4. **Bad verifier:** tamper with the `oidc_verifier` cookie → callback
   should fail with an auth error (not silently succeed).
5. **Missing verifier cookie:** delete cookie mid-flow → callback fails
   gracefully with "session expired, try again."

## Reuses

- `securecookie` infra for the short-lived verifier cookie.
- `coreos/go-oidc` + `oauth2` — both already support PKCE natively.
- Existing `state` generation for CSRF protection.

## Pros / cons recap (from chat)

**Pros:** dramatically simpler setup docs, one fewer env var, no
`client_secret` to leak, aligns with OAuth 2.1 guidance, one Authelia
client to revoke/rotate.

**Cons:** web server loses `client_secret` as a defense layer (PKCE
replaces it — real-world attack requires breaking TLS, so net neutral
for self-hosted); adds ~15 lines of PKCE state management to the web
login flow; small migration step for existing deployments.

## Out of scope (tracked separately)

- **Collapsing post-login auth patterns** (iOS using OIDC access tokens
  directly instead of our API tokens). That would delete
  `/api/auth/exchange` and the `api_tokens` table but push refresh-token
  handling into iOS. Separate decision.
- **CSRF tokens** on state-changing web forms. Pre-existing gap,
  orthogonal to OIDC client count.
