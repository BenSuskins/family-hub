# Todo Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 7 backlog items: meal row height, calendar view date, token display HTML, overdue dashboard highlight, chore history delete, iCal token scoping, and filter dropdowns.

**Architecture:** Changes span templ templates, HTTP handlers, repository layer, and one DB migration. Each task is self-contained and committed independently.

**Tech Stack:** Go, chi, templ (run `PATH=$(go env GOPATH)/bin:$PATH templ generate` after any `.templ` edit), HTMX, Tailwind CSS, SQLite (modernc.org/sqlite). Tests use `testutil.NewTestDatabase(t)` with in-memory SQLite. Test package convention is `repository_test` (external black-box).

---

## Task 1: Meal Planner Row Height

**Files:**
- Modify: `templates/pages/meals.templ:71`

No Go tests — pure template visual fix.

**Step 1: Edit MealRow outer div**

In `templates/pages/meals.templ`, line 71, change:
```
<div id={ mealCellID(date, mealType) } class="px-4 py-2">
```
to:
```
<div id={ mealCellID(date, mealType) } class="px-4 py-3 min-h-[3.5rem] flex items-center">
```

**Step 2: Regenerate templates and build**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go build ./...
```
Expected: no errors

**Step 3: Commit**

```bash
git add templates/pages/meals.templ templates/pages/meals_templ.go
git commit -m "fix: increase meal planner row height for visual consistency"
```

---

## Task 2: Calendar View Toggle Uses Today

**Files:**
- Modify: `templates/pages/calendar.templ:42,46,50,54`

No Go tests — pure template fix.

**Step 1: Update the four view toggle hrefs**

In `templates/pages/calendar.templ`, the four view toggle `<a>` tags (lines ~42–54) currently call `viewURL(view, props.Date)`. Change all four to `todayURL(view)`:

```
href={ templ.SafeURL(todayURL("year")) }
```
```
href={ templ.SafeURL(todayURL("month")) }
```
```
href={ templ.SafeURL(todayURL("week")) }
```
```
href={ templ.SafeURL(todayURL("day")) }
```

**Step 2: Regenerate and build**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go build ./...
```
Expected: no errors

**Step 3: Commit**

```bash
git add templates/pages/calendar.templ templates/pages/calendar_templ.go
git commit -m "fix: calendar view toggle navigates to today in selected view"
```

---

## Task 3: Token Display HTML Fragment

**Files:**
- Modify: `internal/handlers/admin.go`
- Modify: `templates/pages/admin.templ:132`
- Modify: `internal/server/server.go`

**Step 1: Write failing test**

Create `internal/handlers/admin_test.go`:

```go
package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/handlers"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestAdminHandler_CreateToken_ReturnsHTML(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewAPITokenRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)

	admin := models.User{ID: "u1", Name: "Admin", Email: "admin@test.com", Role: models.RoleAdmin}
	if _, err := userRepo.Create(t.Context(), admin); err != nil {
		t.Fatalf("creating user: %v", err)
	}

	handler := handlers.NewAdminHandler(userRepo, tokenRepo, settingsRepo, categoryRepo, assignmentRepo)

	form := url.Values{"name": {"mytoken"}, "scope": {"api"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/tokens", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(middleware.SetUser(req.Context(), admin))

	w := httptest.NewRecorder()
	handler.CreateToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "{") {
		t.Error("response contains raw JSON brace, expected HTML")
	}
	if !strings.Contains(body, "mytoken") {
		t.Error("expected token name in response HTML")
	}
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/handlers/... -run TestAdminHandler_CreateToken_ReturnsHTML -v
```
Expected: FAIL — `handlers.NewAdminHandler` has wrong signature and `CreateToken` method doesn't exist yet.

**Step 3: Check whether middleware.SetUser exists**

```bash
grep -n "SetUser\|func SetUser" internal/middleware/auth.go
```
If `SetUser` does not exist, check how other handler tests inject the user into context (see `internal/handlers/dashboard_test.go`). Use whatever pattern the existing tests use.

**Step 4: Add AdminTokenCreated templ fragment to admin.templ**

Append to the end of `templates/pages/admin.templ` (before the closing `}`):

