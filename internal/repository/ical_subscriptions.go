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
	Delete(ctx context.Context, id string) error
}

type SQLiteICalSubscriptionRepository struct {
	database *sql.DB
}

func NewICalSubscriptionRepository(database *sql.DB) *SQLiteICalSubscriptionRepository {
	return &SQLiteICalSubscriptionRepository{database: database}
}

func (r *SQLiteICalSubscriptionRepository) FindAll(ctx context.Context) ([]models.ICalSubscription, error) {
	rows, err := r.database.QueryContext(ctx,
		`SELECT id, name, url, cached_data, last_fetched_at, created_at
		FROM ical_subscriptions ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("querying ical subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.ICalSubscription
	for rows.Next() {
		var sub models.ICalSubscription
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.URL, &sub.CachedData, &sub.LastFetchedAt, &sub.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning ical subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (r *SQLiteICalSubscriptionRepository) FindByID(ctx context.Context, id string) (models.ICalSubscription, error) {
	var sub models.ICalSubscription
	err := r.database.QueryRowContext(ctx,
		`SELECT id, name, url, cached_data, last_fetched_at, created_at
		FROM ical_subscriptions WHERE id = ?`, id,
	).Scan(&sub.ID, &sub.Name, &sub.URL, &sub.CachedData, &sub.LastFetchedAt, &sub.CreatedAt)
	if err != nil {
		return models.ICalSubscription{}, fmt.Errorf("finding ical subscription by id: %w", err)
	}
	return sub, nil
}

func (r *SQLiteICalSubscriptionRepository) Create(ctx context.Context, sub models.ICalSubscription) error {
	if sub.ID == "" {
		sub.ID = uuid.New().String()
	}
	_, err := r.database.ExecContext(ctx,
		`INSERT INTO ical_subscriptions (id, name, url, created_at) VALUES (?, ?, ?, ?)`,
		sub.ID, sub.Name, sub.URL, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("inserting ical subscription: %w", err)
	}
	return nil
}

func (r *SQLiteICalSubscriptionRepository) UpdateCache(ctx context.Context, id string, data string, fetchedAt time.Time) error {
	_, err := r.database.ExecContext(ctx,
		`UPDATE ical_subscriptions SET cached_data = ?, last_fetched_at = ? WHERE id = ?`,
		data, fetchedAt, id,
	)
	if err != nil {
		return fmt.Errorf("updating ical subscription cache: %w", err)
	}
	return nil
}

func (r *SQLiteICalSubscriptionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.database.ExecContext(ctx,
		`DELETE FROM ical_subscriptions WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("deleting ical subscription: %w", err)
	}
	return nil
}
