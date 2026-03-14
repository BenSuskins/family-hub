package handlers

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"
	"time"

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

func TestBackupHandler_Restore_ValidBackup_RestartsServer(t *testing.T) {
	sourceDB := testutil.NewTestDatabase(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "family-hub.db")

	backupGz := createTestBackupGz(t, sourceDB)

	exitCalled := make(chan struct{})
	handler := &BackupHandler{
		db:           sourceDB,
		databasePath: dbPath,
		exitFunc:     func(code int) { close(exitCalled) },
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

	select {
	case <-exitCalled:
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Error("expected exitFunc to be called after successful restore")
	}
}

func TestBackupHandler_Restore_InvalidFileExtension_Returns400(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tmpDir := t.TempDir()

	exitCalled := false
	handler := &BackupHandler{
		db:           database,
		databasePath: filepath.Join(tmpDir, "family-hub.db"),
		exitFunc:     func(int) { exitCalled = true },
	}

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
