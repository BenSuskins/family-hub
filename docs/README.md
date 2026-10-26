# Family Hub

> Self-hosted family organization hub for chores, meals, recipes, and calendars.

## Screenshots

<img src="ios.png?raw=true" alt="Family Hub Screenshots" title="Screenshots" width="300">

[**iOS app on the App Store: Family Home Hub →**](https://apps.apple.com/gb/app/family-home-hub/id6761337807)

## Overview

Family Hub is a self-hosted household service for chores, meals, recipes,
calendars, shopping, and inventory. The Go server provides a JSON and media API
for the SwiftUI iOS app and its extensions. A separately deployed web client
serves only the QR-paired wall display.

## Features

- **Chores** — recurring schedules (daily/weekly/monthly/cron), random assignment, overdue tracking
- **Notifications** — two daily chore reminders and admin family messages on enabled iPhones; see [setup](chore-reminders.md)
- **Dashboard** — today's chores, overdue items, upcoming events, and completion stats
- **Calendar** — unified view of chores, family events, and external iCal subscriptions
- **iCal subscriptions** — admin-managed feeds (school, sports, etc.)
- **Meal planning** — weekly planner (breakfast/lunch/dinner) linked to the recipe library
- **Recipes** — ingredient groups, cooking times, import from URL (JSON-LD + HTML fallback)
- **Wall display** — a household dashboard paired and configured from iOS; see the [web guide](../web/README.md)
- **REST API** — scoped Bearer tokens for iOS and paired displays. See [`endpoints.md`](endpoints.md)
- **Admin operations** — user/role management, chore categories, API tokens, and database backup/restore through the API

## Tech Stack

- **Backend:** Go, [Chi](https://github.com/go-chi/chi) router, JSON REST API
- **Database:** SQLite (`modernc.org/sqlite` — pure Go, no CGO)
- **Auth:** Public OIDC PKCE for iOS; scoped Family Hub API tokens for iOS and paired displays
- **iOS:** SwiftUI, MVVM
- **Web:** Next.js, React, TypeScript, Bun

## Project Structure

```
family-hub/
├── server/                  # Go API server
├── ios/                     # SwiftUI iOS app and extensions
├── web/                     # Next.js QR pairing and wall display
├── data/                    # Runtime data (SQLite, gitignored)
└── docs/                    # README, ARCHITECTURE, TROUBLESHOOTING, OIDC setup, endpoints
```

## Getting Started

### Prerequisites

- Go 1.27+
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

### Web client

Run the browser client separately from the Go API. See the [web setup guide](../web/README.md)
for Bun commands, the API origin, OIDC callback, and deployment details.

## Environment Variables

The server environment is documented in [`server/.env.example`](../server/.env.example).
It includes optional Google Maps departure estimates, National Rail commute
data, APNs notifications, and database and logging settings. The web client
also needs `NEXT_PUBLIC_FAMILY_HUB_API_URL`; set the server's `WEB_ORIGIN` to
the exact browser origin. See [OIDC setup](oidc-setup.md).

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

The Go API production stack uses Docker Compose:

```bash
cd server && docker compose -f docker-compose.prod.yml up -d
```

Migrations run automatically on startup from `server/internal/database/migrations/`.
The image is pure Go (no CGO required). Deploy the web client as a separate
service; see the [web deployment notes](../web/README.md#deployment).

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for the system diagram, request flow,
OIDC and display-token flows, and key design decisions. See [DESIGN.md](DESIGN.md)
for shared iOS, web, and wall-display guidance, including palette and typography.

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common server, iOS, and web connectivity issues.

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

The iOS app and web client depend on API capabilities provided by the matching
server version. Install server updates before clients that require new routes.

Run the optional provider check with public London locations:

```bash
cd server
RUN_HOUSEHOLD_PROVIDER_CHECK=1 go test ./internal/services -run TestHouseholdProvidersLive -count=1 -v
```

The check calls Open-Meteo. It also checks Google when `GOOGLE_MAPS_API_KEY` is
present in the test process environment. It never prints the key.
