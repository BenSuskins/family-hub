# Family Hub — Developer Context

Family organization hub for managing chores, meals, recipes, and calendars.
Go backend with server-side rendering via Templ + HTMX, SQLite database.

## Endpoint Reference

`docs/endpoints.md` is the canonical list of every HTTP route (usecase, callers,
security, runnable `curl` example). **Whenever you add, remove, rename, or change
the auth/behavior of a route in `server/internal/server/server.go` or any handler,
update `docs/endpoints.md` in the same change.** Do not let it drift.

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

**Auth** — OIDC via Authentik. Two flows: *authed user* (session cookie OR
Bearer API token, unified in `RequireUser`) and *admin user* (`+ RequireAdmin`).
Both mechanisms populate the same `UserContextKey`, so handlers are
mechanism-agnostic. Public routes: `/health`, `/static/*`, `/api/client-config`,
`/login`, `/auth/callback`, `/logout`. Mobile clients obtain an API token via
`POST /api/auth/exchange` (one-time OIDC bearer). No onboarding flow; no iCal
feed export.

## Feature Map

| Area | Handler | Service | Notes |
|------|---------|---------|-------|
| Chores | `handlers/chores.go` | `services/chores.go`, `services/recurrence.go` | Recurrence: daily, weekly, monthly, custom cron |
| Calendar | `handlers/calendar.go` | — | Unified view: chores + events + iCal subscriptions |
| iCal import | `handlers/ical_subscriptions.go` | `services/ical_fetcher.go` | Admin-managed external feeds |
| Meals | `handlers/meals.go` | — | Weekly planner, backed by `meal_plans` table |
| Recipes | `handlers/recipes.go` | — | Ingredient groups, cooking times, linked to meals |
| Dashboard | `handlers/dashboard.go` | — | Stats + leaderboard |
| Admin | `handlers/admin.go` | — | Users, categories, token CRUD |
| REST API | `handlers/api.go` | — | Session or Bearer; chores/users/categories/dashboard/recipes/meals |

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
