package handlers

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
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
