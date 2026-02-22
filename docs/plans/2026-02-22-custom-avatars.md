# Custom User Avatars Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow each user to upload a custom avatar image stored as a data URI in the database, served via a dedicated `/avatar/{userID}` endpoint, without being overwritten by OIDC on login.

**Architecture:** Migration adds an `avatar_data TEXT` column. Three new repository methods manage it without bloating `FindAll` (which excludes the column). The `provisionUser` auth flow skips the OIDC avatar update when `avatar_data` is non-empty. A new `ProfileHandler` exposes a profile page, upload endpoint, remove endpoint, and serve endpoint. The sidebar user section becomes a link to `/profile`.

**Tech Stack:** Go, chi router, templ, HTMX, SQLite (modernc.org/sqlite), `encoding/base64`, `net/http` multipart

---

## Task 1: Add Migration

**Files:**
- Create: `internal/database/migrations/011_user_avatar_data.up.sql`

**Step 1: Write the migration**

```sql
ALTER TABLE users ADD COLUMN avatar_data TEXT NOT NULL DEFAULT '';
```

**Step 2: Verify it runs**

```bash
make test
```

Expected: all tests pass (the migration runner picks up the new file automatically via `testutil.NewTestDatabase`)

**Step 3: Commit**

```bash
git add internal/database/migrations/011_user_avatar_data.up.sql
git commit -m "feat: add avatar_data column to users"
```

---

## Task 2: Extend UserRepository Interface and Implementation

**Files:**
- Modify: `internal/repository/users.go`
- Modify: `internal/repository/users_test.go`

### Step 1: Write the failing tests

Add to `internal/repository/users_test.go`:

```go
func TestUserRepository_UpdateAvatar(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, models.User{
		OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleMember,
	})

	dataURI := "data:image/png;base64,iVBORw0KGgo="

	if err := repo.UpdateAvatar(ctx, created.ID, dataURI); err != nil {
		t.Fatalf("UpdateAvatar: %v", err)
	}

	avatarData, err := repo.FindAvatarData(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindAvatarData: %v", err)
	}
	if avatarData != dataURI {
		t.Errorf("expected %q, got %q", dataURI, avatarData)
	}

	found, _ := repo.FindByID(ctx, created.ID)
	if found.AvatarURL != "/avatar/"+created.ID {
		t.Errorf("expected avatar_url '/avatar/%s', got %q", created.ID, found.AvatarURL)
	}
}

func TestUserRepository_ClearAvatar(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, models.User{
		OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleMember,
	})

	repo.UpdateAvatar(ctx, created.ID, "data:image/png;base64,abc=")

	if err := repo.ClearAvatar(ctx, created.ID); err != nil {
		t.Fatalf("ClearAvatar: %v", err)
	}

	avatarData, err := repo.FindAvatarData(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindAvatarData after clear: %v", err)
	}
	if avatarData != "" {
		t.Errorf("expected empty avatar_data after clear, got %q", avatarData)
	}

	found, _ := repo.FindByID(ctx, created.ID)
	if found.AvatarURL != "" {
		t.Errorf("expected empty avatar_url after clear, got %q", found.AvatarURL)
	}
}

func TestUserRepository_FindAvatarData_EmptyByDefault(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, models.User{
		OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleMember,
	})

	avatarData, err := repo.FindAvatarData(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindAvatarData: %v", err)
	}
	if avatarData != "" {
		t.Errorf("expected empty avatar_data for new user, got %q", avatarData)
	}
}
```

### Step 2: Run tests to confirm they fail

```bash
make test 2>&1 | grep -A3 "FAIL\|undefined"
```

Expected: compile errors about undefined `UpdateAvatar`, `ClearAvatar`, `FindAvatarData`

### Step 3: Extend the interface in `internal/repository/users.go`

Add three methods to the `UserRepository` interface (after `UpdateProfile`):

```go
FindAvatarData(ctx context.Context, userID string) (string, error)
UpdateAvatar(ctx context.Context, userID string, dataURI string) error
ClearAvatar(ctx context.Context, userID string) error
```

Add three methods to `SQLiteUserRepository` (after the existing `UpdateProfile` implementation):