```templ
templ AdminTokenCreated(name string, token string) {
	<div class="rounded-xl bg-emerald-50 border border-emerald-200 p-4">
		<p class="text-sm font-medium text-emerald-800 mb-2">
			Token "<span class="font-mono">{ name }</span>" created — copy it now, it won't be shown again.
		</p>
		<code class="block text-xs font-mono bg-white border border-emerald-200 rounded-lg p-3 break-all text-stone-800 select-all">{ token }</code>
	</div>
}
```

**Step 5: Add assignmentRepo field and update NewAdminHandler**

In `internal/handlers/admin.go`, add `assignmentRepo repository.ChoreAssignmentRepository` field to the struct and parameter to `NewAdminHandler`:

```go
type AdminHandler struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.APITokenRepository
	settingsRepo   repository.SettingsRepository
	categoryRepo   repository.CategoryRepository
	assignmentRepo repository.ChoreAssignmentRepository
}

func NewAdminHandler(
	userRepo repository.UserRepository,
	tokenRepo repository.APITokenRepository,
	settingsRepo repository.SettingsRepository,
	categoryRepo repository.CategoryRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		settingsRepo:   settingsRepo,
		categoryRepo:   categoryRepo,
		assignmentRepo: assignmentRepo,
	}
}
```

**Step 6: Add CreateToken method to AdminHandler**

In `internal/handlers/admin.go`:

```go
func (handler *AdminHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	scope := r.FormValue("scope")
	if scope != "api" && scope != "ical" {
		scope = "api"
	}

	rawToken := generateToken()
	token := models.APIToken{
		Name:            name,
		Scope:           scope,
		TokenHash:       repository.HashToken(rawToken),
		CreatedByUserID: user.ID,
	}

	if _, err := handler.tokenRepo.Create(ctx, token); err != nil {
		slog.Error("creating token", "error", err)
		http.Error(w, "failed to create token", http.StatusInternalServerError)
		return
	}

	pages.AdminTokenCreated(name, rawToken).Render(ctx, w)
}
```

Note: `generateToken()` is already defined in `internal/handlers/api.go` and is accessible within the same package.

**Step 7: Update admin.templ form target**

In `templates/pages/admin.templ`, line 132, change:
```
hx-post="/api/tokens"
```
to:
```
hx-post="/admin/tokens"
```

**Step 8: Update server.go**

In `internal/server/server.go`:

1. Update `NewAdminHandler` call to pass `assignmentRepo`:
```go
adminHandler := handlers.NewAdminHandler(userRepo, tokenRepo, settingsRepo, categoryRepo, assignmentRepo)
```

2. Inside the admin-required group (around line 129), add:
```go
r.Post("/admin/tokens", adminHandler.CreateToken)
```

**Step 9: Regenerate, build, run tests**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go test ./internal/handlers/... -run TestAdminHandler_CreateToken_ReturnsHTML -v
```
Expected: PASS

**Step 10: Run all tests**

```bash
go test ./...
```
Expected: PASS

**Step 11: Commit**

```bash
git add internal/handlers/admin.go internal/handlers/admin_test.go \
        templates/pages/admin.templ templates/pages/admin_templ.go \
        internal/server/server.go
git commit -m "fix: token creation returns HTML fragment instead of raw JSON"
```

---

## Task 4: Highlight Overdue Chores on Dashboard

**Files:**
- Modify: `templates/pages/dashboard.templ:84`

No Go tests — pure template visual fix.

**Step 1: Add conditional class to the `<li>` row**

In `templates/pages/dashboard.templ`, line 84, change:
```
<li id={ "dashboard-chore-" + chore.ID } class="py-3 flex justify-between items-center">
```
to:
```templ
<li id={ "dashboard-chore-" + chore.ID } class={ "py-3 flex justify-between items-center rounded-lg " + overdueChoreRowClass(chore.Status) }>
```

**Step 2: Add helper function at the bottom of dashboard.templ**

```go
func overdueChoreRowClass(status models.ChoreStatus) string {
	if status == models.ChoreStatusOverdue {
		return "bg-red-50 border-l-2 border-red-400 pl-2 -ml-2"
	}
	return ""
}
```

**Step 3: Regenerate and build**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go build ./...
```
Expected: no errors

**Step 4: Commit**

```bash
git add templates/pages/dashboard.templ templates/pages/dashboard_templ.go
git commit -m "fix: highlight overdue chores with red accent on dashboard"
```

---

