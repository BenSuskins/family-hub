# Backup & Restore Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add manual database backup (download `.db.gz`) and restore (upload `.db.gz`, auto-restart) accessible from the admin page.

**Architecture:** A `BackupHandler` uses SQLite's `VACUUM INTO` to snapshot the database to a temp file, gzip-compresses it, and streams it as a download. Restore decompresses an uploaded `.db.gz`, validates it as a real SQLite file, moves it over `DATABASE_PATH`, then calls `os.Exit(0)` — Docker restart policy brings the app back up against the new database.

**Tech Stack:** Go standard library only (`compress/gzip`, `os`, `io`, `net/http`). No new dependencies.

---

### Task 1: BackupHandler — Backup endpoint

**Files:**
- Create: `internal/handlers/backup.go`
- Create: `internal/handlers/backup_test.go`

**Step 1: Write the failing test**

Create `internal/handlers/backup_test.go`:

```go
package handlers

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestBackupHandler_Backup_StreamsGzippedSQLiteFile(t *testing.T) {
	database := testutil.NewTestDatabase(t)

	tmpDir := t.TempDir()
	handler := &BackupHandler{
		db:           database,
		databasePath: filepath.Join(tmpDir, "family-hub.db"),
		exitFunc:     func(int) {},
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/backup", nil)
	w := httptest.NewRecorder()
	handler.Backup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if ct := w.Header().Get("Content-Type"); ct != "application/gzip" {
		t.Errorf("expected Content-Type application/gzip, got %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd == "" {
		t.Error("expected Content-Disposition header to be set")
	}

	gz, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("response body is not valid gzip: %v", err)
	}
	defer gz.Close()

	// SQLite databases start with "SQLite format 3\000"
	header := make([]byte, 16)
	if _, err := gz.Read(header); err != nil {
		t.Fatalf("reading gzip content: %v", err)
	}
	want := "SQLite format 3"
	if string(header[:len(want)]) != want {
		t.Errorf("expected SQLite magic bytes, got %q", header)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/handlers/... -run TestBackupHandler_Backup -v
```

Expected: `FAIL` — `BackupHandler` undefined.

**Step 3: Implement the Backup method**

Create `internal/handlers/backup.go`:

```go
package handlers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type BackupHandler struct {
	db           interface {
		ExecContext(ctx interface{ Done() <-chan struct{} }, query string, args ...any) (interface{}, error)
	}
	databasePath string
	exitFunc     func(int)
}
```

Wait — use the real type. Replace the above with:

```go
package handlers

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type BackupHandler struct {
	db           *sql.DB
	databasePath string
	exitFunc     func(int)
}

func NewBackupHandler(db *sql.DB, databasePath string) *BackupHandler {
	return &BackupHandler{
		db:           db,
		databasePath: databasePath,
		exitFunc:     os.Exit,
	}
}

func (handler *BackupHandler) Backup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tempFile, err := os.CreateTemp("", "family-hub-backup-*.db")
	if err != nil {
		slog.Error("creating temp backup file", "error", err)
		http.Error(w, "backup failed", http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if _, err := handler.db.ExecContext(ctx, "VACUUM INTO ?", tempPath); err != nil {
		slog.Error("vacuuming database to backup", "error", err)
		http.Error(w, "backup failed", http.StatusInternalServerError)
		return
	}

	source, err := os.Open(tempPath)
	if err != nil {
		slog.Error("opening backup file", "error", err)
		http.Error(w, "backup failed", http.StatusInternalServerError)
		return
	}
	defer source.Close()

	filename := fmt.Sprintf("family-hub-backup-%s.db.gz", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Content-Type", "application/gzip")

	gzWriter := gzip.NewWriter(w)
	defer gzWriter.Close()

	if _, err := io.Copy(gzWriter, source); err != nil {
		slog.Error("streaming backup", "error", err)
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/handlers/... -run TestBackupHandler_Backup -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add internal/handlers/backup.go internal/handlers/backup_test.go
git commit -m "feat: add backup handler with VACUUM INTO download"
```

---

### Task 2: BackupHandler — Restore endpoint

**Files:**
- Modify: `internal/handlers/backup.go`
- Modify: `internal/handlers/backup_test.go`

**Step 1: Write the failing tests**

Append to `internal/handlers/backup_test.go`:

