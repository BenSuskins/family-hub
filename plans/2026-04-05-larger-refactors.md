# Larger Refactors — Deferred from 2026-04-05 simplify pass

The 2026-04-05 simplify review surfaced several architectural issues that are too
large or too risky to bundle with mechanical fixes. They are recorded here so
they can be tackled deliberately.

The small/mechanical wins from the same review were already applied in the
follow-up commit: image-handling consolidation, `writeJSONError` /
`decodeJSONBody` / `DateFormat` / `isHTMXRequest` helpers, SSRF gap on iCal
subscription create, `generateToken` error return, ignored-error fixes,
`weekStart` truthful-renaming, `SettingsKeyFamilyName` constant.

---

## Architecture

### 1. Duplicated chore orchestration across HTML + JSON handlers

`handlers/chores.go` (Create/Update/Delete) and `handlers/api.go`
(CreateChore/UpdateChore/DeleteChore) both run the same end-to-end flow:
parse body → build Chore → SetEligibleAssignees → AssignNextUser → set
`series_id` → SeedFutureOccurrences. Error paths and log messages are
copy-pasted with a `"via API"` suffix.

- **Problem.** Two copies of the series-ID/seeding business rule drift easily.
  The HTML side owns `chi.URLParam`/form parsing; the API side owns JSON
  bodies; both reach into repositories and `ChoreService` independently.
- **Proposed shape.** Add `ChoreService.CreateChoreWithRecurrence(...)`,
  `UpdateChoreWithRecurrence(...)`, `DeleteChoreWithSiblings(...)` that take a
  value-typed request struct. Handlers become thin translators (HTTP ↔
  request struct, errors ↔ status codes).
- **Risk.** Touches chore creation/update/delete — highest-traffic write
  path. Needs end-to-end tests for recurrence seeding before landing.

### 2. Handlers reaching directly into repositories for business logic

The chore flows above are the worst offender. Token creation
(`admin.go:CreateToken` and `api.go:CreateToken`) is another instance —
both build the `APIToken`, call `HashToken`, insert, and (in admin.go only)
render a page.

- **Proposed shape.** `TokenService.CreateAPIToken(ctx, name, createdByUserID)
  (plainToken, APIToken, error)`. Both callers drop to ~5 lines.

### 3. `NewAPIHandler` constructor sprawl + `api.go` size

11 positional parameters at `api.go:37`, and the file itself is ~950 lines
covering chores, users, categories, dashboard, meals, recipes, calendar, and
token endpoints.

- **Proposed shape.** Either:
  1. accept an `APIHandlerDeps` struct, or
  2. split by feature area into `api_chores.go`, `api_recipes.go`,
     `api_meals.go`, `api_tokens.go`, `api_calendar.go`, `api_auth.go`,
     each with its own minimal handler struct.
- **Recommendation.** Option 2 — aligns with the existing one-file-per-
  feature pattern in the HTML handler side.

### 4. Inconsistent error response contract

`RequireUser` serves both HTML and JSON routes, but downstream handlers
disagree on error format: `api.go` returns JSON `{"error": "..."}`,
`chores.go` returns `http.Error` plain text, `DeleteToken` returns empty
200 (should be 204), some HTML handlers render error pages. Mobile clients
hitting the `/api` prefix vs. the HTML prefix get different shapes.

- **Proposed shape.** Content-negotiation helper
  `respondError(w, r, status, msg)` that picks JSON vs. text based on
  `Accept` / path prefix. Apply across all handlers.

---

## Efficiency

### 5. Dashboard N+1

`DashboardHandler.collectUserStats` at `handlers/dashboard.go:181-199` runs
4 queries per user (3 COUNT + 1 FindAll) on every `/` and `/leaderboard`
load. For N users this is 4N queries.

- **Fix.** Add `assignmentRepo.CompletedCountsByUser(since)` returning
  `GROUP BY user_id` buckets, plus `choreRepo.PendingCountsByUser()`.
  Call each once.

### 6. `InjectFamilyName` middleware queries DB on every request

`middleware/auth.go:98-109` hits `settings WHERE key='family_name'` on
every HTML request, including the static fileserver route. It's
effectively immutable config.

