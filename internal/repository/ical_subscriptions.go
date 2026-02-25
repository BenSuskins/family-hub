package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type ICalSubscriptionRepository interface {
	FindAll(ctx context.Context) ([]models.ICalSubscription, error)
	FindByID(ctx context.Context, id string) (models.ICalSubscription, error)
	Create(ctx context.Context, sub models.ICalSubscription) error
	UpdateCache(ctx context.Context, id string, data string, fetchedAt time.Time) error
	UpdateColor(ctx context.Context, id string, color string) error
	Delete(ctx context.Context, id string) error
}

type SQLiteICalSubscriptionRepository struct {
	database *sql.DB
}

func NewICalSubscriptionRepository(database *sql.DB) *SQLiteICalSubscriptionRepository {
	return &SQLiteICalSubscriptionRepository{database: database}
}

func (repository *SQLiteICalSubscriptionRepository) FindAll(ctx context.Context) ([]models.ICalSubscription, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, url, color, cached_data, last_fetched_at, created_at
		FROM ical_subscriptions ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying ical subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.ICalSubscription
	for rows.Next() {
		var sub models.ICalSubscription
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.URL, &sub.Color, &sub.CachedData, &sub.LastFetchedAt, &sub.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning ical subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (repository *SQLiteICalSubscriptionRepository) FindByID(ctx context.Context, id string) (models.ICalSubscription, error) {
	var sub models.ICalSubscription
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, name, url, color, cached_data, last_fetched_at, created_at
		FROM ical_subscriptions WHERE id = ?`, id,
	).Scan(&sub.ID, &sub.Name, &sub.URL, &sub.Color, &sub.CachedData, &sub.LastFetchedAt, &sub.CreatedAt)
	if err != nil {
		return models.ICalSubscription{}, fmt.Errorf("finding ical subscription by id: %w", err)
	}
	return sub, nil
}

func (repository *SQLiteICalSubscriptionRepository) Create(ctx context.Context, sub models.ICalSubscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	if sub.Color == "" {
		sub.Color = "indigo"
	}
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO ical_subscriptions (id, name, url, color, created_at) VALUES (?, ?, ?, ?, ?)`,
		sub.ID, sub.Name, sub.URL, sub.Color, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("inserting ical subscription: %w", err)
	}
	return nil
}

func (repository *SQLiteICalSubscriptionRepository) UpdateColor(ctx context.Context, id string, color string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE ical_subscriptions SET color = ? WHERE id = ?`,
		color, id,
	)
	if err != nil {
		return fmt.Errorf("updating ical subscription color: %w", err)
	}
	return nil
}

func (repository *SQLiteICalSubscriptionRepository) UpdateCache(ctx context.Context, id string, data string, fetchedAt time.Time) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE ical_subscriptions SET cached_data = ?, last_fetched_at = ? WHERE id = ?`,
		data, fetchedAt, id,
	)
	if err != nil {
		return fmt.Errorf("updating ical subscription cache: %w", err)
	}
	return nil
}

func (repository *SQLiteICalSubscriptionRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx,
		`DELETE FROM ical_subscriptions WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("deleting ical subscription: %w", err)
	}
	return nil
}
