# Monorepo Reorganization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reorganize the repo from Go-backend-at-root to flat peer directories (`server/`, `ios/`, `home-assistant/`) with per-component CI/CD pipelines.

**Architecture:** File moves using `git mv` to preserve history, followed by config updates to fix paths, then new CI workflows. No code logic changes — this is pure structure.

**Tech Stack:** Git, GitHub Actions, Go, Swift/Xcode, Python/ruff, Docker, HACS

**Spec:** `docs/superpowers/specs/2026-03-14-monorepo-reorg-design.md`

---

## Chunk 1: Move Go backend to server/

### Task 1: Move Go source files and build config

**Files:**
- Move: `internal/` → `server/internal/`
- Move: `templates/` → `server/templates/`
- Move: `static/` → `server/static/`
- Move: `main.go` → `server/main.go`
- Move: `go.mod` → `server/go.mod`
- Move: `go.sum` → `server/go.sum`
- Move: `Makefile` → `server/Makefile`
- Move: `Dockerfile` → `server/Dockerfile`
- Move: `Dockerfile.dev` → `server/Dockerfile.dev`
- Move: `docker-compose.yml` → `server/docker-compose.yml`
- Move: `docker-compose.prod.yml` → `server/docker-compose.prod.yml`
- Move: `.air.toml` → `server/.air.toml`
- Move: `package.json` → `server/package.json`
- Move: `package-lock.json` → `server/package-lock.json`
- Move: `tailwind.config.js` → `server/tailwind.config.js`
- Move: `API.md` → `server/API.md`
- Move: `.env` → `server/.env` (gitignored — use `mv`, not `git mv`)
- `todo.md` stays at root (project-wide, not server-specific)
- `node_modules/` is NOT moved — it is gitignored and will be reinstalled via `npm install`

- [ ] **Step 1: Create server/ and move tracked files using git mv**

```bash
mkdir server
git mv internal server/internal
git mv templates server/templates
git mv static server/static
git mv main.go server/main.go
git mv go.mod server/go.mod
git mv go.sum server/go.sum
git mv Makefile server/Makefile
git mv Dockerfile server/Dockerfile
git mv Dockerfile.dev server/Dockerfile.dev
git mv docker-compose.yml server/docker-compose.yml
git mv docker-compose.prod.yml server/docker-compose.prod.yml
git mv .air.toml server/.air.toml
git mv package.json server/package.json
git mv package-lock.json server/package-lock.json
git mv tailwind.config.js server/tailwind.config.js
git mv API.md server/API.md
```

- [ ] **Step 2: Move gitignored files that are still needed at runtime**

`.env` is gitignored so `git mv` won't work. Move it with the shell:

```bash
mv .env server/.env
```

- [ ] **Step 3: Verify git sees renames (not delete+add)**

```bash
git status
```

Expected: tracked files show `renamed:` not `deleted:` + `new file:`. Gitignored files (`.env`, `node_modules/`, `bin/`, etc.) will not appear in `git status` — that is correct.

- [ ] **Step 4: Commit the move**

```bash
git add -A
git commit -m "refactor: move Go backend to server/"
```

---

### Task 2: Update .air.toml to remove hardcoded path

**Files:**
- Modify: `server/.air.toml`

The current `cmd` uses an absolute path `/Users/bensuskins/go/bin/templ` which breaks on other machines. Air executes `cmd` via `sh -c`, so shell substitutions like `$(go env GOPATH)` expand correctly at runtime — this is safe to use.

- [ ] **Step 1: Update the cmd in server/.air.toml**

Replace the `cmd` line:

```toml
[build]
  cmd = "$(go env GOPATH)/bin/templ generate && npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify && go build -o ./tmp/main ."
  bin = "./tmp/main"
  delay = 1000
  exclude_dir = ["tmp", "node_modules", "data", ".git"]
  exclude_regex = ["_test\\.go$", "_templ\\.go$"]
  include_ext = ["go", "templ", "css", "js"]
  kill_delay = "0s"

[log]
  time = false

[misc]
  clean_on_exit = true
```

