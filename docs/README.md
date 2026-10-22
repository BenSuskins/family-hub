# Family Hub

> Self-hosted family organization hub for chores, meals, recipes, and calendars.

## Screenshots

<img src="ios.png?raw=true" alt="Family Hub Screenshots" title="Screenshots" width="300">

[**iOS app on the App Store: Family Home Hub →**](https://apps.apple.com/gb/app/family-home-hub/id6761337807)

## Overview

Family Hub is a self-hosted household service for chores, meals, recipes,
calendars, shopping, and inventory. The Go server provides a JSON and media API
for the SwiftUI iOS app and its extensions. The iOS app handles OIDC PKCE sign-in
and exchanges the OIDC access token for a long-lived API token.

## Features

- **Chores** — recurring schedules (daily/weekly/monthly/cron), random assignment, overdue tracking
- **Notifications** — two daily chore reminders and admin family messages on enabled iPhones; see [setup](chore-reminders.md)
- **Dashboard** — today's chores, overdue items, upcoming events, and completion stats
- **Calendar** — unified view of chores, family events, and external iCal subscriptions
- **iCal subscriptions** — admin-managed feeds (school, sports, etc.)
- **Meal planning** — weekly planner (breakfast/lunch/dinner) linked to the recipe library
- **Recipes** — ingredient groups, cooking times, import from URL (JSON-LD + HTML fallback)
- **REST API** — Bearer token authentication for the iOS app. See [`endpoints.md`](endpoints.md)
- **Admin operations** — user/role management, chore categories, API tokens, and database backup/restore through the API

## Tech Stack

- **Backend:** Go, [Chi](https://github.com/go-chi/chi) router, JSON REST API
- **Database:** SQLite (`modernc.org/sqlite` — pure Go, no CGO)
- **Auth:** OIDC via Authelia (single public PKCE client, no client secret)
- **iOS:** SwiftUI, MVVM

## Project Structure

```
family-hub/
├── server/                  # Go API server
├── ios/                     # SwiftUI iOS app
├── data/                    # Runtime data (SQLite, gitignored)
└── docs/                    # README, ARCHITECTURE, TROUBLESHOOTING, OIDC setup, endpoints
```

## Getting Started

### Prerequisites

- Go 1.22+
- An OIDC provider supporting public PKCE clients (e.g. Authelia, Keycloak, Auth0) — see [oidc-setup.md](oidc-setup.md)

### Setup — Quick Start (Docker)

```bash
git clone git@github.com:bensuskins/family-hub.git
cd family-hub/server
cp .env.example .env                                   # fill in OIDC + secrets
docker compose -f docker-compose.prod.yml up -d
```

Verify the server with `curl http://localhost:8080/health`. The first user who
exchanges an OIDC token is auto-promoted to admin.

### Setup — Local

```bash
cd server
go mod download
make run
```

For hot reload, run `make dev`.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_PATH` | no | `./data/family-hub.db` | SQLite file path |
| `OIDC_ISSUER` | yes | — | OpenID Connect issuer URL |
| `OIDC_CLIENT_ID` | yes | — | OAuth2 client ID (single public PKCE client) |
| `LOG_LEVEL` | no | `info` | `debug`/`info`/`warn`/`error` |
| `PORT` | no | `8080` | Server port |

For provider-specific OIDC client configuration (Authelia, Keycloak, Auth0), see [oidc-setup.md](oidc-setup.md).

## Commands

Run from `server/`:

| Command | Purpose |
|---------|---------|
| `make build` | Compile the Go server |
| `make run` | Build and run |
| `make dev` | Air hot reload |
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

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common server and iOS connectivity issues.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

[MIT](../LICENSE)

## iOS household features

- **Home Info** — admins save a confirmed device location under Profile → Admin
  → Home Info. The arrival margin starts at 10 minutes.
- **Weather** — Open-Meteo supplies current conditions and a 24-hour forecast.
  Advice uses forecast rules. The free service assumes non-commercial use.
- **Departures** — Google supplies current traffic-aware driving estimates from
  home for located events. Configure `GOOGLE_MAPS_API_KEY` at runtime. Enable
  Geocoding API and Routes API in the Google Cloud project. Restrict the key to
  those APIs and set provider quotas. Google requires a billing account even
  when requests fit its free allowance. No Google result is stored offline.
- **Shopping** — open the list from Home or Inventory. Add inventory items with
  the Add to shopping list action. Purchase offers a separate restock action.
  Cached shopping lists remain readable without a server connection.

The API additions preserve older client contracts. The new iOS features show an
unavailable state when connected to an older server. Install the server before
the new app. No production rollout is included in this change.

Run the optional provider check with public London locations:

```bash
cd server
RUN_HOUSEHOLD_PROVIDER_CHECK=1 go test ./internal/services -run TestHouseholdProvidersLive -count=1 -v
```

The check calls Open-Meteo. It also checks Google when `GOOGLE_MAPS_API_KEY` is
present in the test process environment. It never prints the key.
