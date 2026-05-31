# Family Hub

> Self-hosted family organization hub for chores, meals, recipes, and calendars.

[**iOS app on the App Store: Family Home Hub →**](https://apps.apple.com/gb/app/family-home-hub/id6761337807)

## Overview

Family Hub is a self-hosted web app for managing a household: chores with
recurring schedules and random assignment, a weekly meal planner backed by a
recipe library, unified calendar view with external iCal subscriptions, and
a per-user dashboard. The Go server serves both an HTMX/Templ web UI and a
SwiftUI iOS app from the same REST API. Auth is OIDC via a single public
PKCE client shared across web, iOS, and the Home Assistant integration.

## Features

- **Chores** — recurring schedules (daily/weekly/monthly/cron), random assignment, overdue tracking
- **Dashboard** — today's chores, overdue items, upcoming events, completion stats, household leaderboard
- **Calendar** — unified view of chores, family events, and external iCal subscriptions
- **iCal subscriptions** — admin-managed feeds (school, sports, etc.)
- **Meal planning** — weekly planner (breakfast/lunch/dinner) linked to the recipe library
- **Recipes** — ingredient groups, cooking times, import from URL (JSON-LD + HTML fallback)
- **REST API** — session cookie or Bearer token; same surface for web and iOS. See [`endpoints.md`](endpoints.md)
- **Admin panel** — user/role management, chore categories, API tokens, DB backup/restore

## Tech Stack

- **Backend:** Go, [Chi](https://github.com/go-chi/chi) router, [Templ](https://templ.guide/), [HTMX](https://htmx.org/), Tailwind
- **Database:** SQLite (`modernc.org/sqlite` — pure Go, no CGO)
- **Auth:** OIDC via Authelia (single public PKCE client, no client secret)
- **iOS:** SwiftUI, MVVM
- **HA integration:** custom HACS component

## Project Structure

```
family-hub/
├── server/                  # Go backend (HTTP server, templates, REST API)
├── ios/                     # SwiftUI iOS app
├── home-assistant/          # HACS custom component
├── data/                    # Runtime data (SQLite, gitignored)
└── docs/                    # README, ARCHITECTURE, TROUBLESHOOTING, OIDC setup, endpoints
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js (for Tailwind)
- [Templ](https://templ.guide/) CLI
- An OIDC provider supporting public PKCE clients (e.g. Authelia, Keycloak, Auth0) — see [oidc-setup.md](oidc-setup.md)

### Setup — Quick Start (Docker)

```bash
git clone git@github.com:bensuskins/family-hub.git
cd family-hub/server
cp .env.example .env                                   # fill in OIDC + secrets
docker compose -f docker-compose.prod.yml up -d
```

Verify: open `http://localhost:8080`. **The first user to sign in is auto-promoted to admin.**

### Setup — Local

```bash
cd server
go mod download
npm install
make run
```

For hot reload: `make dev` (Air watches Go + Templ changes).

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_PATH` | no | `./data/family-hub.db` | SQLite file path |
| `OIDC_ISSUER` | yes | — | OpenID Connect issuer URL |
| `OIDC_CLIENT_ID` | yes | — | OAuth2 client ID (single public PKCE client) |
| `OIDC_REDIRECT_URL` | yes | — | OAuth2 callback URL |
| `SESSION_SECRET` | yes | — | Session encryption key |
| `BASE_URL` | no | `http://localhost:8080` | Public base URL of the app |
| `SESSION_ENCRYPTION_KEY` | no | — | 16 or 32 ASCII chars to encrypt session cookies. Unset = signed-only |
| `LOG_LEVEL` | no | `info` | `debug`/`info`/`warn`/`error` |
| `PORT` | no | `8080` | Server port |
| `DEV_MODE` | no | `false` | Bypass OIDC + auto-login as dev admin (**never in prod**) |

For provider-specific OIDC client configuration (Authelia, Keycloak, Auth0), see [oidc-setup.md](oidc-setup.md).

## Commands

Run from `server/`:

| Command | Purpose |
|---------|---------|
| `make build` | Regenerate templates, rebuild CSS, compile Go binary |
| `make run` | Build and run |
| `make dev` | Air hot reload |
| `make templ` | Regenerate Templ templates |
| `make css` | Rebuild Tailwind CSS |
| `make test` | Run all tests |
| `make test-coverage` | Generate HTML coverage report |
| `make docker-dev` | Dev stack via Docker Compose |
| `make docker-prod` | Prod stack via Docker Compose |

## Testing

```bash
cd server && make test
```

Repository tests use in-memory SQLite (`:memory:`). Prefer fakes over mocks;
fakes live alongside the interface.

## Deployment

The production stack is Docker Compose:

```bash
cd server && docker compose -f docker-compose.prod.yml up -d
```

Migrations run automatically on startup from `server/internal/database/migrations/`. The image is pure Go (no CGO required).

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for the system diagram, request flow, OIDC auth flow, and key design decisions.

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues (redirect URI mismatches, missing templ CLI, iOS connectivity, etc.).

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](../LICENSE)
