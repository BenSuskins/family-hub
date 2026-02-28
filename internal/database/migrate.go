package database

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	sqlitedriver "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(db *sql.DB) error {
	if err := convertLegacyMigrationsTable(db); err != nil {
		return err
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("creating migration source: %w", err)
	}

	driver, err := sqlitedriver.WithInstance(db, &sqlitedriver.Config{})
	if err != nil {
		return fmt.Errorf("creating sqlite driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}

func convertLegacyMigrationsTable(db *sql.DB) error {
	var hasLegacyColumn int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('schema_migrations') WHERE name = 'applied_at'
	`).Scan(&hasLegacyColumn)
	if err != nil {
		return fmt.Errorf("checking legacy migrations table: %w", err)
	}
	if hasLegacyColumn == 0 {
		return nil
	}

	var maxVersion sql.NullInt64
	if err := db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&maxVersion); err != nil {
		return fmt.Errorf("reading max migration version: %w", err)
	}
	if !maxVersion.Valid {
		if _, err := db.Exec("DROP TABLE schema_migrations"); err != nil {
			return fmt.Errorf("dropping empty legacy migrations table: %w", err)
		}
		return nil
	}

	if _, err := db.Exec("DROP TABLE schema_migrations"); err != nil {
		return fmt.Errorf("dropping legacy migrations table: %w", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE schema_migrations (version bigint not null primary key, dirty boolean not null)
	`); err != nil {
		return fmt.Errorf("creating schema_migrations table: %w", err)
	}

	if _, err := db.Exec("INSERT INTO schema_migrations (version, dirty) VALUES (?, false)", maxVersion.Int64); err != nil {
		return fmt.Errorf("inserting current version: %w", err)
	}

	return nil
}