```go
func (repository *SQLiteUserRepository) FindAvatarData(ctx context.Context, userID string) (string, error) {
	var avatarData string
	err := repository.database.QueryRowContext(ctx,
		"SELECT avatar_data FROM users WHERE id = ?", userID,
	).Scan(&avatarData)
	if err != nil {
		return "", fmt.Errorf("finding avatar data: %w", err)
	}
	return avatarData, nil
}

func (repository *SQLiteUserRepository) UpdateAvatar(ctx context.Context, userID string, dataURI string) error {
	avatarURL := "/avatar/" + userID
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET avatar_data = ?, avatar_url = ?, updated_at = ? WHERE id = ?",
		dataURI, avatarURL, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("updating avatar: %w", err)
	}
	return nil
}

func (repository *SQLiteUserRepository) ClearAvatar(ctx context.Context, userID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET avatar_data = '', avatar_url = '', updated_at = ? WHERE id = ?",
		time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("clearing avatar: %w", err)
	}
	return nil
}
```

### Step 4: Run tests to confirm they pass

```bash
make test 2>&1 | grep -E "FAIL|PASS|ok"
```

Expected: all PASS

### Step 5: Commit

```bash
git add internal/repository/users.go internal/repository/users_test.go
git commit -m "feat: add FindAvatarData, UpdateAvatar, ClearAvatar to UserRepository"
```

---

## Task 3: Preserve Custom Avatar During OIDC Login

**Files:**
- Modify: `internal/services/auth.go`
- Modify: `internal/services/auth_test.go`

### Step 1: Write the failing test

Add to `internal/services/auth_test.go`:

```go
func TestProvisionUser_PreservesCustomAvatarOnLogin(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	service, err := NewAuthService(context.Background(), config.Config{SessionSecret: "test-secret"}, userRepo)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}
	ctx := context.Background()

	// First login creates the user with an OIDC avatar
	user, err := service.provisionUser(ctx, "subject-123", "alice@test.com", "Alice", "https://oidc.example.com/pic.png")
	if err != nil {
		t.Fatalf("first provisionUser: %v", err)
	}

	// User uploads a custom avatar
	if err := userRepo.UpdateAvatar(ctx, user.ID, "data:image/png;base64,abc="); err != nil {
		t.Fatalf("UpdateAvatar: %v", err)
	}

	// Second login with a different OIDC avatar URL
	returned, err := service.provisionUser(ctx, "subject-123", "alice@test.com", "Alice", "https://oidc.example.com/new-pic.png")
	if err != nil {
		t.Fatalf("second provisionUser: %v", err)
	}

	// Custom avatar URL must be preserved
	if returned.AvatarURL != "/avatar/"+user.ID {
		t.Errorf("expected custom avatar URL '/avatar/%s', got %q", user.ID, returned.AvatarURL)
	}
}
```

### Step 2: Run to confirm it fails

```bash
go test ./internal/services/ -run TestProvisionUser_PreservesCustomAvatarOnLogin -v
```

Expected: FAIL — custom avatar URL gets overwritten by OIDC URL

### Step 3: Update `provisionUser` in `internal/services/auth.go`

Replace the `if err == nil` branch inside `provisionUser`:

```go
if err == nil {
    effectiveAvatarURL := avatarURL
    avatarData, avatarErr := service.userRepo.FindAvatarData(ctx, existingUser.ID)
    if avatarErr == nil && avatarData != "" {
        effectiveAvatarURL = existingUser.AvatarURL
    }
    if err := service.userRepo.UpdateProfile(ctx, existingUser.ID, name, email, effectiveAvatarURL); err != nil {
        slog.Warn("failed to update user profile on login", "error", err)
    }
    existingUser.Name = name
    existingUser.Email = email
    existingUser.AvatarURL = effectiveAvatarURL
    return existingUser, nil
}
```

### Step 4: Run tests to confirm they pass

```bash
make test 2>&1 | grep -E "FAIL|PASS|ok"
```

Expected: all PASS

### Step 5: Commit

