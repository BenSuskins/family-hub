# Monorepo Reorganization Design

**Date:** 2026-03-14
**Status:** Approved

## Overview

Reorganize the Family Hub monorepo from a Go-backend-at-root layout into a flat peer-directory structure with one directory per component. Add per-component CI/CD pipelines.

## Directory Structure

```
/
├── server/                         # Go backend (moved from root)
│   ├── internal/
│   ├── templates/
│   ├── static/
│   ├── main.go
│   ├── go.mod / go.sum
│   ├── Makefile
│   ├── Dockerfile
│   ├── Dockerfile.dev
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── .air.toml
│   ├── package.json
│   ├── tailwind.config.js
│   └── .env
├── ios/                            # iOS app (unchanged; currently ios/FamilyHub/)
│   └── FamilyHub/
├── home-assistant/                 # HA integration (moved from /custom_components/)
│   └── custom_components/
│       └── family_hub/
├── data/                           # Runtime data — SQLite DB, stays at root
├── docs/                           # Shared documentation
├── .github/
│   └── workflows/
│       ├── server.yml
│       ├── ios.yml
│       └── home-assistant.yml
├── hacs.json                       # Must stay at root (HACS requirement)
├── README.md
└── .gitignore
```

### What moves

| From | To |
|------|----|
| `internal/`, `templates/`, `static/`, `main.go`, `go.mod`, `go.sum` | `server/` |
| `Makefile`, `Dockerfile`, `Dockerfile.dev`, `docker-compose*.yml` | `server/` |
| `.air.toml`, `package.json`, `tailwind.config.js`, `.env` | `server/` |
| `custom_components/family_hub/` | `home-assistant/custom_components/family_hub/` |

### What is deleted

| Path | Reason |
|------|--------|
| `.github/workflows/build-and-push.yml` | Replaced by `server.yml` |
| `migrations/` (root-level) | Duplicate of `internal/database/migrations/`; the copy inside `server/` is authoritative |

### What stays at root

- `hacs.json` — HACS requires this at the repo root
- `data/` — runtime data, Docker volume mount target, gitignored
- `docs/` — shared across components
- `README.md`, `.gitignore`

## CI/CD Pipelines

### `server.yml` — triggers on `push`/`pr` to paths `server/**`

1. `go test ./...` — uses in-memory SQLite, no external services needed
2. On push to `main`: `docker build` + push to registry

### `ios.yml` — triggers on `push`/`pr` to paths `ios/**`

1. `macos-latest` runner
2. `xcodebuild test` targeting simulator
3. No signing, no TestFlight deployment

### `home-assistant.yml` — triggers on `push`/`pr` to paths `home-assistant/**`

1. `ruff check` for Python linting
2. `pytest` (passes vacuously until tests are added)
3. On push to `main`:
   - Auto-tag using CalVer + GitHub run number: `vYYYY.MM.DD.<run_number>`
   - Zip `home-assistant/custom_components/family_hub/`
   - Attach `family_hub.zip` to GitHub release

### `hacs.json` update

```json
{
  "name": "Family Hub",
  "render_readme": true,
  "homeassistant": "2024.1.0",
  "zip_release": true,
  "filename": "family_hub.zip"
}
```

## Key Constraints

- **HACS**: `hacs.json` must remain at repo root. The `custom_components/` directory is moved inside `home-assistant/` and HACS is configured to use zip releases instead of direct directory discovery.
- **HACS zip structure**: The zip must contain `custom_components/family_hub/` at its root. The CI step must `cd home-assistant && zip -r family_hub.zip custom_components/family_hub/` so the internal paths are correct. HACS does not require `content_in_root` to be set; zip structure alone determines extraction.
- **Auto-tag scope**: The tag is applied to the full repo commit (not a component-specific ref). This is fine — HACS only fetches the attached zip artifact, not the repo contents at that tag.
- **Docker build context**: Dockerfile and docker-compose paths update to reference `./server/` as the build context. The dev bind mount changes from `.:/app` (entire repo root) to `./server:/app` (server subtree only), which is the correct scope for the Go server container.
- **data/ volume**: Docker Compose (now in `server/`) mounts the data directory using a path relative to `server/`: `../data:/app/data`. Update `docker-compose.yml` volume paths accordingly.
- **.air.toml**: After moving to `server/`, update any absolute paths (e.g. `/Users/bensuskins/go/bin/templ`) to use `$(go env GOPATH)/bin/templ` so the config works on any machine and in CI.
- **CLAUDE.md**: Update path references to reflect new structure.

## What Is Not Changing

- Internal architecture of each component
- Database schema and migrations
- Authentication flow
- HTMX/templ frontend approach
- iOS MVVM structure