## Task 5: Delete Chore History

**Files:**
- Modify: `internal/repository/chore_assignments.go`
- Create: `internal/repository/chore_assignments_test.go`
- Modify: `internal/handlers/admin.go` (already updated in Task 3)
- Modify: `internal/server/server.go`
- Modify: `templates/pages/admin.templ`

**Step 1: Write failing test**

Create `internal/repository/chore_assignments_test.go`:

```go
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestChoreAssignmentRepository_DeleteCompleted(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	chore, err := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
		Status:          models.ChoreStatusPending,
	})
	if err != nil {
		t.Fatalf("creating chore: %v", err)
	}

	completed, err := assignmentRepo.Create(ctx, models.ChoreAssignment{
		ChoreID:    chore.ID,
		UserID:     user.ID,
		AssignedAt: time.Now(),
		Status:     models.AssignmentStatusCompleted,
	})
	if err != nil {
		t.Fatalf("creating completed assignment: %v", err)
	}
	_ = completed

	pending, err := assignmentRepo.Create(ctx, models.ChoreAssignment{
		ChoreID:    chore.ID,
		UserID:     user.ID,
		AssignedAt: time.Now(),
		Status:     models.AssignmentStatusPending,
	})
	if err != nil {
		t.Fatalf("creating pending assignment: %v", err)
	}

	if err := assignmentRepo.DeleteCompleted(ctx); err != nil {
		t.Fatalf("deleting completed assignments: %v", err)
	}

	remaining, err := assignmentRepo.FindByChoreID(ctx, chore.ID)
	if err != nil {
		t.Fatalf("finding assignments: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining assignment, got %d", len(remaining))
	}
	if len(remaining) > 0 && remaining[0].ID != pending.ID {
		t.Errorf("expected pending assignment to remain, got status %s", remaining[0].Status)
	}
}
```

**Step 2: Check AssignmentStatus constants**

```bash
grep -n "AssignmentStatus\|AssignmentStatusCompleted\|AssignmentStatusPending" internal/models/models.go
```
Use whatever constants are defined. Common names: `models.AssignmentStatusCompleted`, `models.AssignmentStatusPending`. If they differ, adjust the test.

**Step 3: Run to verify failure**

```bash
go test ./internal/repository/... -run TestChoreAssignmentRepository_DeleteCompleted -v
```
Expected: FAIL — `DeleteCompleted` method doesn't exist.

**Step 4: Add DeleteCompleted to the interface**

In `internal/repository/chore_assignments.go`, add to the `ChoreAssignmentRepository` interface:

```go
DeleteCompleted(ctx context.Context) error
```

**Step 5: Add implementation**

```go
func (repository *SQLiteChoreAssignmentRepository) DeleteCompleted(ctx context.Context) error {
	_, err := repository.database.ExecContext(ctx,
		`DELETE FROM chore_assignments WHERE status = 'completed'`)
	if err != nil {
		return fmt.Errorf("deleting completed chore assignments: %w", err)
	}
	return nil
}
```

**Step 6: Run test to verify pass**

```bash
go test ./internal/repository/... -run TestChoreAssignmentRepository_DeleteCompleted -v
```
Expected: PASS

**Step 7: Add DeleteChoreHistory handler to AdminHandler**

In `internal/handlers/admin.go`:

```go
func (handler *AdminHandler) DeleteChoreHistory(w http.ResponseWriter, r *http.Request) {
	if err := handler.assignmentRepo.DeleteCompleted(r.Context()); err != nil {
		slog.Error("deleting chore history", "error", err)
		http.Error(w, "failed to delete chore history", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}
```

**Step 8: Register route in server.go**

Inside the admin-required group, add:

```go
r.Post("/admin/chores/history/delete", adminHandler.DeleteChoreHistory)
```

**Step 9: Add Maintenance section to admin.templ**

After the closing `</div>` of the API Tokens section, add:

```templ
<!-- Maintenance -->
<div>
	<h2 class="text-lg font-medium text-stone-900 mb-4">Maintenance</h2>
	<div class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6">
		<p class="text-sm text-stone-500 mb-4">Permanently delete all completed chore assignments from the history log. Active chores are not affected.</p>
		<form
			hx-post="/admin/chores/history/delete"
			hx-confirm="Delete all chore history? This cannot be undone."
			hx-boost="true"
		>
			<button
				type="submit"
				class="inline-flex items-center gap-1.5 bg-red-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-red-700"
			>
				Clear Chore History
			</button>
		</form>
	</div>
</div>
```

