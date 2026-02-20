# Todo Fixes Design

Date: 2026-02-20

## Scope

Seven backlog items, ordered easiest to hardest. Security audit and backup/restore excluded for now.

---

## 1. Meal Planner Rows Narrow

**File:** `templates/pages/meals.templ`

`MealRow` has `px-4 py-2` giving very little vertical space. Increase to `py-3` and add `min-h-[3.5rem]` to match the height of list rows elsewhere in the app.

---

## 2. Calendar Views Don't Use Current Date

**File:** `templates/pages/calendar.templ`

View toggle links use `viewURL(view, props.Date)`. In year view `props.Date` is Jan 1, so switching to day/week lands on Jan 1 instead of today. Fix: replace view toggle hrefs with `todayURL(view)` so switching view type always navigates to today in the selected view.

---

## 3. Token Display Is Raw JSON

**Files:** `internal/handlers/admin.go`, `templates/pages/admin.templ`, new templ fragment

Currently the "Create Token" form POSTs to `/api/tokens` (a JSON endpoint) with `hx-target="#new-token-result"`, so the raw JSON blob renders in the div.

Fix:
- Add `POST /admin/tokens` route that creates the token via the same repo call, then returns an HTML fragment (templ component) showing the plain token value in a copyable box.
- Update the admin form to POST to `/admin/tokens`.

---

## 4. Highlight Overdue Chores on Home Tab

**File:** `templates/pages/dashboard.templ`

Dashboard rows already detect `ChoreStatusOverdue` and show a badge, but the row itself has no visual distinction. Add `border-l-2 border-red-400 bg-red-50` to the row container when `chore.Status == models.ChoreStatusOverdue`.

---

## 5. Delete Historic Chores / Reset

**Files:** `internal/repository/chores.go`, `internal/handlers/admin.go`, `templates/pages/admin.templ`

- Add `DeleteCompletedAssignments(ctx context.Context) error` to `ChoreRepository` interface and SQLite implementation. Deletes all rows from `chore_assignments` where `status = 'completed'`.
- Add `POST /admin/chores/history/delete` handler that calls the repo method and redirects back to admin with a success message.
- Add a "Clear Chore History" button in the admin page under a new "Maintenance" section, with `hx-confirm` prompt.

---

## 6. iCAL Token Scoping

**Files:** `internal/models/models.go`, `internal/repository/api_tokens.go`, `internal/handlers/ical.go`, `internal/handlers/api.go`, `internal/handlers/admin.go`, `templates/pages/admin.templ`, new migration

- **Migration 007:** Add `scope TEXT NOT NULL DEFAULT 'api'` to `api_tokens` table.
- **Model:** Add `Scope string` to `APIToken`. Valid values: `"api"`, `"ical"`.
- **Token creation UI:** Add a scope `<select>` (API / iCal) to the create token form in admin.
- **iCal handler:** After finding token by hash, check `token.Scope == "ical"`. Reject with 401 if not.
- **API handler:** After finding token by hash, check `token.Scope == "api"`. Reject with 401 if not.
- **HA token:** Unchanged — HA uses a separate env-var token, not the scoped system.

---

## 7. Move Chore / Event Filters to Dropdown

**Files:** `templates/pages/chores.templ`, `templates/pages/events.templ`

Currently chores has three rows of inline pills (status / user / category) and events has one (category). Replace each filter group with a compact `<details>/<summary>` dropdown — no JS required.

- Summary shows "Filters" with an active-filter count badge when any filter is applied.
- The open panel contains the existing pill buttons, laid out in a grid inside the dropdown.
- Active filter state is preserved — pills still use HTMX to swap the list in place.
