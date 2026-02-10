package database

import (
	"os"
	"path/filepath"
	"runtime"
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

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("querying migrations: %v", err)
	}
	want, err := migrationFileCount()
	if err != nil {
		t.Fatalf("counting migration files: %v", err)
	}

	if count != want {
		t.Errorf("expected %d migrations, got %d", want, count)
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

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	// ensure the migration count matches the number of migration files
	want, err := migrationFileCount()
	if err != nil {
		t.Fatalf("counting migration files: %v", err)
	}
	if count != want {
		t.Errorf("expected %d migrations after double run, got %d", want, count)
	}
}

func migrationFileCount() (int, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(thisFile), "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, err
	}
	want := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			want++
		}
	}
	return want, nil
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
