# Family Hub

A full-stack family organization hub for managing chores, meals, recipes, and calendars. Built with Go, Templ, and Tailwind CSS, with OAuth2 authentication via OpenID Connect.

## Features

- **Chores** — Create and manage chores with recurring schedules (daily, weekly, monthly, custom), random assignment from eligible groups, and completion tracking with automatic overdue detection.
- **Dashboard** — At-a-glance view of today's chores, overdue items, upcoming events, completion statistics, and a household leaderboard.
- **Calendar** — Unified view of chores, family events, and external iCal subscriptions. Exports your family's schedule as an iCal feed for use in external calendar apps and Home Assistant.
- **iCal Subscriptions** — Admin-managed subscriptions to external iCal feeds (e.g. school calendars, sports schedules) displayed alongside your own events.
- **Family Events** — Schedule one-time or all-day events for the whole family.
- **Meal Planning** — Weekly meal planner organised by meal type (breakfast, lunch, dinner, snack), with entries linked to the recipe library.
- **Recipes** — Manage a family recipe library with ingredient groups and cooking times, used to populate the meal plan.
- **User Profiles** — Per-user avatar upload and profile management.
- **Admin Panel** — Manage users (promote/demote roles), chore categories, and API tokens. Includes database backup (download) and restore (upload).
- **REST API** — Token-authenticated API for third-party integrations, including Home Assistant sensor support.

## Monorepo Layout

| Component | Directory | Description |
|-----------|-----------|-------------|
| Go backend | [`server/`](server/) | HTTP server, templates, REST API |
| iOS app | [`ios/`](ios/) | SwiftUI native client |
| Home Assistant | [`home-assistant/`](home-assistant/) | Custom HACS integration |

## Tech Stack

- **Backend:** Go with [Chi](https://github.com/go-chi/chi) router
- **Templates:** [Templ](https://github.com/a-h/templ) (type-safe server-side rendering)
- **Database:** SQLite
- **Frontend:** Tailwind CSS + HTMX
- **Auth:** OAuth2 / OpenID Connect
- **iOS:** SwiftUI, MVVM
- **Dev tooling:** Air (hot reload), Docker, Make

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js (for Tailwind CSS)
- [Templ](https://templ.guide/) CLI
- An OIDC provider (e.g. Keycloak, Auth0)

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_PATH` | Path to SQLite database file | `./data/family-hub.db` |
| `OIDC_ISSUER` | OpenID Connect issuer URL | *required* |
| `OIDC_CLIENT_ID` | OAuth2 client ID (single public PKCE client, shared by web + iOS) | *required* |
| `OIDC_REDIRECT_URL` | OAuth2 callback URL | *required* |
| `SESSION_SECRET` | Session encryption key | *required* |
| `BASE_URL` | Public base URL of the app | `http://localhost:8080` |
| `HA_API_TOKEN` | Home Assistant API token (for sensor endpoint) | — |
| `LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `PORT` | Server port | `8080` |

### Run with Docker (recommended)

Run from the `server/` directory:

**Development** (with hot reload and source mounting):

```bash
cd server && docker compose up --build
```

**Production:**

```bash
cd server && docker compose -f docker-compose.prod.yml up --build
```

### Run Locally

```bash
cd server

# Install dependencies
go mod download
npm install

# Build and run
make run
```

For development with hot reload:

```bash
cd server && make dev
```

## Make Targets

Run from `server/`:

| Command | Description |
|---|---|
| `make build` | Regenerate templates, rebuild CSS, compile Go binary |
| `make run` | Build and run the application |
| `make dev` | Run with Air hot reload |
| `make templ` | Regenerate Templ templates |
| `make css` | Rebuild Tailwind CSS |
| `make test` | Run all tests |
| `make test-coverage` | Generate HTML coverage report |
| `make docker-dev` | Start dev environment via Docker Compose |
| `make docker-prod` | Start production environment via Docker Compose |
| `make clean` | Remove build artifacts |

## Project Structure

```
├── server/                  # Go backend
│   ├── main.go              # Entry point
│   ├── internal/
│   │   ├── config/          # Environment configuration
│   │   ├── database/        # SQLite setup and migrations
│   │   ├── models/          # Data models
│   │   ├── repository/      # Data access layer
│   │   ├── services/        # Business logic
│   │   ├── handlers/        # HTTP handlers
│   │   ├── middleware/      # Auth middleware
│   │   └── server/          # Router and server setup
│   ├── templates/           # Templ page and layout templates
│   ├── static/              # CSS and JS assets
│   ├── Dockerfile           # Production image
│   ├── Dockerfile.dev       # Development image
│   ├── docker-compose.yml   # Dev compose config
│   └── docker-compose.prod.yml  # Production compose config
├── ios/                     # SwiftUI iOS app
├── home-assistant/          # Home Assistant HACS integration
└── data/                    # Runtime data (SQLite DB, gitignored)
```

## Testing

```bash
cd server && make test
```

Generate a coverage report:

```bash
cd server && make test-coverage
```
