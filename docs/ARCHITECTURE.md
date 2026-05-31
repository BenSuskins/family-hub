# Architecture

> System design, components, and request flow for Family Hub.

## Overview

A single Go binary serves both a server-rendered web UI (Templ + HTMX +
Tailwind) and a JSON REST API consumed by a SwiftUI iOS app and a Home
Assistant HACS integration. All three clients share the same OIDC auth flow
via a public PKCE client. SQLite is the system of record; the iOS app caches
in SwiftData but has no offline write path.

## Diagram

```mermaid
flowchart LR
    Web[Web Browser] -->|HTMX, session cookie| Server
    iOS[iOS App] -->|JSON + Bearer| Server
    HA[Home Assistant] -->|JSON + Bearer| Server
    OIDC[OIDC Provider<br/>Authelia / Keycloak / Auth0] <-.->|PKCE| Server
    Web -.->|public PKCE| OIDC
    iOS -.->|public PKCE| OIDC
    Server --> DB[(SQLite)]
    Server -.->|fetch| External[External iCal / Recipe URLs]
```

## Components

| Component | Responsibility | Location |
|-----------|---------------|----------|
| HTTP server | Chi router, middleware, handlers | `server/internal/server/`, `server/internal/handlers/` |
| Services | Business logic (recurrence, chore assignment, iCal fetch, recipe extraction, SSRF guard) | `server/internal/services/` |
| Repositories | Data access (one per feature area, interface + SQLite impl) | `server/internal/repository/` |
| Templates | Templ pages, layouts, components (compiled to Go) | `server/templates/` |
| Migrations | Auto-run on startup | `server/internal/database/migrations/` |
| iOS app | SwiftUI client (SwiftData cache, no offline writes) | `ios/` |
| HA integration | Custom HACS component | `home-assistant/` |

## Request Flow

```mermaid
sequenceDiagram
    participant C as Client (Web/iOS/HA)
    participant R as Chi Router
    participant M as Middleware (RequireUser)
    participant H as Handler
    participant S as Service
    participant Repo as Repository
    participant DB as SQLite
    C->>R: HTTP request
    R->>M: dispatch
    M->>M: session cookie OR Bearer token<br/>→ UserContextKey
    M->>H: authenticated request
    H->>S: domain operation
    S->>Repo: read/write
    Repo->>DB: SQL
    DB-->>Repo: rows
    Repo-->>S: domain model
    S-->>H: result
    H-->>C: HTML page, HTMX fragment, or JSON
```

Handlers return either a full page or an HTML fragment depending on the
`HX-Request` header — same handler, different render.

## Auth Flow

A single public OIDC client is shared across web and iOS. Two post-login
mechanisms converge on the same context key:

```mermaid
flowchart TB
    subgraph Web
        Web1[Cookie-bearing request] --> RU[RequireUser middleware]
    end
    subgraph Mobile
        iOS1[OIDC bearer token] --> Exchange[POST /api/auth/exchange]
        Exchange --> APIToken[API token]
        APIToken --> RU
    end
    RU --> Ctx[UserContextKey]
    Ctx --> Handlers
    Handlers -. + RequireAdmin .-> AdminHandlers
```

Public routes (no auth): `/health`, `/static/*`, `/api/client-config`,
`/login`, `/auth/callback`, `/logout`. Mobile obtains an API token via
`POST /api/auth/exchange` (one-time OIDC bearer). There is no onboarding
flow and no iCal feed export.

## SSRF Guard

Any handler that fetches an external URL on a user's behalf (iCal
subscriptions, recipe URL import) routes through
`services.ValidateExternalURL` / `services.NewSafeHTTPClient` in
`services/safenet.go`. This blocks private IP ranges and non-HTTP(S) schemes.

## Data Model

See `server/internal/models/` for the full domain. Headline shape:

```mermaid
erDiagram
    USER ||--o{ CHORE_ASSIGNMENT : completes
    USER ||--o{ EVENT : owns
    USER ||--o{ MEAL_PLAN : owns
    CHORE ||--o{ CHORE_ASSIGNMENT : produces
    CHORE }o--|| CATEGORY : in
    RECIPE ||--o{ INGREDIENT_GROUP : has
    MEAL_PLAN ||--o{ MEAL : contains
    MEAL }o--|| RECIPE : references
    USER {
        uuid id
        string oidc_sub
        string role
    }
    CHORE {
        uuid id
        string title
        string recurrence
    }
```

## External Dependencies

| Service | Purpose | Failure mode |
|---------|---------|--------------|
| OIDC provider | Auth for all clients | Login blocked; existing sessions/tokens unaffected |
| External iCal feeds | Calendar subscriptions | Subscription marked stale; existing events remain |
| Recipe URLs | Recipe import (JSON-LD + HTML fallback) | Import returns an error to the user |

## Key Decisions

- **One binary, three clients.** Web, iOS, and HA all hit the same REST surface — no separate "mobile API" — so feature work doesn't fan out.
- **Public PKCE OIDC client for everything.** No client secrets to manage; same client ID for web and iOS, distinguished only by redirect URI.
- **Pure-Go SQLite.** `modernc.org/sqlite` removes the CGO toolchain dependency for builds and tests; `:memory:` in tests.
- **No offline writes on iOS.** SwiftData is a cache; mutations require connectivity. Removes a class of sync bugs at the cost of usability when offline.
- **Repository → Service → Handler.** Handlers never reach a repository directly; this makes business logic testable with fakes.
- **Migrations run on startup.** No separate migration step in deployment; SQLite is a file so it's safe to gate behind a server bootstrap.
