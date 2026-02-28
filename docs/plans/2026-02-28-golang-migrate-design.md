# Design: Adopt golang-migrate for Database Migrations

**Date:** 2026-02-28

## Context

The project uses a hand-rolled migration runner in `internal/database/migrate.go`. It applies
`*.up.sql` files from an embedded FS, tracking applied versions in a `schema_migrations` table.
The goal is to replace this with `golang-migrate`, a well-maintained library that supports the
same file naming convention and works with `modernc.org/sqlite` (pure Go, no CGO).

## Scope

- Up-only migrations (no down migrations)
- No baselining — existing 12 migration files stay as-is
- Library only — no CLI tool

## Dependencies

- `github.com/golang-migrate/migrate/v4` — core library
- `github.com/golang-migrate/migrate/v4/database/sqlite` — modernc.org/sqlite driver
- `github.com/golang-migrate/migrate/v4/source/iofs` — embed.FS migration source

## Implementation

### migrate.go

Replace the custom implementation with ~10 lines using the golang-migrate API:

1. Create an `iofs.Source` from the embedded `migrations/*.sql` FS
2. Create a `sqlite.Driver` from the `*sql.DB`
3. Instantiate a `migrate.Migrate` with both
4. Call `Up()`, ignoring `migrate.ErrNoChange`

### schema_migrations table migration

golang-migrate stores only the latest applied version: `(version bigint, dirty boolean)`.
The current custom runner stores one row per migration: `(version INTEGER, applied_at TIMESTAMP)`.

The `Migrate()` function must handle this one-time conversion for existing databases:

1. Detect old format by checking for the `applied_at` column in `schema_migrations`
2. Read the max version from the old table
3. Drop the old table
4. Create the new golang-migrate table and insert `(version=maxVersion, dirty=false)`
5. Proceed with `Up()` — this will be a no-op for existing DBs

New databases get the golang-migrate table created automatically.

### migrate_test.go

Update tests to use in-memory SQLite and verify golang-migrate behaviour (applies migrations,
handles already-applied state, handles the old table format conversion).

## Files Changed

- `go.mod` / `go.sum` — add three new dependencies
- `internal/database/migrate.go` — full replacement
- `internal/database/migrate_test.go` — updated tests