**Step 10: Regenerate, build, run all tests**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go test ./...
```
Expected: PASS

**Step 11: Commit**

```bash
git add internal/repository/chore_assignments.go \
        internal/repository/chore_assignments_test.go \
        internal/handlers/admin.go \
        internal/server/server.go \
        templates/pages/admin.templ templates/pages/admin_templ.go
git commit -m "feat: add chore history deletion to admin maintenance panel"
```

---

## Task 6: iCal Token Scoping

**Files:**
- Create: `internal/database/migrations/007_token_scope.up.sql`
- Modify: `internal/models/models.go`
- Modify: `internal/repository/api_tokens.go`
- Modify: `internal/middleware/auth.go`
- Modify: `internal/handlers/ical.go`
- Modify: `templates/pages/admin.templ`
- Modify: `internal/repository/api_tokens_test.go`

**Step 1: Create migration**

Create `internal/database/migrations/007_token_scope.up.sql`:

```sql
ALTER TABLE api_tokens ADD COLUMN scope TEXT NOT NULL DEFAULT 'api';
```

**Step 2: Add Scope field to APIToken model**

In `internal/models/models.go`, update `APIToken`:

```go
type APIToken struct {
	ID              string
	Name            string
	TokenHash       string
	Scope           string // "api" or "ical"
	CreatedByUserID string
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}
```

**Step 3: Write failing tests for scope in api_tokens_test.go**

In `internal/repository/api_tokens_test.go`, add:

```go
func TestAPITokenRepository_ScopeRoundTrip(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewAPITokenRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	icalToken, err := tokenRepo.Create(ctx, models.APIToken{
		Name: "iCal Feed", TokenHash: "hash-ical", Scope: "ical", CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating ical token: %v", err)
	}

	apiToken, err := tokenRepo.Create(ctx, models.APIToken{
		Name: "API Key", TokenHash: "hash-api", Scope: "api", CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating api token: %v", err)
	}

	found, err := tokenRepo.FindByTokenHash(ctx, "hash-ical")
	if err != nil {
		t.Fatalf("finding ical token: %v", err)
	}
	if found.Scope != "ical" {
		t.Errorf("expected scope 'ical', got '%s'", found.Scope)
	}
	_ = icalToken

	found, err = tokenRepo.FindByTokenHash(ctx, "hash-api")
	if err != nil {
		t.Fatalf("finding api token: %v", err)
	}
	if found.Scope != "api" {
		t.Errorf("expected scope 'api', got '%s'", found.Scope)
	}
	_ = apiToken

	all, err := tokenRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding all tokens: %v", err)
	}
	scopes := make(map[string]string)
	for _, tok := range all {
		scopes[tok.Name] = tok.Scope
	}
	if scopes["iCal Feed"] != "ical" {
		t.Errorf("expected FindAll to return ical scope, got '%s'", scopes["iCal Feed"])
	}
}
```

**Step 4: Run to verify failure**

```bash
go test ./internal/repository/... -run TestAPITokenRepository_ScopeRoundTrip -v
```
Expected: FAIL — `scope` column not yet in INSERT/SELECT.

**Step 5: Update repository SQL in api_tokens.go**

Update `Create` to include `scope`:
```go
_, err := repository.database.ExecContext(ctx,
    `INSERT INTO api_tokens (id, name, token_hash, scope, created_by_user_id, expires_at, created_at)
    VALUES (?, ?, ?, ?, ?, ?, ?)`,
    token.ID, token.Name, token.TokenHash, token.Scope, token.CreatedByUserID, token.ExpiresAt, token.CreatedAt,
)
```

Update all SELECT statements in `FindByTokenHash`, `FindByUserIDAndName`, `FindAll` to include `scope` in the column list and in the `.Scan` call:

```go
// SELECT:
`SELECT id, name, token_hash, scope, created_by_user_id, expires_at, created_at
FROM api_tokens WHERE token_hash = ?`

// Scan:
).Scan(&token.ID, &token.Name, &token.TokenHash, &token.Scope, &token.CreatedByUserID, &token.ExpiresAt, &token.CreatedAt)
```
Apply this same pattern to `FindByUserIDAndName` and `FindAll`.

**Step 6: Run tests to verify pass**

```bash
go test ./internal/repository/... -run TestAPITokenRepository_ScopeRoundTrip -v
```
Expected: PASS. Then run all:
```bash
go test ./internal/repository/...
```
Expected: PASS (existing tests still pass because `DEFAULT 'api'` means zero-value scope is fine).

**Step 7: Update APITokenAuth middleware to enforce api scope**

In `internal/middleware/auth.go`, after the `FindByTokenHash` call, add scope check:

```go
token, err := tokenRepo.FindByTokenHash(r.Context(), tokenHash)
if err != nil {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}

