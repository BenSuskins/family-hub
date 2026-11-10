# Family Hub — Developer Context

Family organization hub for chores, meals, recipes, calendars, shopping, and
inventory. The Go server is a JSON and media API for the native Apple clients.

## Endpoint Reference

`docs/endpoints.md` is the canonical list of every HTTP route (use case,
callers, security, runnable `curl` example). Whenever you add, remove, rename,
or change the auth or behavior of a route in
`server/internal/server/server.go` or a handler, update that file in the same
change.

## Monorepo Layout

| Component | Directory | Description |
|-----------|-----------|-------------|
| Go API server | `server/` | HTTP API, business logic, SQLite |
| iOS app | `ios/` | SwiftUI native client |

### iOS targets

| Target | Directory | Notes |
|--------|-----------|-------|
| `FamilyHub` | `ios/FamilyHub/FamilyHub/` | The app |
| `FamilyHubWidgets` | `ios/FamilyHub/FamilyHubWidgets/` | WidgetKit extension |
| `ShareExtension` | `ios/FamilyHub/ShareExtension/` | Recipe share sheet |
| `FamilyHubWatch Watch App` | `ios/FamilyHub/FamilyHubWatch Watch App/` | Cook mode on the wrist |
| `FamilyHubKit` | `ios/FamilyHubKit/` | Shared models and networking, Foundation-only for Linux CI |

Only the app can run the OIDC flow. It mirrors the server URL and long-lived
API token into the shared app group so extensions can read them. See
`SharedContainer` in FamilyHubKit. Put new extension logic in FamilyHubKit.

## Architecture

Chores, meals, inventory, shopping, scratchpad, and home settings flow through feature application operations to SQLite adapters. Other features can call repositories from handlers.

```
server/
├── internal/
│   ├── application/     # Feature operations and storage ports
│   ├── domain/          # Feature rules
│   ├── config/          # Environment parsing
│   ├── database/        # DB connection and migration runner
│   ├── models/          # Domain types
│   ├── repository/      # Interfaces and SQLite implementations
│   ├── services/        # Business logic and external URL guards
│   ├── handlers/        # JSON and binary API handlers
│   ├── middleware/      # Bearer auth and admin enforcement
│   ├── server/          # Chi router wiring
│   └── testutil/        # Shared test helpers
```

## Key Patterns

**Repositories** — SQLite adapters live in `internal/repository/`. Migrated feature ports
live in their application packages; other repository interfaces remain beside
their implementations. Repository tests use in-memory SQLite (`:memory:`).

**Chores** — Chore operations and storage ports live in
`internal/application/chores/`. Chore rules live in `internal/domain/chores/`.
The SQLite adapter implements the ports. Shared adapter contracts live in
`internal/contracttests/`.

**Household data** — Meal, inventory, shopping, scratchpad, and home settings
operations live in `internal/application/`. Their rules live in the matching
`internal/domain/` packages when needed. Shopping restock and inventory writes
commit together. Inventory stock changes use a compare-and-swap update. Shared
stock and shopping contracts live in `internal/contracttests/`.

**Services** — Other business logic lives in `internal/services/`. Avoid direct
repository access from handlers when logic belongs in a service.

**Authentication** — `POST /api/auth/exchange` exchanges a one-time OIDC bearer
token for a Family Hub API token. All other authenticated routes require
`Authorization: Bearer <token>`. Admin routes also require the admin role.
The API does not issue web sessions. An opt-in setting serves the API explorer.

**Errors and media** — API errors use a JSON object with an `error` field.
Image and database backup routes return binary data on success.

**SSRF guard** — Any handler that fetches an external URL for a user must use
`services.ValidateExternalURL` and `services.NewSafeHTTPClient` in
`services/safenet.go`.

## Feature Map

| Area | Handler | Service | Notes |
|------|---------|---------|-------|
| Chores | `handlers/api.go` | `application/chores/`, `domain/chores/` | Recurrence: daily, weekly, monthly, custom cron |
| Calendar | `handlers/api.go`, `handlers/calendar_subscriptions_api.go` | `services/ical_fetcher.go` | Unified events and admin-managed iCal feeds |
| Meals | `handlers/api.go` | `application/meals/` | Weekly planner backed by `meal_plans` |
| Recipes | `handlers/api.go` | `services/recipe_extractor.go` | URL extraction and API CRUD; share sheet uses JSON routes |
| Inventory | `handlers/api_inventory.go` | `application/inventory/`, `domain/inventory/` | Areas, items, stock |
| Shopping | `handlers/shopping.go` | `application/shopping/`, `domain/shopping/` | Shared shopping list |
| Scratchpad | `handlers/scratchpad.go` | `application/scratchpad/`, `domain/scratchpad/` | Shared drawing |
| Home settings | `handlers/household.go` | `application/household/`, `domain/household/` | Location and time zone |
| Admin | `handlers/api.go`, `handlers/backup.go` | — | Users, categories, tokens, settings, backup/restore |

## Gotchas

- `modernc.org/sqlite` is pure Go (no CGO). Use `:memory:` for isolated tests.
- Database migrations run automatically on startup from
  `internal/database/migrations/`.
- OIDC tokens are accepted only by `/api/auth/exchange`; other API routes need
  the long-lived Family Hub Bearer token.

## Testing

- Tests are table-driven where practical.
- Repository tests use in-memory SQLite.
- Prefer fakes over mocks; fake implementations live alongside the interface.
- Run server tests: `cd server && go test ./...`.
- Run FamilyHubKit API contract tests from the iOS project’s documented test
  workflow.

## Dev Workflow

```bash
cd server && make run
cd server && make test
cd server && make docker-dev
cd server && make docker-prod
```
