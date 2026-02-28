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
	os.Remove(tempPath) // VACUUM INTO requires destination to not exist
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
	defer func() {
		if err := gzWriter.Close(); err != nil {
			slog.Error("closing gzip writer for backup", "error", err)
		}
	}()

	if _, err := io.Copy(gzWriter, source); err != nil {
		// Headers already sent; cannot change status code. Client will receive a truncated archive.
		slog.Error("streaming backup", "error", err)
	}
}