```go
func TestBackupHandler_Restore_ValidBackup_RestartsServer(t *testing.T) {
	sourceDB := testutil.NewTestDatabase(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "family-hub.db")

	// Create a valid .db.gz from the test database
	backupGz := createTestBackupGz(t, sourceDB)

	exitCalled := false
	handler := &BackupHandler{
		db:           sourceDB,
		databasePath: dbPath,
		exitFunc:     func(code int) { exitCalled = true },
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/restore", backupGz.body)
	req.Header.Set("Content-Type", backupGz.contentType)
	w := httptest.NewRecorder()
	handler.Restore(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected restored database file to exist at dbPath")
	}
	if !exitCalled {
		t.Error("expected exitFunc to be called after successful restore")
	}
}

func TestBackupHandler_Restore_InvalidFile_Returns400(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tmpDir := t.TempDir()

	exitCalled := false
	handler := &BackupHandler{
		db:           database,
		databasePath: filepath.Join(tmpDir, "family-hub.db"),
		exitFunc:     func(int) { exitCalled = true },
	}

	// Build a multipart form with a .txt file (wrong extension)
	body, contentType := buildMultipartFile(t, "backup.txt", []byte("not a database"))

	req := httptest.NewRequest(http.MethodPost, "/admin/restore", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	handler.Restore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if exitCalled {
		t.Error("expected exitFunc NOT to be called on invalid file")
	}
}

func TestBackupHandler_Restore_CorruptGzip_Returns400(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tmpDir := t.TempDir()

	exitCalled := false
	handler := &BackupHandler{
		db:           database,
		databasePath: filepath.Join(tmpDir, "family-hub.db"),
		exitFunc:     func(int) { exitCalled = true },
	}

	body, contentType := buildMultipartFile(t, "backup.db.gz", []byte("not valid gzip data"))

	req := httptest.NewRequest(http.MethodPost, "/admin/restore", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	handler.Restore(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if exitCalled {
		t.Error("expected exitFunc NOT to be called on corrupt gzip")
	}
}
```

Add these helpers at the bottom of the test file:

```go
import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
)

type testBackupGz struct {
	body        io.Reader
	contentType string
}

func createTestBackupGz(t *testing.T, db *sql.DB) testBackupGz {
	t.Helper()

	tempPath := filepath.Join(t.TempDir(), "source.db")
	if _, err := db.ExecContext(t.Context(), "VACUUM INTO ?", tempPath); err != nil {
		t.Fatalf("creating backup for test: %v", err)
	}

	raw, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("reading backup file: %v", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(raw); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	gz.Close()

	body, contentType := buildMultipartFile(t, "backup.db.gz", buf.Bytes())
	return testBackupGz{body: body, contentType: contentType}
}

func buildMultipartFile(t *testing.T, filename string, data []byte) (io.Reader, string) {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="backup"; filename="%s"`, filename))
	header.Set("Content-Type", "application/octet-stream")

	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("creating multipart: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("writing multipart: %v", err)
	}
	writer.Close()

	return &buf, writer.FormDataContentType()
}
```

> Note: you'll need to add `"fmt"` to the test file imports if it's not already there.

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/handlers/... -run TestBackupHandler_Restore -v
```

Expected: `FAIL` — `Restore` method undefined.

**Step 3: Implement the Restore method**

Append to `internal/handlers/backup.go` (add needed imports: `compress/gzip`, `database/sql`, `io`, `os`, `path/filepath`, `strings`, `time`, `github.com/bensuskins/family-hub/internal/database`):

```go
func (handler *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("backup")
	if err != nil {
		http.Error(w, "backup file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(fileHeader.Filename, ".db.gz") {
		http.Error(w, "file must be a .db.gz archive", http.StatusBadRequest)
		return
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		http.Error(w, "invalid gzip archive", http.StatusBadRequest)
		return
	}
	defer gzReader.Close()

	tempFile, err := os.CreateTemp("", "family-hub-restore-*.db")
	if err != nil {
		slog.Error("creating temp restore file", "error", err)
		http.Error(w, "restore failed", http.StatusInternalServerError)
		return
	}
	tempPath := tempFile.Name()

	if _, err := io.Copy(tempFile, gzReader); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		http.Error(w, "invalid backup file", http.StatusBadRequest)
		return
	}
	tempFile.Close()

	validationDB, err := database.Open(tempPath)
	if err != nil {
		os.Remove(tempPath)
		http.Error(w, "uploaded file is not a valid SQLite database", http.StatusBadRequest)
		return
	}
	validationDB.Close()

	if err := replaceFile(tempPath, handler.databasePath); err != nil {
		os.Remove(tempPath)
		slog.Error("replacing database file", "error", err)
		http.Error(w, "restore failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database restored successfully. Server is restarting..."))

	go func() {
		time.Sleep(100 * time.Millisecond)
		handler.exitFunc(0)
	}()
}

// replaceFile attempts os.Rename first (atomic on same filesystem),
// falling back to copy+remove for cross-device moves.
func replaceFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	sourceFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return fmt.Errorf("copying file: %w", err)
	}

	return os.Remove(source)
}
```

**Step 4: Fix imports in backup.go**

The final import block for `backup.go`:

```go
import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/database"
)
```

**Step 5: Run all backup tests**

```bash
go test ./internal/handlers/... -run TestBackupHandler -v
```

Expected: all `PASS`

**Step 6: Commit**

```bash
git add internal/handlers/backup.go internal/handlers/backup_test.go
git commit -m "feat: add restore endpoint with validation and auto-restart"
```

