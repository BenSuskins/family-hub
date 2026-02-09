package database

import (
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
	if count != 1 {
		t.Errorf("expected 1 migration, got %d", count)
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
	if count != 1 {
		t.Errorf("expected 1 migration after double run, got %d", count)
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
