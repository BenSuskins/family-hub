# Backup & Restore — Design

Date: 2026-02-28

## Summary

Add a manual backup and restore capability to the Family Hub admin page. Admins can download a compressed snapshot of the SQLite database and upload a previous snapshot to restore it. Restore triggers an automatic server restart via `os.Exit(0)` with Docker restart policy.

## Approach

SQLite `VACUUM INTO` for backup — the canonical online backup mechanism. Safe under concurrent reads and writes. Backup format is gzip-compressed SQLite (`.db.gz`).

## Endpoints

| Method | Path            | Description                                      |
|--------|-----------------|--------------------------------------------------|
| GET    | /admin/backup   | Stream database backup as `.db.gz` download      |
| POST   | /admin/restore  | Upload `.db.gz`, validate, swap file, exit       |

Both routes sit inside the existing `RequireAuth` + `RequireAdmin` middleware group.

## Backup Flow

1. Execute `VACUUM INTO '/tmp/family-hub-backup-<unix-timestamp>.db'` via `*sql.DB`
2. Open the temp file, wrap in `gzip.NewWriter`, stream to `http.ResponseWriter`
   - `Content-Disposition: attachment; filename="family-hub-backup-YYYY-MM-DD.db.gz"`
   - `Content-Type: application/gzip`
3. Delete the temp file on completion (deferred)

## Restore Flow

1. Receive multipart upload; reject anything that is not `.db.gz` (400)
2. Decompress into a temp file via `compress/gzip`
3. Open the temp file with `database.Open()` to validate it is a real SQLite database
4. Close the validation connection, delete temp file on failure (400), original DB untouched
5. Move temp file over `DATABASE_PATH` (atomic on same filesystem; copy+replace otherwise)
6. Call `exitFunc(0)` — Docker restart policy brings the app back against the new database

## New Code

- `internal/handlers/backup.go` — `BackupHandler` with `Backup` and `Restore` methods
- `templates/pages/admin.templ` — "Database" section appended below API Tokens
- `internal/server/server.go` — wire `BackupHandler` and two new routes

## Error Handling

- Backup failure before streaming begins → 500, no partial download
- Invalid file extension or MIME → 400, original DB untouched
- Decompression or SQLite validation failure → 400, temp file cleaned up, original DB untouched
- File move failure → 500, original DB untouched

## Testing

- `BackupHandler` accepts `databasePath string` and an `exitFunc func(int)` for injection
- Backup test: in-memory DB, assert response is valid gzip containing SQLite magic bytes (`53 51 4C 69 74 65`)
- Restore test: generate a valid `.db.gz` fixture, POST it, assert 200 and that `exitFunc` was called with 0
- Restore validation test: POST invalid file, assert 400 and `exitFunc` not called
