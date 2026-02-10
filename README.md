# Family Hub

A full-stack family organization hub that lets families manage chores, events, and calendars. Built with Go, Templ, and Tailwind CSS, with OAuth2 authentication via OpenID Connect.

## Features

- **Chores** -- Create and manage chores with recurring schedules (daily, weekly, monthly, custom), random assignment from groups, and completion tracking with automatic overdue detection.
- **Calendar** -- View chores and family events in a unified calendar, exportable as an iCal feed for use in external calendar apps and Home Assistant.
- **Family Events** -- Schedule one-time or all-day events for the whole family.
- **Dashboard** -- At-a-glance view of today's chores, overdue items, upcoming events, and completion statistics.
- **Admin Panel** -- Manage users (promote/demote roles), categories, and API tokens.
- **REST API** -- Token-authenticated API for third-party integrations including Home Assistant sensor support.

## Tech Stack

- **Backend:** Go with [Chi](https://github.com/go-chi/chi) router
- **Templates:** [Templ](https://github.com/a-h/templ) (type-safe server-side rendering)
- **Database:** SQLite
- **Frontend:** Tailwind CSS
- **Auth:** OAuth2 / OpenID Connect
- **Dev tooling:** Air (hot reload), Docker, Make

## Getting Started

### Prerequisites

- Go 1.25+
- Node.js (for Tailwind CSS)
- [Templ](https://templ.guide/) CLI
- An OIDC provider (e.g. Authentik, Keycloak, Auth0)

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_PATH` | Path to SQLite database file | `./data/family-hub.db` |
| `OIDC_ISSUER` | OpenID Connect issuer URL | *required* |
| `OIDC_CLIENT_ID` | OAuth2 client ID | *required* |
| `OIDC_CLIENT_SECRET` | OAuth2 client secret | *required* |
| `OIDC_REDIRECT_URL` | OAuth2 callback URL | `http://localhost:8080/auth/callback` |
| `SESSION_SECRET` | Session encryption key | *required* |
| `HA_API_TOKEN` | Home Assistant API token | *optional* |
| `LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `PORT` | Server port | `8080` |

### Run with Docker (recommended)

**Development** (with hot reload and source mounting):

```bash
docker compose up --build
```

**Production:**

```bash
docker compose -f docker-compose.prod.yml up --build
```

### Run Locally

```bash
# Install dependencies
go mod download
npm install

# Build and run
make run
```

For development with hot reload:

```bash
make dev
```

## Make Targets

| Command | Description |
|---|---|
| `make build` | Generate templates, build CSS, compile Go binary |
| `make run` | Build and run the application |
| `make dev` | Run with Air hot reload |
| `make test` | Run all tests |
| `make test-coverage` | Generate HTML coverage report |
| `make docker-dev` | Start dev environment via Docker Compose |
| `make docker-prod` | Start production environment via Docker Compose |
| `make clean` | Remove build artifacts |

## Project Structure

```
├── main.go                  # Entry point
├── internal/
│   ├── config/              # Environment configuration
│   ├── database/            # SQLite setup and migrations
│   ├── models/              # Data models
│   ├── repository/          # Data access layer
│   ├── services/            # Business logic
│   ├── handlers/            # HTTP handlers
│   ├── middleware/           # Auth middleware
│   └── server/              # Router and server setup
├── templates/               # Templ page and layout templates
├── static/                  # CSS and JS assets
├── Dockerfile               # Production image
├── Dockerfile.dev           # Development image
├── docker-compose.yml       # Dev compose config
└── docker-compose.prod.yml  # Production compose config
```

## Testing

```bash
make test
```

Generate a coverage report:

```bash
make test-coverage
```
