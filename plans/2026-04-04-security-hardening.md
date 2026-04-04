# Security Hardening Plan

Audit-driven fixes to prepare Family Hub for public internet exposure.

## Fixes

### Critical
1. **Session cookie `Secure` flag** — Add `Secure: true` to session + oauth_state cookies (`services/auth.go`, `handlers/auth.go`)
2. **Rate limiting** — Add `go-chi/httprate` middleware on `/login`, `/api/auth/exchange`, and API routes (`server/server.go`)
3. **Security headers** — New middleware for CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy (`middleware/security.go`)
4. **Block dev login in production** — Refuse to start if OIDC not configured unless explicit `DEV_MODE=true` env var (`config/config.go`, `services/auth.go`)

### High
5. **SSRF protection on recipe extractor** — Block private/internal IPs + cloud metadata (`services/recipe_extractor.go`)
6. **SSRF protection on iCal fetcher** — Same IP blocking via shared utility (`services/ical_fetcher.go`)
7. **iCal fetch body size limit** — Use `io.LimitReader` on iCal fetcher response (`services/ical_fetcher.go`)
8. **Remove `/api/client-config` from public routes** — Or accept risk since client IDs aren't secret (move behind auth or document decision)

### Medium
9. **Document data access model** — Family app = shared access by design. No code change needed, just awareness.
10. **Recipe CRUD admin gating** — Intentional design choice for family app. No change unless requested.
11. **Constant-time HA token comparison** — Use `subtle.ConstantTimeCompare` (`handlers/ical.go`)
12. **Separate HA token from iCal token** — Already uses scoped tokens in DB; HA_API_TOKEN is a legacy shortcut. Document.

### Low / Hardening
13. **CSRF protection** — Not implementing full CSRF tokens (HTMX + SameSite=Lax is sufficient for this app's threat model). SameSite already blocks cross-origin POSTs.
14. **SecureCookie encryption key** — Add encryption (second key) to `securecookie.New` (`services/auth.go`, `config/config.go`)
15. **Logout doesn't invalidate server-side** — Cookie-based sessions have no server-side state to invalidate. Clear cookie + Secure flag is sufficient.
16. **Session secret validation** — Enforce minimum length for SESSION_SECRET

## Files to modify
- `server/internal/config/config.go` — Add `DevMode`, `SessionEncryptionKey` fields
- `server/internal/services/auth.go` — Secure cookies, encryption key, dev login guard
- `server/internal/handlers/auth.go` — Secure flag on oauth_state cookie
- `server/internal/handlers/ical.go` — Constant-time compare for HA token
- `server/internal/middleware/security.go` — New file: security headers middleware
- `server/internal/middleware/ratelimit.go` — New file: rate limiting middleware
- `server/internal/services/recipe_extractor.go` — SSRF protection
- `server/internal/services/ical_fetcher.go` — SSRF protection + body size limit
- `server/internal/server/server.go` — Wire up new middleware
- `server/go.mod` — Add `go-chi/httprate` dependency

## Items NOT changing (by design)
- #9: Shared family data access (intentional for family app)
- #10: Recipe CRUD open to all members (intentional)
- #13: No CSRF tokens (SameSite=Lax sufficient)
- #15: No server-side session store (cookie-only is standard for this pattern)