```bash
git add internal/services/auth.go internal/services/auth_test.go
git commit -m "feat: preserve custom avatar during OIDC login sync"
```

---

## Task 4: Profile Handler

**Files:**
- Create: `internal/handlers/profile.go`
- Create: `internal/handlers/profile_test.go`

### Step 1: Write the failing tests

Create `internal/handlers/profile_test.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func setupProfileHandler(t *testing.T) (*ProfileHandler, models.User, repository.UserRepository) {
	t.Helper()
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	user, err := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-1",
		Email:       "alice@test.com",
		Name:        "Alice",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	return NewProfileHandler(userRepo), user, userRepo
}

func multipartUpload(t *testing.T, fieldName, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("copying file content: %v", err)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

func TestProfileHandler_Upload_StoresDataURI(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	// 1x1 pixel PNG (minimal valid image)
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}

	body, contentType := multipartUpload(t, "avatar", "test.png", pngBytes)

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Upload(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}

	avatarData, err := userRepo.FindAvatarData(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindAvatarData: %v", err)
	}
	if !strings.HasPrefix(avatarData, "data:") {
		t.Errorf("expected data URI, got %q", avatarData[:min(len(avatarData), 30)])
	}
}

func TestProfileHandler_Upload_RejectsTooLarge(t *testing.T) {
	handler, user, _ := setupProfileHandler(t)

	// Build a 2MB payload (exceeds 1MB limit)
	largeContent := make([]byte, 2*1024*1024)
	body, contentType := multipartUpload(t, "avatar", "big.jpg", largeContent)

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Upload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProfileHandler_Remove_ClearsAvatar(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	userRepo.UpdateAvatar(context.Background(), user.ID, "data:image/png;base64,abc=")

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar/delete", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Remove(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}

	avatarData, _ := userRepo.FindAvatarData(context.Background(), user.ID)
	if avatarData != "" {
		t.Errorf("expected empty avatar_data after remove, got %q", avatarData)
	}
}

func TestProfileHandler_Serve_ReturnsImageBytes(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	userRepo.UpdateAvatar(context.Background(), user.ID, "data:image/png;base64,iVBORw0KGgo=")

	router := chi.NewRouter()
	router.Get("/avatar/{userID}", handler.Serve)

	req := httptest.NewRequest(http.MethodGet, "/avatar/"+user.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "image/png") {
		t.Errorf("expected image/png content-type, got %q", w.Header().Get("Content-Type"))
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestProfileHandler_Serve_Returns404WhenNoAvatar(t *testing.T) {
	handler, user, _ := setupProfileHandler(t)

	router := chi.NewRouter()
	router.Get("/avatar/{userID}", handler.Serve)

	req := httptest.NewRequest(http.MethodGet, "/avatar/"+user.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

### Step 2: Run to confirm compile failure

```bash
go test ./internal/handlers/ -run TestProfileHandler -v 2>&1 | head -20
```

Expected: compile error — `ProfileHandler` undefined

### Step 3: Create `internal/handlers/profile.go`

```go
package handlers

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

const maxAvatarBytes = 1 * 1024 * 1024 // 1 MB

type ProfileHandler struct {
	userRepo repository.UserRepository
}

