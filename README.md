# Family Hub

A family organization hub for managing chores, meals, recipes, and calendars. Go + Templ + HTMX + Tailwind web app backed by SQLite, with a SwiftUI iOS client and a Home Assistant integration sharing the same REST API. Auth via OIDC (single public PKCE client).

**iOS app available on the App Store:** [Family Home Hub](https://apps.apple.com/gb/app/family-home-hub/id6761337807)

## Features

- **Chores** — Create and manage chores with recurring schedules (daily, weekly, monthly, custom), random assignment from eligible groups, and completion tracking with automatic overdue detection.
- **Dashboard** — At-a-glance view of today's chores, overdue items, upcoming events, completion statistics, and a household leaderboard.
- **Calendar** — Unified view of chores, family events, and external iCal subscriptions.
- **iCal Subscriptions** — Admin-managed subscriptions to external iCal feeds (e.g. school calendars, sports schedules) displayed alongside your own events.
- **Meal Planning** — Weekly meal planner organised by meal type (breakfast, lunch, dinner), with entries linked to the recipe library.
- **Recipes** — Manage a family recipe library with ingredient groups and cooking times, used to populate the meal plan. Supports importing recipes from a URL.
- **User Profiles** — Per-user avatar upload and profile management.
- **Admin Panel** — Manage users (promote/demote roles), chore categories, and API tokens. Includes database backup (download) and restore (upload).
- **REST API** — Session cookie or Bearer token; same surface for browser and iOS clients. See [`docs/endpoints.md`](docs/endpoints.md) for every route.

## Quick Start (Docker)

The fastest way to get running — three steps:

```bash
# 1. Copy the example env file and fill in your values
cp server/.env.example server/.env

# 2. Edit server/.env with your OIDC provider details and a random session secret
#    (see Environment Variables below for the full reference)

# 3. Start the stack
cd server && docker compose -f docker-compose.prod.yml up -d
```

Then open `http://localhost:8080` in your browser. **The first user to sign in is automatically made admin.**

> [!NOTE]
> You need an OIDC provider (e.g. Authelia, Keycloak, or Auth0) configured with a public PKCE client before the login flow will work. See [OIDC Client Setup](#oidc-client-setup) below.

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
- **Auth:** OIDC via a single public PKCE client (shared by web + iOS, no client secret)
- **iOS:** SwiftUI, MVVM
- **Dev tooling:** Air (hot reload), Docker, Make

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js (for Tailwind CSS)
- [Templ](https://templ.guide/) CLI
- An OIDC provider with public-client + PKCE support (e.g. Authelia, Keycloak, Auth0)

### Environment Variables

| Variable | Description | Default |
|---|---|---|
| `DATABASE_PATH` | Path to SQLite database file | `./data/family-hub.db` |
| `OIDC_ISSUER` | OpenID Connect issuer URL | *required* |
| `OIDC_CLIENT_ID` | OAuth2 client ID (single public PKCE client, shared by web + iOS) | *required* |
| `OIDC_REDIRECT_URL` | OAuth2 callback URL | *required* |
| `SESSION_SECRET` | Session encryption key | *required* |
| `BASE_URL` | Public base URL of the app | `http://localhost:8080` |
| `SESSION_ENCRYPTION_KEY` | 16 or 32 ASCII chars used to encrypt session cookies (e.g. `openssl rand -hex 16` → 32 chars). If unset cookies are signed-only. | — |
| `LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `PORT` | Server port | `8080` |
| `DEV_MODE` | Bypass OIDC and auto-login as a dev admin (**never set in production**) | `false` |

### OIDC Client Setup

Family Hub uses a **single public OIDC client** with PKCE for both the web UI
and the iOS app — no client secret, two redirect URIs.

<details>
<summary><strong>Authelia</strong></summary>

```yaml
identity_providers:
  oidc:
    clients:
      - client_id: family-hub
        client_name: Family Hub
        public: true
        token_endpoint_auth_method: none
        require_pkce: true
        pkce_challenge_method: S256
        redirect_uris:
          - https://hub.example.com/auth/callback    # web
          - familyhub://callback                      # iOS
        scopes: [openid, profile, email]
        grant_types: [authorization_code]
        response_types: [code]
        userinfo_signed_response_alg: none
```

</details>

<details>
<summary><strong>Keycloak</strong></summary>

1. Create a new client with **Client ID** `family-hub`.
2. Set **Client authentication** to `OFF` (public client).
3. Enable **Standard flow** only.
4. Under **Valid redirect URIs** add `https://hub.example.com/auth/callback` (and `familyhub://callback` for iOS).
5. Under **Advanced → Proof Key for Code Exchange Code Challenge Method** select `S256`.
6. Set `OIDC_ISSUER` to `https://<keycloak-host>/realms/<your-realm>`.

</details>

<details>
<summary><strong>Auth0</strong></summary>

1. Create a **Single Page Application** (SPA) — this gives you a public PKCE client.
2. Under **Allowed Callback URLs** add `https://hub.example.com/auth/callback`.
3. Under **Allowed Web Origins** add `https://hub.example.com`.
4. Set `OIDC_ISSUER` to `https://<your-tenant>.auth0.com/`.
5. Set `OIDC_CLIENT_ID` to the Auth0 Client ID shown in the application settings.

> [!NOTE]
> Auth0 free tier limits to 7,500 monthly active users — more than enough for a family.

</details>

The iOS app discovers the client ID and issuer at runtime via
`GET /api/client-config`, so you only configure those values on the server.

### First Login

> [!IMPORTANT]
> **The first user to sign in becomes the admin.** No pre-seeding or invitation step is required — just deploy, configure your OIDC provider, and log in. Once you're in, use the Admin panel to invite or promote other household members.

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

## Troubleshooting

### "redirect_uri_mismatch" or login loop

The redirect URI set in `OIDC_REDIRECT_URL` must exactly match the URI registered with your OIDC provider (including scheme, host, port, and path). For example:

```
OIDC_REDIRECT_URL=https://hub.example.com/auth/callback
```

### Blank page or 500 after OIDC callback

Check that `SESSION_SECRET` is set and non-empty. Without it the server cannot encode session cookies and will reject every login.

### `templ: command not found` during local build

The `templ` CLI must be on your `PATH`. Install it and add Go's bin directory:

```bash
go install github.com/a-h/templ/cmd/templ@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

### iOS app shows "Unable to connect"

The iOS app fetches `GET /api/client-config` to discover the OIDC issuer and client ID. Make sure `BASE_URL` is set to the publicly reachable URL of your server and that the endpoint is not blocked by a firewall or reverse proxy.

### First user is not admin

Only the very first account to complete the OIDC login flow is auto-promoted to admin. If you logged in with a different account by mistake, you can manually update the `role` column in the SQLite database, or use `DEV_MODE=true` temporarily to log in as a dev admin and promote your real account from the Admin panel.

## Security

Found a vulnerability? Please report it privately via GitHub's **Report a vulnerability** button on the Security tab. See [SECURITY.md](SECURITY.md) for details — please do not open a public issue for security problems.

## License

Family Hub is released under the [MIT License](LICENSE).
