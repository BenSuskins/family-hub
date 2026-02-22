# Dev Auth Bypass Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** When OIDC is not configured, hitting `/login` automatically creates (or fetches) a dev admin user, sets a session cookie, and redirects to `/`.

**Architecture:** Add `DevLogin` to `AuthService` to encapsulate the dev user logic, then update `LoginPage` to call it instead of returning 503. The middleware, session handling, and DB layer are unchanged.

**Tech Stack:** Go, gorilla/securecookie, SQLite via `internal/testutil.NewTestDatabase`, `go test`

---

### Task 1: Add `DevLogin` to `AuthService`

**Files:**
- Modify: `internal/services/auth.go`
- Create: `internal/services/auth_test.go`

**Step 1: Write the failing tests**

Create `internal/services/auth_test.go`:

```go
package services

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func newDevAuthService(t *testing.T) *AuthService {
	t.Helper()
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	service, err := NewAuthService(context.Background(), config.Config{SessionSecret: "test-secret"}, userRepo)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}
	return service
}

func TestDevLogin_CreatesDevAdminUser(t *testing.T) {
	service := newDevAuthService(t)

	user, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("DevLogin: %v", err)
	}

	if user.Name != "Dev Admin" {
		t.Errorf("expected name 'Dev Admin', got %q", user.Name)
	}
	if user.Email != "dev@localhost" {
		t.Errorf("expected email 'dev@localhost', got %q", user.Email)
	}
	if user.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got %q", user.Role)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestDevLogin_IdempotentOnSecondCall(t *testing.T) {
	service := newDevAuthService(t)

	first, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("first DevLogin: %v", err)
	}

	second, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("second DevLogin: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same user ID, got %q and %q", first.ID, second.ID)
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/services/... -run TestDevLogin -v
```

Expected: `FAIL` — `service.DevLogin undefined`

**Step 3: Add `DevLogin` to `internal/services/auth.go`**

Add the constant and method after the `OIDCConfigured` method (around line 69):

```go
const devUserOIDCSubject = "dev-user"

func (service *AuthService) DevLogin(ctx context.Context) (models.User, error) {
	slog.Warn("dev auto-login, do not use in production")

	existing, err := service.userRepo.FindByOIDCSubject(ctx, devUserOIDCSubject)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) && !isNotFound(err) {
		return models.User{}, fmt.Errorf("looking up dev user: %w", err)
	}

	return service.userRepo.Create(ctx, models.User{
		OIDCSubject: devUserOIDCSubject,
		Email:       "dev@localhost",
		Name:        "Dev Admin",
		Role:        models.RoleAdmin,
	})
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/services/... -run TestDevLogin -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/services/auth.go internal/services/auth_test.go
git commit -m "feat: add DevLogin to AuthService for local dev without OIDC"
```

---

### Task 2: Update `LoginPage` handler to call `DevLogin`

**Files:**
- Modify: `internal/handlers/auth.go`
- Create: `internal/handlers/auth_test.go`

**Step 1: Write the failing test**

Create `internal/handlers/auth_test.go`:

```go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestLoginPage_DevAutoLogin_WhenOIDCNotConfigured(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)

	authService, err := services.NewAuthService(
		context.Background(),
		config.Config{SessionSecret: "test-secret"},
		userRepo,
	)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}

	handler := NewAuthHandler(authService)

	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()
	handler.LoginPage(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Errorf("expected 302, got %d\nbody: %s", recorder.Code, recorder.Body.String())
	}

	location := recorder.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %q", location)
	}

	var sessionCookie string
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie.Value
			break
		}
	}
	if sessionCookie == "" {
		t.Error("expected session cookie to be set")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/handlers/... -run TestLoginPage_DevAutoLogin -v
```

Expected: `FAIL` — returns 503 instead of 302

**Step 3: Update `LoginPage` in `internal/handlers/auth.go`**

Replace the `!OIDCConfigured()` branch (lines 19–22):

```go
// Before:
if !handler.authService.OIDCConfigured() {
    http.Error(w, "OIDC not configured", http.StatusServiceUnavailable)
    return
}

// After:
if !handler.authService.OIDCConfigured() {
    user, err := handler.authService.DevLogin(r.Context())
    if err != nil {
        slog.Error("dev login", "error", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    if err := handler.authService.SetSession(w, user.ID); err != nil {
        slog.Error("setting session for dev login", "error", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/", http.StatusFound)
    return
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./internal/handlers/... -run TestLoginPage_DevAutoLogin -v
```

Expected: `PASS`

**Step 5: Run all tests to check for regressions**

```bash
go test ./...
```

Expected: all `PASS`

**Step 6: Commit**

```bash
git add internal/handlers/auth.go internal/handlers/auth_test.go
git commit -m "feat: dev auto-login when OIDC not configured"
```
