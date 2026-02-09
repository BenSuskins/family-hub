package testutil

import (
	"database/sql"
	"testing"

	"github.com/bensuskins/family-hub/internal/database"
)

func NewTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