- [ ] **Step 2: Commit**

```bash
git add server/.air.toml
git commit -m "fix: use go env GOPATH in .air.toml instead of absolute path"
```

---

### Task 3: Fix Docker volume paths in docker-compose files

**Files:**
- Modify: `server/docker-compose.yml`
- Modify: `server/docker-compose.prod.yml`

The `data/` directory stays at the repo root. Docker Compose files moved to `server/`, so the relative path changes from `./data` to `../data`. Docker Compose resolves `.` relative to the compose file's own location — so after the move, `.:/app` mounts `server/` into the container, and `../data:/data` mounts the repo-root `data/` directory. Both are correct.

- [ ] **Step 1: Update server/docker-compose.yml**

Change the volumes section from:
```yaml
    volumes:
      - .:/app
      - ./data:/data
      - go-cache:/root/go
```
To:
```yaml
    volumes:
      - .:/app
      - ../data:/data
      - go-cache:/root/go
```

- [ ] **Step 2: Update server/docker-compose.prod.yml**

Change the volumes section from:
```yaml
    volumes:
      - ./data:/data
```
To:
```yaml
    volumes:
      - ../data:/data
```

- [ ] **Step 3: Verify docker build still works (dry run)**

Run from `server/` after `.env` has been moved there (Task 1 Step 2):

```bash
cd server && docker compose config
```

Expected: valid YAML output with no errors. The `data` volume path should show the absolute path to the repo-root `data/` directory. Warnings about unset optional env vars (e.g. `OIDC_ISSUER`) are fine — the prod compose file requires them but they're not needed for config validation.

- [ ] **Step 4: Commit**

```bash
git add server/docker-compose.yml server/docker-compose.prod.yml
git commit -m "fix: update data volume path in docker-compose after server/ move"
```

---

### Task 4: Verify Go server still builds and tests pass

