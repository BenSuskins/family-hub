# Onboarding Wizard Design

Date: 2026-02-28

## Overview

Two distinct onboarding flows:

1. **Admin setup wizard** (`/setup`) — runs once on fresh install; covers family name, user invite awareness, and first chore category.
2. **Member welcome flow** (`/welcome`) — runs once per new user on first login; covers a welcome message and profile setup (name + avatar).

Both flows are HTMX-driven: a single URL per flow, with steps swapping in as partials via HTMX POSTs.

---

## Data Model

### Migration 013

**`settings` table** (existing key/value store):
- New key: `onboarding_complete = "true"` — written when the admin completes the setup wizard.

**`users` table**:
- New column: `onboarded_at TIMESTAMP NULL` — null indicates the user has not yet completed the welcome flow.

---

## Middleware

A new `RequireOnboarding` middleware in `internal/middleware/onboarding.go`, applied after `RequireAuth` to all authenticated routes.

**Logic:**
1. If `settings.onboarding_complete != "true"` → redirect to `/setup`
2. Else if current user's `onboarded_at` is null → redirect to `/welcome`
3. Otherwise, pass through

The `/setup` and `/welcome` routes are exempt from this middleware to prevent redirect loops.

---

## Admin Setup Wizard (`/setup`)

Three steps. Each step POSTs and swaps in the next step partial.

### Step 1 — Family Name
- Pre-filled with current `family_name` setting (default: `"Family"`)
- `POST /setup/family-name` → saves to settings, returns step 2 partial

### Step 2 — Invite Awareness
- Displays current users; explains others join by logging in via OIDC
- Single "Got it" button, no data saved
- `POST /setup/acknowledge-users` → returns step 3 partial

### Step 3 — First Category
- Optional text field to create one chore category (e.g. "Cleaning")
- Can be skipped
- `POST /setup/first-category` → creates category if provided, sets `onboarding_complete = "true"`, redirects to `/`

---

## Member Welcome Flow (`/welcome`)

Two steps.

### Step 1 — Welcome
- Displays "Welcome to [Family Name], [User Name]!"
- Brief description of the hub (chores, meals, calendar)
- "Set up your profile" button
- `POST /welcome/start` → returns step 2 partial

### Step 2 — Profile Setup
- Editable name field (pre-filled from OIDC)
- Avatar upload (reuses existing profile upload logic)
- `POST /welcome/profile` → saves name + avatar, sets `onboarded_at = now()`, redirects to `/`

---

## Handlers & Routes

**New handler:** `OnboardingHandler` in `internal/handlers/onboarding.go`

Dependencies: `SettingsRepository`, `UserRepository`, `CategoryRepository`

**Routes (authenticated, exempt from `RequireOnboarding`):**

```
GET  /setup                    → setup wizard page (step 1)
POST /setup/family-name        → save family name, return step 2 partial
POST /setup/acknowledge-users  → return step 3 partial
POST /setup/first-category     → save category, complete setup, redirect /

GET  /welcome                  → welcome page (step 1)
POST /welcome/start            → return step 2 partial
POST /welcome/profile          → save name + avatar, set onboarded_at, redirect /
```

---

## Templates

- `templates/pages/setup.templ` — setup wizard shell + step 1 content
- `templates/components/setup_steps.templ` — step 2 and step 3 partials
- `templates/pages/welcome.templ` — welcome shell + step 1 content
- `templates/components/welcome_steps.templ` — step 2 partial

---

## Testing

- `internal/middleware/onboarding_test.go` — table-driven tests for the three redirect cases (setup incomplete, user not onboarded, pass-through)
- `internal/handlers/onboarding_test.go` — table-driven tests for each POST endpoint: correct data saved, correct partial returned, redirect on final step
- Uses in-memory SQLite as per existing pattern; no new fakes needed
