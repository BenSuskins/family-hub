package database

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	sqlitedriver "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestMigrate016_DropsRuleColumnsAddsFKPreservesData(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	defer db.Close()

	if err := convertLegacyMigrationsTable(db); err != nil {
		t.Fatalf("legacy conversion: %v", err)
	}
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	defer source.Close()
	driver, err := sqlitedriver.WithInstance(db, &sqlitedriver.Config{NoTxWrap: true})
	if err != nil {
		t.Fatalf("driver: %v", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		t.Fatalf("migrator: %v", err)
	}

	// Migrate to just before the Phase 4d rebuild.
	if err := migrator.Migrate(15); err != nil {
		t.Fatalf("migrate to 15: %v", err)
	}

	// Seed representative pre-016 data using the old schema (rule columns live
	// on chores; no chore_series rows yet).
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("seed exec %q: %v", q, err)
		}
	}
	exec(`INSERT INTO users (id, oidc_subject, email, name, role) VALUES ('u1','s1','u1@x','U1','member')`)
	// A recurring series of two occurrences (series_id set on both).
	exec(`INSERT INTO chores (id, name, description, created_by_user_id, assigned_to_user_id, due_date,
		recurrence_type, recurrence_value, recur_on_complete, series_id, status)
		VALUES ('c1','Bins','',  'u1','u1','2025-01-01','weekly','{"interval":1}',0,'c1','pending')`)
	exec(`INSERT INTO chores (id, name, description, created_by_user_id, assigned_to_user_id, due_date,
		recurrence_type, recurrence_value, recur_on_complete, series_id, status)
		VALUES ('c2','Bins','',  'u1','u1','2025-01-08','weekly','{"interval":1}',0,'c1','pending')`)
	// A legacy recurring chore without series_id.
	exec(`INSERT INTO chores (id, name, description, created_by_user_id, due_date,
		recurrence_type, recurrence_value, recur_on_complete, status)
		VALUES ('c3','Mop','','u1','2025-02-01','monthly','{"interval":1}',0,'pending')`)
	// A non-recurring chore.
	exec(`INSERT INTO chores (id, name, description, created_by_user_id, status)
		VALUES ('c4','One off','','u1','pending')`)
	// Children with ON DELETE CASCADE that must survive the rebuild.
	exec(`INSERT INTO chore_assignments (id, chore_id, user_id, status) VALUES ('a1','c1','u1','assigned')`)
	exec(`INSERT INTO chore_eligible_assignees (chore_id, user_id) VALUES ('c1','u1')`)

	// Apply the Phase 4d rebuild.
	if err := migrator.Migrate(16); err != nil {
		t.Fatalf("migrate to 16: %v", err)
	}

	// Rule columns dropped from chores.
	for _, col := range []string{"recurrence_type", "recurrence_value", "recur_on_complete", "recurrence_until", "recurrence_count"} {
		if _, err := db.Exec("SELECT " + col + " FROM chores LIMIT 1"); err == nil {
			t.Errorf("expected column %s to be dropped from chores", col)
		}
	}

	// Children preserved (no cascade).
	if n := scalar(t, db, "SELECT COUNT(*) FROM chore_assignments"); n != 1 {
		t.Errorf("chore_assignments lost: got %d want 1", n)
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM chore_eligible_assignees"); n != 1 {
		t.Errorf("chore_eligible_assignees lost: got %d want 1", n)
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM chores"); n != 4 {
		t.Errorf("chores lost: got %d want 4", n)
	}

	// Series backfilled: c1 (shared) and c3 (legacy) become definitions; c4 does not.
	if n := scalar(t, db, "SELECT COUNT(*) FROM chore_series"); n != 2 {
		t.Errorf("expected 2 backfilled series, got %d", n)
	}
	if rt := strScalar(t, db, "SELECT recurrence_type FROM chore_series WHERE id='c1'"); rt != "weekly" {
		t.Errorf("series c1 recurrence_type = %q, want weekly", rt)
	}
	// Legacy chore c3 had its series_id set to its own id.
	if sid := strScalar(t, db, "SELECT series_id FROM chores WHERE id='c3'"); sid != "c3" {
		t.Errorf("legacy chore series_id = %q, want c3", sid)
	}
	// Anchor eligible pool copied to the series.
	if n := scalar(t, db, "SELECT COUNT(*) FROM chore_series_eligible_assignees WHERE series_id='c1'"); n != 1 {
		t.Errorf("series eligible pool not backfilled: got %d want 1", n)
	}

	// New FK enforced on series_id; NULL still allowed.
	if _, err := db.Exec(`INSERT INTO chores (id,name,created_by_user_id,series_id,status) VALUES ('bad','B','u1','nope','pending')`); err == nil || !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Errorf("expected FK violation for bogus series_id, got: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chores (id,name,created_by_user_id,series_id,status) VALUES ('ok','B','u1',NULL,'pending')`); err != nil {
		t.Errorf("NULL series_id should be allowed: %v", err)
	}
}

func scalar(t *testing.T, db *sql.DB, q string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q).Scan(&n); err != nil {
		t.Fatalf("scalar %q: %v", q, err)
	}
	return n
}

func strScalar(t *testing.T, db *sql.DB, q string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(q).Scan(&s); err != nil {
		t.Fatalf("strScalar %q: %v", q, err)
	}
	return s
}