- **Fix.** Load once at server start (or cache with short TTL + invalidate
  on `UpdateSettings`).

### 7. iCal cache is re-parsed on every request

`services/ical_fetcher.go` re-parses the cached calendar string via
`ical.ParseCalendar` on every invocation — even when `needsFetch` is false.
Called from dashboard, calendar view, and `/api/dashboard`.

- **Fix.** In-memory cache keyed by `(subscriptionID, LastFetchedAt)` that
  stores already-parsed events; invalidate when `UpdateCache` runs.

### 8. Sequential iCal fetching

`ical_fetcher.go:FetchForRange` iterates subscriptions one-at-a-time. With
K feeds and a 10s timeout, a slow feed blocks every subsequent one on the
request path.

- **Fix.** Fan out with `errgroup` and a small concurrency bound.

### 9. Missing composite index on `chore_assignments`

Leaderboard / stats queries filter on `(user_id, status, completed_at >= ?)`
but only `(user_id)` and `(chore_id)` single-column indexes exist.

- **Fix.** Add migration:
  ```sql
  CREATE INDEX idx_chore_assignments_user_completed
    ON chore_assignments(user_id, status, completed_at);
  ```

### 10. Image serving: no caching, per-request base64 decode

Avatars and recipe images are stored base64-encoded in SQLite and decoded
on every request with no `Cache-Control` / `ETag`.

- **Short-term fix.** Emit `Cache-Control: public, max-age=…` with an
  `updated_at`-derived `ETag`; return 304 on match.
- **Long-term fix.** Store raw bytes + `content_type` column; drop the
  base64 layer entirely.

### 11. Other per-request hot paths

- `ChoreHandler.Complete` HTMX branch re-runs `userRepo.FindAll` on every
  click (`handlers/chores.go:456-468`).
- `MealHandler.Planner` loads the entire recipes table (including
  ingredients JSON) just to populate a `<select>` dropdown.
- `SeedExistingRecurringChores` runs in a goroutine on every process start
  (`server/server.go:187-192`) though labeled "one-time" — should be a
  migration or a `settings` flag.
- `repository/chores.go:SetEligibleAssignees` inserts rows in a loop
  instead of a single multi-row `INSERT`.

---

## Model cleanup

### 12. Unused / dead model fields

- `models.User.OnboardedAt` — CLAUDE.md explicitly says "No onboarding flow".
- `models.Event.{CategoryID, CreatedByUserID, CreatedAt, UpdatedAt, Color,
  EndTime}` — `Event` is only produced by the iCal fetcher, never
  persisted, so DB-style timestamps are noise.
- `models.Recipe.HasImage` — "computed: image_data != ''" — should be a
  method, not a field.
- `models.Recipe.Instructions` — legacy, fallback-to-Steps logic repeats
  at `handlers/recipes.go:163-165` and `:383-385`. Migrate once and delete.
- `models.Chore.LastAssignedIndex` — derivable from `chore_assignments`
  history; currently requires propagation on every recurrence creation.

### 13. `repository.chores.OrderBy*` string constants leak SQL

`repository/chores.go:14-17` exports ORDER BY SQL fragments as package
constants used directly from handlers. Leaks SQL into the handler layer.

- **Fix.** Replace with a typed enum (`ChoreOrder int`) and resolve the
  SQL inside the repository.

---

## Small reuse opportunities (lower priority)

- `buildUserMaps(users)` helper — loop at `chores.go:63`, `chores.go:458`,
  `dashboard.go:125`, `calendar.go:135` builds the same two `id→name` /
  `id→avatar` maps.
- `parseCalendarRange(r)` — `calendar.go:50-107` and `api.go:397-432`
  duplicate the week/day/month range parsing with identical Monday-offset
  math.
- `scanRows[T]` generic helper for the ~8 repository scan loops that each
  re-implement `defer rows.Close()` + `for rows.Next()` + `rows.Err()`
  (some sites currently skip `rows.Err()` — worth auditing during the
  migration).
- `queryIntOr(r, key, default)` for ~6 sites that parse integer query
  params with a default.
