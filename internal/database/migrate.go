package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(database *sql.DB) error {
	if _, err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	var upMigrations []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			upMigrations = append(upMigrations, entry.Name())
		}
	}
	sort.Strings(upMigrations)

	for _, filename := range upMigrations {
		version := extractVersion(filename)

		var exists int
		err := database.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("checking migration %d: %w", version, err)
		}
		if exists > 0 {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", filename, err)
		}

		transaction, err := database.Begin()
		if err != nil {
			return fmt.Errorf("beginning transaction for migration %d: %w", version, err)
		}

		if _, err := transaction.Exec(string(content)); err != nil {
			transaction.Rollback()
			return fmt.Errorf("executing migration %s: %w", filename, err)
		}

		if _, err := transaction.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			transaction.Rollback()
			return fmt.Errorf("recording migration %d: %w", version, err)
		}

		if err := transaction.Commit(); err != nil {
			return fmt.Errorf("committing migration %d: %w", version, err)
		}

		slog.Info("applied migration", "version", version, "file", filename)
	}

	return nil
}

func extractVersion(filename string) int {
	var version int
	fmt.Sscanf(filename, "%d_", &version)
	return version
}
