package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
}

type SQLiteSettingsRepository struct {
	database *sql.DB
}

func NewSettingsRepository(database *sql.DB) *SQLiteSettingsRepository {
	return &SQLiteSettingsRepository{database: database}
}

func (repository *SQLiteSettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := repository.database.QueryRowContext(ctx,
		"SELECT value FROM settings WHERE key = ?", key,
	).Scan(&value)
	if err != nil {
		return "", fmt.Errorf("getting setting %s: %w", key, err)
	}
	return value, nil
}

func (repository *SQLiteSettingsRepository) Set(ctx context.Context, key string, value string) error {
	_, err := repository.database.ExecContext(ctx,
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?",
		key, value, value,
	)
	if err != nil {
		return fmt.Errorf("setting %s: %w", key, err)
	}
	return nil
}