---

### Task 3: Admin UI — Database section

**Files:**
- Modify: `templates/pages/admin.templ`

**Context:** The admin template uses the templ framework. Edit the `.templ` source — never edit `_templ.go` files directly. Run `templ generate` after editing.

The admin page already has Hub Settings, Categories, Users, and API Tokens sections. Add a Database section after API Tokens, matching the existing card style.

**Step 1: Add the Database section to admin.templ**

Open `templates/pages/admin.templ`. After the closing `</div>` of the API Tokens section (around line 188), add:

```templ
		<!-- Database -->
		<div>
			<h2 class="text-lg font-medium text-stone-900 dark:text-slate-300 mb-4">Database</h2>
			<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6 space-y-6">
				<div>
					<h3 class="text-sm font-medium text-stone-700 dark:text-slate-300 mb-1">Backup</h3>
					<p class="text-xs text-stone-500 dark:text-slate-400 mb-3">Download a compressed snapshot of the current database.</p>
					<a
						href="/admin/backup"
						class="inline-flex items-center gap-1.5 bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0"
					>
						@components.IconArrowDown("h-4 w-4")
						Download Backup
					</a>
				</div>
				<div class="border-t border-zinc-200 dark:border-slate-700 pt-6">
					<h3 class="text-sm font-medium text-stone-700 dark:text-slate-300 mb-1">Restore</h3>
					<p class="text-xs text-stone-500 dark:text-slate-400 mb-3">Upload a <code>.db.gz</code> backup file. This will overwrite all current data and restart the server.</p>
					<div id="restore-result" class="mb-3"></div>
					<form
						hx-post="/admin/restore"
						hx-target="#restore-result"
						hx-swap="innerHTML"
						hx-encoding="multipart/form-data"
						class="flex gap-3 items-center"
					>
						<input
							type="file"
							name="backup"
							accept=".db.gz"
							required
							class="text-sm text-stone-600 dark:text-slate-300 file:mr-3 file:py-2 file:px-4 file:rounded-xl file:border-0 file:text-sm file:font-medium file:bg-stone-100 dark:file:bg-slate-700 file:text-stone-700 dark:file:text-slate-300 hover:file:bg-stone-200 dark:hover:file:bg-slate-600 file:transition-colors file:duration-150"
						/>
						<button
							type="submit"
							hx-confirm="This will replace ALL current data and restart the server. Are you sure?"
							class="flex-shrink-0 bg-red-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-red-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0"
						>
							Restore
						</button>
					</form>
				</div>
			</div>
		</div>
```

**Step 2: Check if `IconArrowDown` exists**

```bash
grep -r "IconArrowDown" templates/
```

If it does not exist, you need to add it to the icons component. Check `templates/components/` for the pattern used for other icons (e.g. `IconPlus`). Add a matching `IconArrowDown` function that renders an SVG arrow-down icon (Heroicons outline style used throughout):

```templ
templ IconArrowDown(class string) {
	<svg xmlns="http://www.w3.org/2000/svg" class={ class } fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
		<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 13.5 12 21m0 0-7.5-7.5M12 21V3"></path>
	</svg>
}
```

Find the icons file:

```bash
grep -rl "IconPlus" templates/components/
```

Add `IconArrowDown` to that file.

**Step 3: Regenerate templ files**

```bash
templ generate
```

Expected: new `*_templ.go` files generated without errors.

**Step 4: Build to verify compilation**

```bash
go build ./...
```

Expected: no errors.

**Step 5: Commit**

```bash
git add templates/
git commit -m "feat: add database backup/restore section to admin UI"
```

---

### Task 4: Wire handler into the server

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Instantiate and register the BackupHandler**

Open `internal/server/server.go`. In the `New` function, after the existing handler instantiations (around line 50), add:

```go
backupHandler := handlers.NewBackupHandler(database, cfg.DatabasePath)
```

Then in the `RequireAdmin` router group (around line 144), add two new routes:

```go
r.Get("/admin/backup", backupHandler.Backup)
r.Post("/admin/restore", backupHandler.Restore)
```

**Step 2: Build to verify wiring**

```bash
go build ./...
```

Expected: no errors.

**Step 3: Run all tests**

```bash
make test
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/server/server.go
git commit -m "feat: wire backup/restore handler into server routes"
```

---

### Task 5: Manual smoke test

Start the server locally and verify the feature end-to-end:

```bash
make dev
```

1. Log in as an admin and navigate to `/admin/users`
2. Scroll to the Database section — verify "Download Backup" and the restore form are visible
3. Click "Download Backup" — verify a `.db.gz` file is downloaded and is non-empty
4. In the restore form, select the downloaded `.db.gz` and click "Restore"
5. Confirm the dialog — verify the success message appears and the server restarts (Docker logs show restart)
6. After restart, verify the app loads normally

If smoke test passes, the feature is complete.
