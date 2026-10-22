package handlers

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
		writeJSONError(w, http.StatusInternalServerError, "backup failed")
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	os.Remove(tempPath) // VACUUM INTO requires destination to not exist
	defer os.Remove(tempPath)

	if _, err := handler.db.ExecContext(ctx, "VACUUM INTO ?", tempPath); err != nil {
		slog.Error("vacuuming database to backup", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "backup failed")
		return
	}

	source, err := os.Open(tempPath)
	if err != nil {
		slog.Error("opening backup file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "backup failed")
		return
	}
	defer source.Close()

	filename := fmt.Sprintf("family-hub-backup-%s.db.gz", time.Now().Format(DateFormat))
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

func (handler *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}

	file, fileHeader, err := r.FormFile("backup")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "backup file required")
		return
	}
	defer file.Close()

	if !strings.HasSuffix(fileHeader.Filename, ".db.gz") {
		writeJSONError(w, http.StatusBadRequest, "file must be a .db.gz archive")
		return
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid gzip archive")
		return
	}
	defer gzReader.Close()

	tempFile, err := os.CreateTemp("", "family-hub-restore-*.db")
	if err != nil {
		slog.Error("creating temp restore file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "restore failed")
		return
	}
	tempPath := tempFile.Name()

	if _, err := io.Copy(tempFile, gzReader); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		writeJSONError(w, http.StatusBadRequest, "invalid backup file")
		return
	}
	tempFile.Close()

	validationDB, err := database.Open(tempPath)
	if err != nil {
		os.Remove(tempPath)
		writeJSONError(w, http.StatusBadRequest, "uploaded file is not a valid SQLite database")
		return
	}
	validationDB.Close()

	if err := replaceFile(tempPath, handler.databasePath); err != nil {
		os.Remove(tempPath)
		slog.Error("replacing database file", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "restore failed")
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database restored successfully. Server is restarting..."))

	// Flush response before exiting so the client sees the message.
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Remove stale WAL/SHM files after the response is sent so that the
		// restarted process doesn't try to apply the old write-ahead log
		// against the freshly restored database.
		os.Remove(handler.databasePath + "-wal")
		os.Remove(handler.databasePath + "-shm")
		handler.exitFunc(0)
	}()
}

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

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		destinationFile.Close()
		os.Remove(destination)
		return fmt.Errorf("copying file: %w", err)
	}
	destinationFile.Close()

	return os.Remove(source)
}
