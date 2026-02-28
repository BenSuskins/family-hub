package database

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestMigrate_Success(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	want, err := latestMigrationVersion()
	if err != nil {
		t.Fatalf("getting latest migration version: %v", err)
	}

	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if version != want {
		t.Errorf("expected version %d, got %d", want, version)
	}
	if dirty {
		t.Error("expected dirty=false after successful migration")
	}
}

func TestMigrate_Idempotent(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second migration should not fail: %v", err)
	}

	want, err := latestMigrationVersion()
	if err != nil {
		t.Fatalf("getting latest migration version: %v", err)
	}

	var version int
	if err := db.QueryRow("SELECT version FROM schema_migrations").Scan(&version); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if version != want {
		t.Errorf("expected version %d after double run, got %d", want, version)
	}
}

func TestMigrate_CreatesAllTables(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	expectedTables := []string{"users", "categories", "chores", "events", "chore_assignments", "api_tokens"}
	for _, table := range expectedTables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table '%s' not found: %v", table, err)
		}
	}
}

func TestMigrate_LegacyTableConversion(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	// Simulate old-format schema_migrations table already present
	if _, err := db.Exec(`
		CREATE TABLE schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("creating legacy table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (5), (6), (7)`); err != nil {
		t.Fatalf("inserting legacy rows: %v", err)
	}

	if err := convertLegacyMigrationsTable(db); err != nil {
		t.Fatalf("convertLegacyMigrationsTable: %v", err)
	}

	// Old rows should be gone; new table should have one row with version=7
	var version int
	var dirty bool
	err = db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Fatalf("querying converted table: %v", err)
	}
	if version != 7 {
		t.Errorf("expected version 7, got %d", version)
	}
	if dirty {
		t.Error("expected dirty=false after conversion")
	}
}

func TestMigrate_LegacyTableConversion_NoOpWhenAlreadyNew(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	// New-format table already present — convertLegacyMigrationsTable should leave it alone
	if _, err := db.Exec(`
		CREATE TABLE schema_migrations (version bigint not null primary key, dirty boolean not null)
	`); err != nil {
		t.Fatalf("creating new-format table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations VALUES (12, false)`); err != nil {
		t.Fatalf("inserting row: %v", err)
	}

	if err := convertLegacyMigrationsTable(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("querying schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

// latestMigrationVersion returns the highest version number found in the migrations directory.
func latestMigrationVersion() (int, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, err
	}
	latest := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		parts := strings.SplitN(e.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}
		v, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if v > latest {
			latest = v
		}
	}
	return latest, nil
}
