# Onboarding Wizard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a two-track onboarding wizard: a one-time admin setup flow (`/setup`) and a per-user welcome flow (`/welcome`), both gated by middleware that redirects unonboarded requests.

**Architecture:** A `RequireOnboarding` middleware sits after `RequireAuth` and checks two conditions: whether the `onboarding_complete` settings key is set, and whether the current user's `onboarded_at` timestamp is populated. The wizard steps are HTMX-driven partials served by a new `OnboardingHandler`.

**Tech Stack:** Go, chi router, HTMX, templ, modernc.org/sqlite, Tailwind CSS.

---

### Task 1: Migration — add `onboarded_at` to users

**Files:**
- Create: `internal/database/migrations/013_onboarding.up.sql`

**Step 1: Write the migration**

```sql
ALTER TABLE users ADD COLUMN onboarded_at TIMESTAMP NULL;
```

**Step 2: Verify migration runs**

Run: `make test`
Expected: all tests pass (migration runs automatically via `testutil.NewTestDatabase`)

**Step 3: Commit**

```bash
git add internal/database/migrations/013_onboarding.up.sql
git commit -m "feat: add onboarded_at column to users"
```

---

### Task 2: Update User model and UserRepository

**Files:**
- Modify: `internal/models/user.go` (or wherever `User` struct is defined — it's in `internal/models/`)
- Modify: `internal/repository/users.go`

**Step 1: Write the failing test**

Add to `internal/repository/users_test.go`:

```go
func TestUserRepository_MarkOnboarded(t *testing.T) {
    database := testutil.NewTestDatabase(t)
    repo := NewUserRepository(database)
    ctx := context.Background()

    user, err := repo.Create(ctx, models.User{
        OIDCSubject: "sub-onboard-test",
        Email:       "onboard@example.com",
        Name:        "Onboard User",
        Role:        models.RoleMember,
    })
    if err != nil {
        t.Fatalf("creating user: %v", err)
    }

    if user.OnboardedAt != nil {
        t.Fatal("expected OnboardedAt to be nil before onboarding")
    }

    if err := repo.MarkOnboarded(ctx, user.ID); err != nil {
        t.Fatalf("marking user as onboarded: %v", err)
    }

    updated, err := repo.FindByID(ctx, user.ID)
    if err != nil {
        t.Fatalf("finding user after onboarding: %v", err)
    }

    if updated.OnboardedAt == nil {
        t.Fatal("expected OnboardedAt to be set after onboarding")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/repository/ -run TestUserRepository_MarkOnboarded -v`
Expected: FAIL — `MarkOnboarded` not defined

**Step 3: Add `OnboardedAt` to the User model**

In `internal/models/` find the `User` struct (in the file that defines it) and add:

```go
OnboardedAt *time.Time
```

Add it after `UpdatedAt time.Time`.

**Step 4: Update `userColumns` and `scanUserFields` in `internal/repository/users.go`**

Change:
```go
const userColumns = "id, oidc_subject, email, name, avatar_url, role, created_at, updated_at"
```
To:
```go
const userColumns = "id, oidc_subject, email, name, avatar_url, role, created_at, updated_at, onboarded_at"
```

Change `scanUserFields`:
```go
func scanUserFields(user *models.User) []any {
    return []any{
        &user.ID, &user.OIDCSubject, &user.Email, &user.Name,
        &user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt,
        &user.OnboardedAt,
    }
}
```

**Step 5: Add `MarkOnboarded` to the interface and implement it**

Add to the `UserRepository` interface:
```go
MarkOnboarded(ctx context.Context, id string) error
```

Add the implementation to `SQLiteUserRepository`:
```go
func (repository *SQLiteUserRepository) MarkOnboarded(ctx context.Context, id string) error {
    _, err := repository.database.ExecContext(ctx,
        "UPDATE users SET onboarded_at = ?, updated_at = ? WHERE id = ?",
        time.Now(), time.Now(), id,
    )
    if err != nil {
        return fmt.Errorf("marking user as onboarded: %w", err)
    }
    return nil
}
```

**Step 6: Run tests**

Run: `make test`
Expected: all tests pass

**Step 7: Commit**

```bash
git add internal/models/ internal/repository/users.go internal/repository/users_test.go
git commit -m "feat: add OnboardedAt to User model and MarkOnboarded to repository"
```

---

### Task 3: `RequireOnboarding` middleware

**Files:**
- Create: `internal/middleware/onboarding.go`
- Create: `internal/middleware/onboarding_test.go`

**Step 1: Write the failing tests**

```go
package middleware

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/bensuskins/family-hub/internal/models"
    "github.com/bensuskins/family-hub/internal/testutil"
    "github.com/bensuskins/family-hub/internal/repository"
)

func TestRequireOnboarding_RedirectsToSetupWhenNotComplete(t *testing.T) {
    database := testutil.NewTestDatabase(t)
    settingsRepo := repository.NewSettingsRepository(database)
    // onboarding_complete not set → should redirect to /setup

    user := models.User{ID: "u1", Role: models.RoleAdmin}
    handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    ctx := context.WithValue(req.Context(), UserContextKey, user)
    req = req.WithContext(ctx)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusFound {
        t.Errorf("expected 302, got %d", rec.Code)
    }
    if loc := rec.Header().Get("Location"); loc != "/setup" {
        t.Errorf("expected redirect to /setup, got %q", loc)
    }
}

func TestRequireOnboarding_RedirectsToWelcomeWhenUserNotOnboarded(t *testing.T) {
    database := testutil.NewTestDatabase(t)
    settingsRepo := repository.NewSettingsRepository(database)
    // Mark setup as complete
    _ = settingsRepo.Set(context.Background(), "onboarding_complete", "true")

    // User has nil OnboardedAt
    user := models.User{ID: "u2", OnboardedAt: nil}
    handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    ctx := context.WithValue(req.Context(), UserContextKey, user)
    req = req.WithContext(ctx)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusFound {
        t.Errorf("expected 302, got %d", rec.Code)
    }
    if loc := rec.Header().Get("Location"); loc != "/welcome" {
        t.Errorf("expected redirect to /welcome, got %q", loc)
    }
}

func TestRequireOnboarding_PassesThroughWhenFullyOnboarded(t *testing.T) {
    database := testutil.NewTestDatabase(t)
    settingsRepo := repository.NewSettingsRepository(database)
    _ = settingsRepo.Set(context.Background(), "onboarding_complete", "true")

    now := time.Now()
    user := models.User{ID: "u3", OnboardedAt: &now}
    handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))

    req := httptest.NewRequest(http.MethodGet, "/", nil)
    ctx := context.WithValue(req.Context(), UserContextKey, user)
    req = req.WithContext(ctx)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/middleware/ -run TestRequireOnboarding -v`
Expected: FAIL — `RequireOnboarding` not defined

**Step 3: Implement the middleware**

Create `internal/middleware/onboarding.go`:

```go
package middleware

import (
    "net/http"

    "github.com/bensuskins/family-hub/internal/repository"
)

func RequireOnboarding(settingsRepo repository.SettingsRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            complete, _ := settingsRepo.Get(r.Context(), "onboarding_complete")
            if complete != "true" {
                http.Redirect(w, r, "/setup", http.StatusFound)
                return
            }

            user := GetUser(r.Context())
            if user.OnboardedAt == nil {
                http.Redirect(w, r, "/welcome", http.StatusFound)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

**Step 4: Run tests**

Run: `go test ./internal/middleware/ -run TestRequireOnboarding -v`
Expected: all three pass

**Step 5: Commit**

```bash
git add internal/middleware/onboarding.go internal/middleware/onboarding_test.go
git commit -m "feat: add RequireOnboarding middleware"
```

---

### Task 4: OnboardingHandler — setup flow

**Files:**
- Create: `internal/handlers/onboarding.go`
- Create: `internal/handlers/onboarding_test.go`

**Step 1: Write the failing tests**

```go
package handlers

import (
    "context"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"

    "github.com/bensuskins/family-hub/internal/middleware"
    "github.com/bensuskins/family-hub/internal/models"
    "github.com/bensuskins/family-hub/internal/repository"
    "github.com/bensuskins/family-hub/internal/testutil"
)

func setupOnboardingHandler(t *testing.T) (*OnboardingHandler, models.User, *repository.SQLiteSettingsRepository, *repository.SQLiteCategoryRepository) {
    t.Helper()
    database := testutil.NewTestDatabase(t)
    userRepo := repository.NewUserRepository(database)
    settingsRepo := repository.NewSettingsRepository(database)
    categoryRepo := repository.NewCategoryRepository(database)

    user, err := userRepo.Create(context.Background(), models.User{
        OIDCSubject: "sub-setup-test",
        Email:       "admin@example.com",
        Name:        "Admin User",
        Role:        models.RoleAdmin,
    })
    if err != nil {
        t.Fatalf("creating test user: %v", err)
    }

    handler := NewOnboardingHandler(settingsRepo, userRepo, categoryRepo)
    return handler, user, settingsRepo, categoryRepo
}

func TestOnboarding_SetupPage(t *testing.T) {
    handler, user, _, _ := setupOnboardingHandler(t)

    req := httptest.NewRequest(http.MethodGet, "/setup", nil)
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.SetupPage(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}

func TestOnboarding_SaveFamilyName(t *testing.T) {
    handler, user, settingsRepo, _ := setupOnboardingHandler(t)

    form := url.Values{"family_name": {"The Smiths"}}
    req := httptest.NewRequest(http.MethodPost, "/setup/family-name", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.SaveFamilyName(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }

    saved, err := settingsRepo.Get(context.Background(), "family_name")
    if err != nil || saved != "The Smiths" {
        t.Errorf("expected family_name to be 'The Smiths', got %q (err: %v)", saved, err)
    }
}

func TestOnboarding_AcknowledgeUsers(t *testing.T) {
    handler, user, _, _ := setupOnboardingHandler(t)

    req := httptest.NewRequest(http.MethodPost, "/setup/acknowledge-users", nil)
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.AcknowledgeUsers(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}

func TestOnboarding_CompleteSetup_WithCategory(t *testing.T) {
    handler, user, settingsRepo, categoryRepo := setupOnboardingHandler(t)

    form := url.Values{"category_name": {"Cleaning"}}
    req := httptest.NewRequest(http.MethodPost, "/setup/first-category", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.CompleteSetup(rec, req)

    if rec.Code != http.StatusFound {
        t.Errorf("expected 302, got %d", rec.Code)
    }
    if loc := rec.Header().Get("Location"); loc != "/" {
        t.Errorf("expected redirect to /, got %q", loc)
    }

    complete, _ := settingsRepo.Get(context.Background(), "onboarding_complete")
    if complete != "true" {
        t.Error("expected onboarding_complete to be true")
    }

    categories, _ := categoryRepo.FindAll(context.Background())
    if len(categories) != 1 || categories[0].Name != "Cleaning" {
        t.Errorf("expected one category named 'Cleaning', got %v", categories)
    }
}

func TestOnboarding_CompleteSetup_WithoutCategory(t *testing.T) {
    handler, user, settingsRepo, categoryRepo := setupOnboardingHandler(t)

    req := httptest.NewRequest(http.MethodPost, "/setup/first-category", strings.NewReader(""))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.CompleteSetup(rec, req)

    if rec.Code != http.StatusFound {
        t.Errorf("expected 302, got %d", rec.Code)
    }

    complete, _ := settingsRepo.Get(context.Background(), "onboarding_complete")
    if complete != "true" {
        t.Error("expected onboarding_complete to be true")
    }

    categories, _ := categoryRepo.FindAll(context.Background())
    if len(categories) != 0 {
        t.Errorf("expected no categories, got %v", categories)
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/handlers/ -run TestOnboarding_Setup -v`
Expected: FAIL — `OnboardingHandler` not defined

**Step 3: Implement the setup flow handlers**

Create `internal/handlers/onboarding.go`:

```go
package handlers

import (
    "log/slog"
    "net/http"
    "strings"

    "github.com/bensuskins/family-hub/internal/middleware"
    "github.com/bensuskins/family-hub/internal/models"
    "github.com/bensuskins/family-hub/internal/repository"
    "github.com/bensuskins/family-hub/templates/pages"
    "github.com/bensuskins/family-hub/templates/components"
)

type OnboardingHandler struct {
    settingsRepo  repository.SettingsRepository
    userRepo      repository.UserRepository
    categoryRepo  repository.CategoryRepository
}

func NewOnboardingHandler(
    settingsRepo repository.SettingsRepository,
    userRepo repository.UserRepository,
    categoryRepo repository.CategoryRepository,
) *OnboardingHandler {
    return &OnboardingHandler{
        settingsRepo: settingsRepo,
        userRepo:     userRepo,
        categoryRepo: categoryRepo,
    }
}

func (handler *OnboardingHandler) SetupPage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)

    familyName, _ := handler.settingsRepo.Get(ctx, "family_name")
    if familyName == "" {
        familyName = "Family"
    }

    component := pages.Setup(pages.SetupProps{User: user, FamilyName: familyName})
    component.Render(ctx, w)
}

func (handler *OnboardingHandler) SaveFamilyName(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    name := strings.TrimSpace(r.FormValue("family_name"))
    if name == "" {
        name = "Family"
    }

    if err := handler.settingsRepo.Set(ctx, "family_name", name); err != nil {
        slog.Error("saving family name", "error", err)
        http.Error(w, "Error saving family name", http.StatusInternalServerError)
        return
    }

    allUsers, err := handler.userRepo.FindAll(ctx)
    if err != nil {
        slog.Error("finding users", "error", err)
        allUsers = []models.User{user}
    }

    component := components.SetupStepUsers(components.SetupStepUsersProps{Users: allUsers})
    component.Render(ctx, w)
}

func (handler *OnboardingHandler) AcknowledgeUsers(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    component := components.SetupStepCategory()
    component.Render(ctx, w)
}

func (handler *OnboardingHandler) CompleteSetup(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    categoryName := strings.TrimSpace(r.FormValue("category_name"))
    if categoryName != "" {
        if _, err := handler.categoryRepo.Create(ctx, models.Category{
            Name:            categoryName,
            CreatedByUserID: user.ID,
        }); err != nil {
            slog.Error("creating first category", "error", err)
        }
    }

    if err := handler.settingsRepo.Set(ctx, "onboarding_complete", "true"); err != nil {
        slog.Error("setting onboarding_complete", "error", err)
        http.Error(w, "Error completing setup", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/", http.StatusFound)
}
```

**Step 4: Run tests**

Run: `go test ./internal/handlers/ -run TestOnboarding_Setup -v`
Expected: FAIL — template types not defined yet (that's OK, continue to welcome flow first, then templates together)

**Step 5: Commit (after templates are done — skip for now, continue to Task 5)**

---

### Task 5: OnboardingHandler — welcome flow

**Files:**
- Modify: `internal/handlers/onboarding.go`
- Modify: `internal/handlers/onboarding_test.go`

**Step 1: Write the failing tests**

Add to `internal/handlers/onboarding_test.go`:

```go
func TestOnboarding_WelcomePage(t *testing.T) {
    handler, user, _, _ := setupOnboardingHandler(t)

    req := httptest.NewRequest(http.MethodGet, "/welcome", nil)
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.WelcomePage(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}

func TestOnboarding_WelcomeStart(t *testing.T) {
    handler, user, _, _ := setupOnboardingHandler(t)

    req := httptest.NewRequest(http.MethodPost, "/welcome/start", nil)
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.WelcomeStart(rec, req)

    if rec.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", rec.Code)
    }
}

func TestOnboarding_CompleteWelcome(t *testing.T) {
    database := testutil.NewTestDatabase(t)
    userRepo := repository.NewUserRepository(database)
    settingsRepo := repository.NewSettingsRepository(database)
    categoryRepo := repository.NewCategoryRepository(database)

    user, _ := userRepo.Create(context.Background(), models.User{
        OIDCSubject: "sub-welcome-test",
        Email:       "member@example.com",
        Name:        "Old Name",
        Role:        models.RoleMember,
    })

    handler := NewOnboardingHandler(settingsRepo, userRepo, categoryRepo)

    form := url.Values{"name": {"New Name"}}
    req := httptest.NewRequest(http.MethodPost, "/welcome/profile", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req = requestWithUser(req, user)
    rec := httptest.NewRecorder()
    handler.CompleteWelcome(rec, req)

    if rec.Code != http.StatusFound {
        t.Errorf("expected 302, got %d", rec.Code)
    }
    if loc := rec.Header().Get("Location"); loc != "/" {
        t.Errorf("expected redirect to /, got %q", loc)
    }

    updated, _ := userRepo.FindByID(context.Background(), user.ID)
    if updated.Name != "New Name" {
        t.Errorf("expected name to be 'New Name', got %q", updated.Name)
    }
    if updated.OnboardedAt == nil {
        t.Error("expected OnboardedAt to be set")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/handlers/ -run TestOnboarding_Welcome -v`
Expected: FAIL

**Step 3: Add welcome handlers to `internal/handlers/onboarding.go`**

Add these methods to `OnboardingHandler`:

```go
func (handler *OnboardingHandler) WelcomePage(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)

    familyName, _ := handler.settingsRepo.Get(ctx, "family_name")
    if familyName == "" {
        familyName = "Family"
    }

    component := pages.Welcome(pages.WelcomeProps{User: user, FamilyName: familyName})
    component.Render(ctx, w)
}

func (handler *OnboardingHandler) WelcomeStart(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)
    component := components.WelcomeStepProfile(components.WelcomeStepProfileProps{User: user})
    component.Render(ctx, w)
}

func (handler *OnboardingHandler) CompleteWelcome(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    user := middleware.GetUser(ctx)

    if err := r.ParseForm(); err != nil {
        http.Error(w, "Bad request", http.StatusBadRequest)
        return
    }

    name := strings.TrimSpace(r.FormValue("name"))
    if name == "" {
        name = user.Name
    }

    if err := handler.userRepo.UpdateProfile(ctx, user.ID, name, user.Email, user.AvatarURL); err != nil {
        slog.Error("updating profile during welcome", "error", err)
        http.Error(w, "Error saving profile", http.StatusInternalServerError)
        return
    }

    if err := handler.userRepo.MarkOnboarded(ctx, user.ID); err != nil {
        slog.Error("marking user as onboarded", "error", err)
        http.Error(w, "Error completing welcome", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/", http.StatusFound)
}
```

**Step 4: Continue to templates (tests will pass after templates are created)**

---

### Task 6: Templates — setup wizard

**Files:**
- Create: `templates/pages/setup.templ`
- Create: `templates/components/setup_steps.templ`

**Step 1: Create `templates/pages/setup.templ`**

```go
package pages

import (
    "github.com/bensuskins/family-hub/internal/models"
    "github.com/bensuskins/family-hub/templates/layouts"
)

type SetupProps struct {
    User       models.User
    FamilyName string
}

templ Setup(props SetupProps) {
    @layouts.Base("Setup", props.User, "/setup") {
        <div class="max-w-lg mx-auto space-y-6">
            <div class="text-center space-y-1">
                <h1 class="text-2xl font-semibold text-stone-800 dark:text-slate-100">Welcome to Family Hub</h1>
                <p class="text-sm text-stone-500 dark:text-slate-400">Let's get a few things set up.</p>
            </div>
            <div id="setup-step" class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6">
                <h2 class="text-base font-medium text-stone-800 dark:text-slate-100 mb-4">What's your family name?</h2>
                <form
                    hx-post="/setup/family-name"
                    hx-target="#setup-step"
                    hx-swap="outerHTML"
                    class="space-y-4"
                >
                    <input
                        type="text"
                        name="family_name"
                        value={ props.FamilyName }
                        placeholder="e.g. The Smiths"
                        class="w-full rounded-xl border border-zinc-200 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-stone-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    />
                    <button
                        type="submit"
                        class="w-full bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150"
                    >
                        Continue
                    </button>
                </form>
            </div>
        </div>
    }
}
```

**Step 2: Create `templates/components/setup_steps.templ`**

```go
package components

import "github.com/bensuskins/family-hub/internal/models"

type SetupStepUsersProps struct {
    Users []models.User
}

templ SetupStepUsers(props SetupStepUsersProps) {
    <div id="setup-step" class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-4">
        <h2 class="text-base font-medium text-stone-800 dark:text-slate-100">Inviting family members</h2>
        <p class="text-sm text-stone-500 dark:text-slate-400">
            Anyone with access to your Authentik instance can join by logging in.
            They'll appear here once they do.
        </p>
        <ul class="space-y-2">
            for _, user := range props.Users {
                <li class="flex items-center gap-3 text-sm text-stone-700 dark:text-slate-300">
                    @UserAvatar(user.Name, user.AvatarURL, "h-7 w-7 text-xs")
                    <span>{ user.Name }</span>
                </li>
            }
        </ul>
        <form
            hx-post="/setup/acknowledge-users"
            hx-target="#setup-step"
            hx-swap="outerHTML"
        >
            <button
                type="submit"
                class="w-full bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150"
            >
                Got it
            </button>
        </form>
    </div>
}

templ SetupStepCategory() {
    <div id="setup-step" class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-4">
        <h2 class="text-base font-medium text-stone-800 dark:text-slate-100">Create a chore category</h2>
        <p class="text-sm text-stone-500 dark:text-slate-400">
            Categories help organise chores (e.g. Cleaning, Kitchen, Garden). You can add more later.
        </p>
        <form
            hx-post="/setup/first-category"
            hx-boost="false"
            class="space-y-4"
        >
            <input
                type="text"
                name="category_name"
                placeholder="e.g. Cleaning (optional)"
                class="w-full rounded-xl border border-zinc-200 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-stone-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <div class="flex gap-3">
                <button
                    type="submit"
                    name="category_name"
                    value=""
                    formaction="/setup/first-category"
                    class="flex-1 text-sm text-stone-500 dark:text-slate-400 hover:underline"
                >
                    Skip
                </button>
                <button
                    type="submit"
                    class="flex-1 bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150"
                >
                    Finish setup
                </button>
            </div>
        </form>
    </div>
}
```

**Note on the final step:** The `CompleteSetup` handler redirects to `/`, which is a full-page navigation — using standard form submit (`hx-boost="false"`) is correct here so the browser follows the redirect properly.

**Step 3: Regenerate templates**

Run: `make templ`
Expected: no errors, new `*_templ.go` files generated

**Step 4: Run tests**

Run: `make test`
Expected: all tests pass

**Step 5: Commit**

```bash
git add templates/pages/setup.templ templates/pages/setup_templ.go \
        templates/components/setup_steps.templ templates/components/setup_steps_templ.go \
        internal/handlers/onboarding.go
git commit -m "feat: add setup wizard handler and templates"
```

---

### Task 7: Templates — welcome flow

**Files:**
- Create: `templates/pages/welcome.templ`
- Create: `templates/components/welcome_steps.templ`

**Step 1: Create `templates/pages/welcome.templ`**

```go
package pages

import (
    "github.com/bensuskins/family-hub/internal/models"
    "github.com/bensuskins/family-hub/templates/layouts"
)

type WelcomeProps struct {
    User       models.User
    FamilyName string
}

templ Welcome(props WelcomeProps) {
    @layouts.Base("Welcome", props.User, "/welcome") {
        <div class="max-w-lg mx-auto space-y-6">
            <div class="text-center space-y-1">
                <h1 class="text-2xl font-semibold text-stone-800 dark:text-slate-100">Welcome to { props.FamilyName } Hub!</h1>
                <p class="text-sm text-stone-500 dark:text-slate-400">Your family's home for chores, meals, and calendars.</p>
            </div>
            <div id="welcome-step" class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-4">
                <p class="text-sm text-stone-700 dark:text-slate-300">
                    Hi { props.User.Name }! Let's set up your profile before you dive in.
                </p>
                <form
                    hx-post="/welcome/start"
                    hx-target="#welcome-step"
                    hx-swap="outerHTML"
                >
                    <button
                        type="submit"
                        class="w-full bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150"
                    >
                        Set up my profile
                    </button>
                </form>
            </div>
        </div>
    }
}
```

**Step 2: Create `templates/components/welcome_steps.templ`**

```go
package components

import "github.com/bensuskins/family-hub/internal/models"

type WelcomeStepProfileProps struct {
    User models.User
}

templ WelcomeStepProfile(props WelcomeStepProfileProps) {
    <div id="welcome-step" class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-4">
        <h2 class="text-base font-medium text-stone-800 dark:text-slate-100">Your profile</h2>
        <form
            method="POST"
            action="/welcome/profile"
            enctype="multipart/form-data"
            class="space-y-4"
        >
            <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-slate-300 mb-1">Display name</label>
                <input
                    type="text"
                    name="name"
                    value={ props.User.Name }
                    class="w-full rounded-xl border border-zinc-200 dark:border-slate-600 bg-white dark:bg-slate-700 px-3 py-2 text-sm text-stone-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
            </div>
            <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-slate-300 mb-1">Avatar <span class="font-normal text-stone-400">(optional)</span></label>
                <input
                    type="file"
                    name="avatar"
                    accept="image/*"
                    class="text-sm text-stone-600 dark:text-slate-400 file:mr-3 file:py-2 file:px-3 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-zinc-100 file:text-stone-700 dark:file:bg-slate-700 dark:file:text-slate-200"
                />
            </div>
            <button
                type="submit"
                class="w-full bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150"
            >
                Let's go
            </button>
        </form>
    </div>
}
```

**Note:** The profile step uses a standard `multipart/form-data` form (not HTMX) because it may include a file upload. `CompleteWelcome` handles the avatar the same way the existing `ProfileHandler.Upload` does.

**Step 3: Update `CompleteWelcome` to handle optional avatar upload**

In `internal/handlers/onboarding.go`, update `CompleteWelcome` to optionally process a file upload before calling `MarkOnboarded`. Add after `UpdateProfile`:

```go
// Handle optional avatar upload
if file, header, err := r.FormFile("avatar"); err == nil {
    defer file.Close()
    const maxBytes = 1 * 1024 * 1024
    imageBytes, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
    if err == nil && len(imageBytes) <= maxBytes {
        contentType := header.Header.Get("Content-Type")
        if contentType == "" {
            contentType = http.DetectContentType(imageBytes)
        }
        encoded := base64.StdEncoding.EncodeToString(imageBytes)
        dataURI := "data:" + contentType + ";base64," + encoded
        if avatarErr := handler.userRepo.UpdateAvatar(ctx, user.ID, dataURI); avatarErr != nil {
            slog.Error("updating avatar during welcome", "error", avatarErr)
        }
    }
}
```

Also add the imports `"encoding/base64"` and `"io"` to the file.

**Step 4: Regenerate templates**

Run: `make templ`
Expected: no errors

**Step 5: Run all tests**

Run: `make test`
Expected: all tests pass

**Step 6: Commit**

```bash
git add templates/pages/welcome.templ templates/pages/welcome_templ.go \
        templates/components/welcome_steps.templ templates/components/welcome_steps_templ.go \
        internal/handlers/onboarding.go internal/handlers/onboarding_test.go
git commit -m "feat: add welcome flow handler and templates"
```

---

### Task 8: Router wiring

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Wire up the handler and routes**

In `server.go`, after the existing handler instantiations, add:

```go
onboardingHandler := handlers.NewOnboardingHandler(settingsRepo, userRepo, categoryRepo)
```

Inside the `router.Group` that uses `middleware.RequireAuth(authService)`, add `RequireOnboarding` to the group and register the exempt routes **before** applying it:

```go
router.Group(func(r chi.Router) {
    r.Use(middleware.RequireAuth(authService))

    // Onboarding routes — exempt from RequireOnboarding
    r.Get("/setup", onboardingHandler.SetupPage)
    r.Post("/setup/family-name", onboardingHandler.SaveFamilyName)
    r.Post("/setup/acknowledge-users", onboardingHandler.AcknowledgeUsers)
    r.Post("/setup/first-category", onboardingHandler.CompleteSetup)

    r.Get("/welcome", onboardingHandler.WelcomePage)
    r.Post("/welcome/start", onboardingHandler.WelcomeStart)
    r.Post("/welcome/profile", onboardingHandler.CompleteWelcome)

    // All other authenticated routes have onboarding enforced
    r.Group(func(r chi.Router) {
        r.Use(middleware.RequireOnboarding(settingsRepo))

        r.Get("/", dashboardHandler.Dashboard)
        // ... all existing routes move inside this inner group
    })
})
```

**Important:** Move all existing authenticated routes (dashboard, chores, meals, recipes, admin, etc.) into the inner group that applies `RequireOnboarding`. The setup/welcome routes stay in the outer group.

**Step 2: Run tests**

Run: `make test`
Expected: all tests pass

**Step 3: Smoke test manually**

Run: `make dev`
- Fresh install: visiting `/` should redirect to `/setup`
- After completing setup: visiting `/` as a new member should redirect to `/welcome`
- After completing welcome: visiting `/` should load the dashboard

**Step 4: Commit**

```bash
git add internal/server/server.go
git commit -m "feat: wire onboarding routes and RequireOnboarding middleware"
```