func NewProfileHandler(userRepo repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

func (handler *ProfileHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	avatarData, err := handler.userRepo.FindAvatarData(ctx, user.ID)
	if err != nil {
		slog.Error("finding avatar data", "error", err)
	}

	component := pages.Profile(pages.ProfileProps{
		User:            user,
		HasCustomAvatar: avatarData != "",
	})
	component.Render(ctx, w)
}

func (handler *ProfileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseMultipartForm(maxAvatarBytes + 1024); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Missing avatar file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	if len(imageBytes) > maxAvatarBytes {
		http.Error(w, "Image exceeds 1 MB limit", http.StatusBadRequest)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	dataURI := "data:" + contentType + ";base64," + encoded

	if err := handler.userRepo.UpdateAvatar(ctx, user.ID, dataURI); err != nil {
		slog.Error("updating avatar", "error", err)
		http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (handler *ProfileHandler) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := handler.userRepo.ClearAvatar(ctx, user.ID); err != nil {
		slog.Error("clearing avatar", "error", err)
		http.Error(w, "Failed to remove avatar", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (handler *ProfileHandler) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	avatarData, err := handler.userRepo.FindAvatarData(ctx, userID)
	if err != nil || avatarData == "" {
		http.NotFound(w, r)
		return
	}

	// Parse "data:<mime>;base64,<payload>"
	withoutPrefix, ok := strings.CutPrefix(avatarData, "data:")
	if !ok {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(withoutPrefix, ";base64,", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	mimeType := parts[0]
	payload := parts[1]

	imageBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		slog.Error("decoding avatar base64", "error", err)
		http.Error(w, "Corrupted avatar data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	w.Write(imageBytes)
}
```

### Step 4: Run tests to confirm they pass

```bash
go test ./internal/handlers/ -run TestProfileHandler -v
```

Expected: all PASS

### Step 5: Commit

```bash
git add internal/handlers/profile.go internal/handlers/profile_test.go
git commit -m "feat: add ProfileHandler with avatar upload, remove, and serve"
```

---

## Task 5: Profile Page Template

**Files:**
- Create: `templates/pages/profile.templ`

**Note:** After editing `.templ` files, always run `templ generate` before `go build` or `make test`. Add `$(go env GOPATH)/bin` to PATH if `templ` is not found.

### Step 1: Create `templates/pages/profile.templ`

```go
package pages

import (
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/templates/components"
	"github.com/bensuskins/family-hub/templates/layouts"
)

type ProfileProps struct {
	User            models.User
	HasCustomAvatar bool
}

templ Profile(props ProfileProps) {
	@layouts.Base("Profile", props.User, "/profile") {
		<div class="max-w-lg space-y-6">
			<h1 class="text-xl font-semibold text-stone-800 dark:text-slate-100">Profile</h1>
			<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-6">
				<div class="flex items-center gap-4">
					@components.UserAvatar(props.User.Name, props.User.AvatarURL, "h-20 w-20 text-2xl")
					<div>
						<p class="text-sm font-medium text-stone-900 dark:text-slate-100">{ props.User.Name }</p>
						<p class="text-sm text-stone-500 dark:text-slate-400">{ props.User.Email }</p>
					</div>
				</div>
				<div>
					<h2 class="text-sm font-medium text-stone-700 dark:text-slate-300 mb-2">Upload custom avatar</h2>
					<p class="text-xs text-stone-500 dark:text-slate-400 mb-3">Any image format, up to 1 MB.</p>
					<form method="POST" action="/profile/avatar" enctype="multipart/form-data" class="flex gap-3 items-center flex-wrap">
						<input
							type="file"
							name="avatar"
							accept="image/*"
							required
							class="text-sm text-stone-600 dark:text-slate-400 file:mr-3 file:py-2 file:px-3 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-zinc-100 file:text-stone-700 dark:file:bg-slate-700 dark:file:text-slate-200"
						/>
						<button
							type="submit"
							class="bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0"
						>
							Upload
						</button>
					</form>
				</div>
				if props.HasCustomAvatar {
					<div class="border-t border-zinc-100 dark:border-slate-700 pt-4">
						<p class="text-xs text-stone-500 dark:text-slate-400 mb-2">Remove your custom avatar to revert to your account provider photo.</p>
						<form method="POST" action="/profile/avatar/delete">
							<button
								type="submit"
								class="text-sm text-red-600 dark:text-red-400 hover:underline"
							>
								Remove custom avatar
							</button>
						</form>
					</div>
				}
			</div>
		</div>
	}
}
```

### Step 2: Generate the templ output

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
templ generate
```

Expected: generates `templates/pages/profile_templ.go` with no errors

### Step 3: Confirm it compiles

```bash
go build ./...
```

Expected: no errors

### Step 4: Commit

```bash
git add templates/pages/profile.templ templates/pages/profile_templ.go
git commit -m "feat: add profile page template"
```

---

## Task 6: Wire Routes in Server

**Files:**
- Modify: `internal/server/server.go`

### Step 1: Add the profile handler construction

In `server.go`, after the existing handler constructions (e.g. after `icalSubHandler := ...`), add:

```go
profileHandler := handlers.NewProfileHandler(userRepo)
```

### Step 2: Register the routes

Inside the `RequireAuth` group (the `router.Group(func(r chi.Router) { r.Use(middleware.RequireAuth(authService)) ... })` block), add these four routes near the top of the group (after `r.Get("/", dashboardHandler.Dashboard)`):

```go
r.Get("/profile", profileHandler.Page)
r.Post("/profile/avatar", profileHandler.Upload)
r.Post("/profile/avatar/delete", profileHandler.Remove)
r.Get("/avatar/{userID}", profileHandler.Serve)
```

### Step 3: Build and confirm no errors

```bash
go build ./...
```

Expected: no errors

### Step 4: Run all tests

```bash
make test
```

Expected: all PASS

### Step 5: Commit

```bash
git add internal/server/server.go
git commit -m "feat: wire profile and avatar routes"
```

---

## Task 7: Make Sidebar User Section a Link to /profile

**Files:**
- Modify: `templates/layouts/base.templ`

### Step 1: Locate the user section in `base.templ`

Find the `<!-- User section -->` block (lines ~100–115). It currently wraps avatar + name in a plain `<div class="flex items-center gap-3">`.

### Step 2: Wrap the avatar and name with a link

Replace the inner `<div class="flex items-center gap-3">` with a structure that links the avatar + name to `/profile`. The logout button and theme toggle stay outside the link:

```go
<div class="border-t border-zinc-100 dark:border-slate-800 bg-zinc-50 dark:bg-slate-950 px-4 py-3 shrink-0">
    <div class="flex items-center gap-3">
        <a href="/profile" class="flex items-center gap-3 flex-1 min-w-0 hover:opacity-80 transition-opacity duration-150">
            @components.UserAvatar(user.Name, user.AvatarURL, "h-8 w-8 text-xs")
            <p class="text-sm font-medium text-stone-900 dark:text-slate-100 truncate">{ user.Name }</p>
        </a>
        <button onclick="toggleTheme()" title="Toggle dark mode" class="text-stone-400 dark:text-slate-500 hover:text-stone-600 dark:hover:text-slate-300 transition-colors duration-150">
            <span class="hidden dark:block">@components.IconSun("h-5 w-5")</span>
            <span class="dark:hidden">@components.IconMoon("h-5 w-5")</span>
        </button>
        <a href="/logout" class="text-stone-400 dark:text-slate-500 hover:text-stone-600 dark:hover:text-slate-300 transition-colors duration-150" title="Logout">
            @components.IconArrowRightOnRectangle("h-5 w-5")
        </a>
    </div>
</div>
```

Also wrap the mobile top bar avatar in a link. Find the mobile top bar section (lines ~51–54) and change:

```go
<div class="flex items-center gap-2">
    @components.UserAvatar(user.Name, user.AvatarURL, "h-7 w-7 text-xs")
</div>
```

to:

```go
<a href="/profile" class="flex items-center gap-2 hover:opacity-80 transition-opacity duration-150">
    @components.UserAvatar(user.Name, user.AvatarURL, "h-7 w-7 text-xs")
</a>
```

### Step 3: Regenerate templates

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
templ generate
```

Expected: `templates/layouts/base_templ.go` updated, no errors

### Step 4: Build and test

```bash
go build ./...
make test
```

Expected: all PASS

### Step 5: Commit

```bash
git add templates/layouts/base.templ templates/layouts/base_templ.go
git commit -m "feat: link sidebar user section to /profile"
```

---

## Final Verification

```bash
make test
go build ./...
```

Both should pass cleanly. Then start the dev server with `make dev` and manually verify:

1. Navigate to `/profile` — profile page loads with avatar and upload form
2. Upload a small image — avatar updates in the sidebar immediately after redirect
3. Navigate to another page — sidebar shows the new custom avatar
4. Return to `/profile` — "Remove custom avatar" button is visible
5. Click Remove — avatar reverts, button disappears
6. (If OIDC is configured) Log out and log back in — custom avatar persists if one was set
