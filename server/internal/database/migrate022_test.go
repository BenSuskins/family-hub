package database

import (
	"testing"

	"github.com/golang-migrate/migrate/v4"
	sqlitedriver "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestMigrate022PreservesHistoryAndOldestOwner(t *testing.T) {
	connection, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	driver, err := sqlitedriver.WithInstance(connection, &sqlitedriver.Config{NoTxWrap: true})
	if err != nil {
		t.Fatal(err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrator.Migrate(21); err != nil {
		t.Fatal(err)
	}
	_, err = connection.Exec(`
 INSERT INTO users (id,oidc_subject,email,name) VALUES ('ben','ben','ben@x','Ben'),('alex','alex','alex@x','Alex');
 INSERT INTO chore_series (id,name,created_by_user_id,recurrence_type,rotation_cursor_user_id) VALUES ('bins','Bins','ben','daily','alex');
 INSERT INTO chores (id,name,created_by_user_id,series_id,due_date,assigned_to_user_id,status) VALUES
 ('old','Bins','ben','bins','2026-10-01','ben','overdue'),
 ('duplicate','Bins','ben','bins','2026-10-02','alex','overdue'),
 ('future','Bins','ben','bins','2026-11-01','alex','pending'),
 ('done','Bins','ben','bins','2026-09-01','ben','completed'),
 ('one','One off','ben',NULL,NULL,'ben','pending');
 INSERT INTO chore_assignments (id,chore_id,user_id) VALUES ('assignment','old','ben'),('helped','done','ben');
UPDATE chores SET completed_by_user_id='alex',completed_at='2026-09-01' WHERE id='done';
 INSERT INTO chore_eligible_assignees (chore_id,user_id) VALUES ('old','ben');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrator.Migrate(22); err != nil {
		t.Fatal(err)
	}
	if got := scalar(t, connection, "SELECT COUNT(*) FROM chores"); got != 5 {
		t.Fatalf("lost history: %d", got)
	}
	if got := scalar(t, connection, "SELECT COUNT(*) FROM chores WHERE status='superseded'"); got != 2 {
		t.Fatalf("superseded: %d", got)
	}
	if got := strScalar(t, connection, "SELECT assigned_to_user_id FROM chores WHERE id='old'"); got != "ben" {
		t.Fatal(got)
	}
	if got := strScalar(t, connection, "SELECT rotation_cursor_user_id FROM chore_series WHERE id='bins'"); got != "ben" {
		t.Fatal(got)
	}
	if got := strScalar(t, connection, "SELECT series_id FROM chores WHERE id='one'"); got != "one" {
		t.Fatal(got)
	}
	if got := scalar(t, connection, "SELECT COUNT(*) FROM chore_assignments"); got != 3 {
		t.Fatal(got)
	}
	if got := strScalar(t, connection, "SELECT user_id FROM chore_assignments WHERE chore_id='done' AND status='completed'"); got != "alex" {
		t.Fatal(got)
	}
	rows, err := connection.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("broken foreign keys")
	}
	rows.Close()
	if _, err := connection.Exec(`INSERT INTO chores (id,name,created_by_user_id,series_id) VALUES ('extra','Bins','ben','bins')`); err == nil {
		t.Fatal("duplicate outstanding task accepted")
	}
	if err := Migrate(connection); err != nil {
		t.Fatal(err)
	}
}