- [ ] **Step 1: Install dependencies from server/**

```bash
cd server && npm install
```

Expected: node_modules populated (or already present).

- [ ] **Step 2: Regenerate templ files**

```bash
cd server && export PATH=$PATH:$(go env GOPATH)/bin && templ generate
```

Expected: `*_templ.go` files regenerated with no errors.

- [ ] **Step 3: Run tests**

```bash
cd server && go test ./...
```

Expected: all tests pass. If `go` complains about missing module, ensure you're in `server/` where `go.mod` now lives.

- [ ] **Step 4: Build binary**

```bash
cd server && go build -o /dev/null .
```

Expected: exits 0, no output.

---

## Chunk 2: Move Home Assistant integration and cleanup

### Task 5: Move HA integration to home-assistant/

**Files:**
- Move: `custom_components/family_hub/` → `home-assistant/custom_components/family_hub/`
- Delete: `migrations/` (root-level duplicate of `server/internal/database/migrations/`)

- [ ] **Step 1: Confirm root migrations/ is empty and safe to delete**

```bash
ls migrations/
```

Expected: empty output (no files). The root `migrations/` directory is an empty placeholder — the real migrations live in `server/internal/database/migrations/`. If any files are listed, stop and investigate before deleting.

- [ ] **Step 2: Move HA component**

```bash
mkdir -p home-assistant
git mv custom_components home-assistant/custom_components
```

- [ ] **Step 3: Delete root migrations/ directory**

```bash
git rm -r migrations/
```

- [ ] **Step 4: Verify structure**

```bash
ls home-assistant/custom_components/family_hub/
```

Expected: `__init__.py`, `api.py`, `config_flow.py`, `const.py`, `coordinator.py`, `sensor.py`, `manifest.json`, `strings.json`, `translations/`

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: move HA integration to home-assistant/, delete duplicate migrations/"
```

---

### Task 6: Update hacs.json for zip release

**Files:**
- Modify: `hacs.json`

HACS needs `zip_release: true` and `filename` so it fetches the attached zip artifact instead of looking for `custom_components/` at the repo root.

- [ ] **Step 1: Update hacs.json**

```json
{
  "name": "Family Hub",
  "render_readme": true,
  "homeassistant": "2024.1.0",
  "zip_release": true,
  "filename": "family_hub.zip"
}
```

- [ ] **Step 2: Commit**

```bash
git add hacs.json
git commit -m "feat: configure HACS to use zip releases for home-assistant/ subdirectory"
```

---

## Chunk 3: CI/CD Workflows

### Task 7: Delete old workflow, create server.yml

**Files:**
- Delete: `.github/workflows/build-and-push.yml`
- Create: `.github/workflows/server.yml`

The new workflow triggers only on changes to `server/**`, runs templ+css setup before tests, then builds and pushes the Docker image on `main` pushes.

- [ ] **Step 1: Delete old workflow**

```bash
git rm .github/workflows/build-and-push.yml
```

- [ ] **Step 2: Create .github/workflows/server.yml**

```yaml
name: Server CI

on:
  push:
    branches: [main]
    paths: ['server/**']
  pull_request:
    paths: ['server/**']

jobs:
  test:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: server

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
          cache-dependency-path: server/go.sum

      - uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Setup
        run: |
          export PATH="$PATH:$(go env GOPATH)/bin"
          go mod download
          npm install
          go install github.com/a-h/templ/cmd/templ@latest
          templ generate
          npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify

      - name: Test
        run: go test ./...

  docker:
    if: github.ref == 'refs/heads/main'
    needs: test
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Build and push
        run: |
          COMMIT_SHA=${GITHUB_SHA::7}
          IMAGE_NAME=ghcr.io/bensuskins/family-hub
          docker build -t $IMAGE_NAME:latest -t $IMAGE_NAME:$COMMIT_SHA server/
          docker push $IMAGE_NAME:latest
          docker push $IMAGE_NAME:$COMMIT_SHA
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/
git commit -m "ci: replace build-and-push.yml with server.yml scoped to server/**"
```

---

### Task 8: Create ios.yml

**Files:**
- Create: `.github/workflows/ios.yml`

Triggers on changes to `ios/**`. Runs on `macos-latest`. Uses `xcodebuild test` against the `FamilyHub` scheme.

- [ ] **Step 1: Create .github/workflows/ios.yml**

```yaml
name: iOS CI

on:
  push:
    branches: [main]
    paths: ['ios/**']
  pull_request:
    paths: ['ios/**']

jobs:
  test:
    runs-on: macos-latest

    steps:
      - uses: actions/checkout@v4

      - name: Test
        run: |
          xcodebuild test \
            -project ios/FamilyHub/FamilyHub.xcodeproj \
            -scheme FamilyHub \
            -destination 'platform=iOS Simulator,name=iPhone 16,OS=latest' \
            -resultBundlePath ios-test-results \
            CODE_SIGNING_ALLOWED=NO
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ios.yml
git commit -m "ci: add iOS build and test workflow"
```

---

### Task 9: Create home-assistant.yml

**Files:**
- Create: `.github/workflows/home-assistant.yml`

Triggers on changes to `home-assistant/**`. Runs lint (ruff) and tests (pytest). On push to `main`, auto-tags with CalVer + run number and creates a GitHub release with `family_hub.zip` attached.

The zip is built from inside `home-assistant/` so the internal structure is `custom_components/family_hub/` — what HACS expects.

- [ ] **Step 1: Create .github/workflows/home-assistant.yml**

```yaml
name: Home Assistant CI

on:
  push:
    branches: [main]
    paths: ['home-assistant/**']
  pull_request:
    paths: ['home-assistant/**']

jobs:
  lint-and-test:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'

      - name: Install tools
        run: pip install ruff pytest

      - name: Lint
        run: ruff check home-assistant/

      - name: Test
        run: |
          cd home-assistant
          pytest || true   # pass until tests are added

  release:
    if: github.ref == 'refs/heads/main'
    needs: lint-and-test
    runs-on: ubuntu-latest

    permissions:
      contents: write

    steps:
      - uses: actions/checkout@v4

      - name: Generate tag
        id: tag
        run: |
          TAG="v$(date +'%Y.%m.%d').${{ github.run_number }}"
          echo "tag=$TAG" >> $GITHUB_OUTPUT

      - name: Create zip
        run: |
          cd home-assistant
          zip -r ../family_hub.zip custom_components/family_hub/

      - name: Create release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ steps.tag.outputs.tag }}
          name: ${{ steps.tag.outputs.tag }}
          files: family_hub.zip
          generate_release_notes: true
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/home-assistant.yml
git commit -m "ci: add Home Assistant lint, test, and auto-release workflow"
```

---

## Chunk 4: Update docs and gitignore

### Task 10: Update CLAUDE.md

**Files:**
- Modify: `CLAUDE.md`

The project CLAUDE.md documents paths and dev commands that reference the old root layout.

- [ ] **Step 1: Update CLAUDE.md**

Update the following sections:

**Architecture section** — update the `internal/` tree to show it now lives under `server/`:
```
server/
├── internal/
│   ├── config/
│   ├── database/
│   ├── models/
│   ├── repository/
│   ├── services/
│   ├── handlers/
│   ├── middleware/
│   └── server/
templates/           (under server/)
├── layouts/
├── pages/
└── components/
```

**Dev Workflow section** — update commands to run from `server/`:
```bash
cd server && make dev          # Air hot reload
docker compose up              # Full stack (run from server/)
cd server && make templ        # Regenerate templ files
cd server && make css          # Rebuild Tailwind CSS
```

**Testing section**:
```bash
cd server && make test
```

Also add a section noting the monorepo layout:
```
# Monorepo Layout
server/           Go backend (this component)
ios/              iOS SwiftUI app
home-assistant/   Home Assistant custom component
```

- [ ] **Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: update CLAUDE.md for server/ subdirectory layout"
```

---

### Task 11: Update root .gitignore for new layout

**Files:**
- Modify: `.gitignore`

Most gitignore patterns (like `bin/`, `tmp/`, `node_modules/`) apply recursively and will still cover `server/bin/` etc. However, a few root-specific entries need reviewing.

- [ ] **Step 1: Add server-specific paths to ensure coverage**

The pattern `family-hub` (the compiled binary name) currently matches at root. After the move, it's compiled as `server/family-hub` — the pattern `family-hub` still matches it (no leading `/`). No change needed for the binary.

Verify these patterns still make sense and add any missing ones:
```gitignore
# These cover server/ recursively — no change needed
bin/
tmp/
node_modules/
static/css/styles.css
*.out
coverage.*

# data/ stays at root — correct
data/
```

If no changes are needed, skip to Step 2.

- [ ] **Step 2: Commit (only if changes made)**

```bash
git add .gitignore
git commit -m "chore: update .gitignore for monorepo layout"
```

---

### Task 12: Final verification

- [ ] **Step 1: Verify root directory is clean**

```bash
ls -la
```

Expected at root: `server/`, `ios/`, `home-assistant/`, `data/`, `docs/`, `.github/`, `hacs.json`, `README.md`, `CLAUDE.md`, `.gitignore`. No stray Go files, Dockerfiles, or Makefiles.

- [ ] **Step 2: Verify server still builds from its directory**

```bash
cd server && go build -o /dev/null .
```

Expected: exits 0.

- [ ] **Step 3: Verify tests still pass**

```bash
cd server && go test ./...
```

Expected: all pass.

- [ ] **Step 4: Verify HA structure is HACS-ready**

```bash
ls home-assistant/custom_components/family_hub/
cat hacs.json
```

Expected: component files present; `hacs.json` contains `"zip_release": true`.

- [ ] **Step 5: Verify no broken references remain at root**

```bash
grep -r "internal/" . --include="*.md" --include="*.yml" --include="*.yaml" \
  --exclude-dir=server --exclude-dir=.git
```

Expected: any matches should be in docs referencing `server/internal/`, not bare `internal/`.

- [ ] **Step 6: Final commit if anything was missed**

```bash
git status
```

If clean: done. If dirty: stage and commit any remaining cleanup.