if token.Scope != "api" {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

**Step 8: Update iCal handler to enforce ical scope**

In `internal/handlers/ical.go`, replace the token check block (lines ~51–55):

```go
authorized := handler.haToken != "" && token == handler.haToken
if !authorized {
    tokenHash := repository.HashToken(token)
    if found, err := handler.tokenRepo.FindByTokenHash(r.Context(), tokenHash); err == nil && found.Scope == "ical" {
        authorized = true
    }
}
```

**Step 9: Add scope column to token list in admin.templ**

In the API Tokens table in `templates/pages/admin.templ`:

1. Add `<th>` for Scope after the Name column header:
```templ
<th class="px-4 py-2 text-left text-xs font-medium text-stone-500 uppercase">Scope</th>
```

2. Add `<td>` for scope in the row loop:
```templ
<td class="px-4 py-2 text-sm text-stone-500">{ token.Scope }</td>
```

3. Add a scope `<select>` to the create form (before the submit button):
```templ
<select
    name="scope"
    class="rounded-xl border-stone-200 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
>
    <option value="api">API</option>
    <option value="ical">iCal</option>
</select>
```

**Step 10: Regenerate and run all tests**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go test ./...
```
Expected: PASS

**Step 11: Commit**

```bash
git add internal/database/migrations/007_token_scope.up.sql \
        internal/models/models.go \
        internal/repository/api_tokens.go \
        internal/repository/api_tokens_test.go \
        internal/middleware/auth.go \
        internal/handlers/ical.go \
        templates/pages/admin.templ templates/pages/admin_templ.go
git commit -m "feat: add scope to API tokens to separate iCal from API access"
```

---

## Task 7: Filters to Dropdown

**Files:**
- Modify: `templates/pages/chores.templ:87–146`
- Modify: `templates/pages/events.templ:35–55`

No Go tests — pure template change.

**Step 1: Replace chores filter section**

In `templates/pages/chores.templ`, replace the entire `<!-- Filters -->` div (from `<div class="flex flex-wrap items-center gap-2">` through its closing `</div>`, lines ~88–146) with:

```templ
<details class="relative">
	<summary class="inline-flex items-center gap-2 cursor-pointer list-none px-3 py-1.5 rounded-xl border border-stone-200 bg-white text-sm text-stone-700 hover:bg-stone-50 select-none">
		Filters
		if hasActiveChoreFilter(props.Filter) {
			<span class="inline-flex items-center justify-center h-4 w-4 rounded-full bg-indigo-600 text-white text-xs font-bold">!</span>
		}
	</summary>
	<div class="absolute z-10 mt-1 bg-white border border-stone-200 rounded-xl shadow-lg p-4 min-w-[300px] space-y-4">
		if props.ActiveTab != "history" {
			<div>
				<p class="text-xs font-medium text-stone-500 uppercase tracking-wide mb-2">Status</p>
				<div class="flex flex-wrap gap-1.5">
					<button type="button" class={ choreStatusPillClass(props.Filter, "") } hx-get={ choreFilterURL(props.ActiveTab, "", choreFilterAssignedTo(props.Filter), choreFilterCategoryID(props.Filter)) } hx-target="#chore-table-content" hx-swap="outerHTML">All</button>
					<button type="button" class={ choreStatusPillClass(props.Filter, "pending") } hx-get={ choreFilterURL(props.ActiveTab, "pending", choreFilterAssignedTo(props.Filter), choreFilterCategoryID(props.Filter)) } hx-target="#chore-table-content" hx-swap="outerHTML">Pending</button>
					<button type="button" class={ choreStatusPillClass(props.Filter, "overdue") } hx-get={ choreFilterURL(props.ActiveTab, "overdue", choreFilterAssignedTo(props.Filter), choreFilterCategoryID(props.Filter)) } hx-target="#chore-table-content" hx-swap="outerHTML">Overdue</button>
				</div>
			</div>
		}
		<div>
			<p class="text-xs font-medium text-stone-500 uppercase tracking-wide mb-2">User</p>
			<div class="flex flex-wrap gap-1.5">
				<button type="button" class={ choreUserPillClass(props.Filter, "") } hx-get={ choreFilterURL(props.ActiveTab, choreFilterStatus(props.Filter), "", choreFilterCategoryID(props.Filter)) } hx-target="#chore-table-content" hx-swap="outerHTML">All</button>
				for _, u := range props.Users {
					<button type="button" class={ choreUserPillClass(props.Filter, u.ID) } hx-get={ choreFilterURL(props.ActiveTab, choreFilterStatus(props.Filter), u.ID, choreFilterCategoryID(props.Filter)) } hx-target="#chore-table-content" hx-swap="outerHTML">{ u.Name }</button>
				}
			</div>
		</div>
		if len(props.Categories) > 0 {
			<div>
				<p class="text-xs font-medium text-stone-500 uppercase tracking-wide mb-2">Category</p>
				<div class="flex flex-wrap gap-1.5">
					<button type="button" class={ choreCategoryPillClass(props.Filter, "") } hx-get={ choreFilterURL(props.ActiveTab, choreFilterStatus(props.Filter), choreFilterAssignedTo(props.Filter), "") } hx-target="#chore-table-content" hx-swap="outerHTML">All</button>
					for _, c := range props.Categories {
						<button type="button" class={ choreCategoryPillClass(props.Filter, c.ID) } hx-get={ choreFilterURL(props.ActiveTab, choreFilterStatus(props.Filter), choreFilterAssignedTo(props.Filter), c.ID) } hx-target="#chore-table-content" hx-swap="outerHTML">{ c.Name }</button>
					}
				</div>
			</div>
		}
	</div>
</details>
```

Add the helper function at the bottom of `chores.templ`:

```go
func hasActiveChoreFilter(filter repository.ChoreFilter) bool {
	return filter.Status != nil || filter.AssignedToUser != nil || filter.CategoryID != nil
}
```

**Step 2: Replace events filter section**

In `templates/pages/events.templ`, replace the `<!-- Category Filter Pills -->` block (lines ~35–55) with:

```templ
if len(props.Categories) > 0 {
	<details class="relative">
		<summary class="inline-flex items-center gap-2 cursor-pointer list-none px-3 py-1.5 rounded-xl border border-stone-200 bg-white text-sm text-stone-700 hover:bg-stone-50 select-none">
			Filters
			if props.CategoryFilter != nil {
				<span class="inline-flex items-center justify-center h-4 w-4 rounded-full bg-indigo-600 text-white text-xs font-bold">!</span>
			}
		</summary>
		<div class="absolute z-10 mt-1 bg-white border border-stone-200 rounded-xl shadow-lg p-4 min-w-[220px]">
			<p class="text-xs font-medium text-stone-500 uppercase tracking-wide mb-2">Category</p>
			<div class="flex flex-wrap gap-1.5">
				<button type="button" class={ eventCategoryPillClass(props.CategoryFilter, "") } hx-get="/events" hx-target="#event-list-content" hx-swap="outerHTML">All</button>
				for _, c := range props.Categories {
					<button type="button" class={ eventCategoryPillClass(props.CategoryFilter, c.ID) } hx-get={ fmt.Sprintf("/events?category=%s", c.ID) } hx-target="#event-list-content" hx-swap="outerHTML">{ c.Name }</button>
				}
			</div>
		</div>
	</details>
}
```

**Step 3: Regenerate and build**

```bash
PATH=$(go env GOPATH)/bin:$PATH templ generate
go build ./...
```
Expected: no errors

**Step 4: Run all tests**

```bash
go test ./...
```
Expected: PASS

**Step 5: Commit**

```bash
git add templates/pages/chores.templ templates/pages/chores_templ.go \
        templates/pages/events.templ templates/pages/events_templ.go
git commit -m "fix: replace inline filter pills with dropdown on chores and events"
```
