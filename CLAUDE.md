# Family Hub — Developer Context

Family organization hub for managing chores, meals, recipes, and calendars.
Go backend with server-side rendering via Templ + HTMX, SQLite database.

## Monorepo Layout

| Component | Directory | Description |
|-----------|-----------|-------------|
| Go backend | `server/` | HTTP server, templates, API |
| iOS app | `ios/` | SwiftUI native client |
| Home Assistant | `home-assistant/` | Custom HACS integration |

## Architecture

Request flow: HTTP → Chi router → Middleware → Handler → Service → Repository → SQLite

```
server/
├── internal/
│   ├── config/          # Env var parsing
│   ├── database/        # DB connection + migration runner
│   ├── models/          # Domain types (Chore, User, Recipe, etc.)
│   ├── repository/      # Interfaces + SQLite implementations
│   ├── services/        # Business logic (recurrence, chore assignment, iCal fetch)
│   ├── handlers/        # HTTP handlers (one file per feature area)
│   ├── middleware/       # Auth + admin enforcement
│   └── server/          # Chi router wiring
templates/           (under server/)
├── layouts/         # Base page layout
├── pages/           # Full-page templ files
└── components/      # Reusable UI components
```

## Key Patterns

**Repositories** — Each feature area has an interface in `internal/repository/` with a SQLite
implementation. Tests use in-memory SQLite (`:memory:`).

**Services** — Business logic lives in `internal/services/`. Handlers call services; services
call repositories. Direct repository access from handlers is avoided.

**Templ** — Type-safe templates compiled to Go. Run `templ generate` after editing `.templ`
files. The generated `*_templ.go` files are committed but should not be edited directly.

**HTMX** — Handlers return either full pages or HTML fragments depending on whether the
request is an HTMX partial. Fragments are returned for `HX-Request` headers.

**Auth** — OIDC via Authentik. Session stored as an encrypted cookie. The `RequireAuth`
middleware gates all routes except `/login`, `/auth/callback`, `/health`, and `/ical`.
The `RequireAdmin` middleware gates admin-only routes.

## Feature Map

| Area | Handler | Service | Notes |
|------|---------|---------|-------|
| Chores | `handlers/chores.go` | `services/chores.go`, `services/recurrence.go` | Recurrence: daily, weekly, monthly, custom cron |
| Calendar | `handlers/calendar.go` | — | Unified view: chores + events + iCal subscriptions |
| iCal export | `handlers/ical.go` | — | `GET /ical`, token-scoped (`ical` scope) |
| iCal import | `handlers/ical_subscriptions.go` | `services/ical_fetcher.go` | Admin-managed external feeds |
| Meals | `handlers/meals.go` | — | Weekly planner, backed by `meal_plans` table |
| Recipes | `handlers/recipes.go` | — | Ingredient groups, cooking times, linked to meals |
| Dashboard | `handlers/dashboard.go` | — | Stats + leaderboard |
| Admin | `handlers/admin.go` | — | Users, categories, token CRUD |
| REST API | `handlers/api.go` | — | Token-auth, chores/users/categories/dashboard |
| Home Assistant | `handlers/ical.go` (`HASensorHandler`) | — | Sensor data endpoint, co-located with iCal handler |

## Gotchas

- `templ generate` requires `$(go env GOPATH)/bin` in `PATH`
- Multi-return Go functions cannot be called directly inside templ format strings — assign to a variable first
- `modernc.org/sqlite` is pure Go (no CGO). Use `:memory:` for in-memory DBs in tests
- Migrations run automatically on startup from `internal/database/migrations/`

## Testing

- Table-driven tests throughout
- Repository tests use in-memory SQLite for isolation
- Prefer fakes over mocks; fake implementations live alongside the interface
- Run tests: `cd server && make test`

## Dev Workflow

```bash
cd server && make dev          # Air hot reload (watches Go + templ changes)
docker compose up              # Full stack with Docker (run from server/)
cd server && make templ        # Regenerate templ files manually
cd server && make css          # Rebuild Tailwind CSS
```
