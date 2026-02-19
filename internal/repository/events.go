package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type EventFilter struct {
	StartAfter  *time.Time
	StartBefore *time.Time
	CategoryID  *string
}

type EventRepository interface {
	FindByID(ctx context.Context, id string) (models.Event, error)
	FindAll(ctx context.Context, filter EventFilter) ([]models.Event, error)
	Create(ctx context.Context, event models.Event) (models.Event, error)
	Update(ctx context.Context, event models.Event) error
	Delete(ctx context.Context, id string) error
}

type SQLiteEventRepository struct {
	database *sql.DB
}

func NewEventRepository(database *sql.DB) *SQLiteEventRepository {
	return &SQLiteEventRepository{database: database}
}

func (repository *SQLiteEventRepository) FindByID(ctx context.Context, id string) (models.Event, error) {
	var event models.Event
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, title, description, location, start_time, end_time, all_day,
			category_id, created_by_user_id, created_at, updated_at
		FROM events WHERE id = ?`, id,
	).Scan(
		&event.ID, &event.Title, &event.Description, &event.Location,
		&event.StartTime, &event.EndTime, &event.AllDay,
		&event.CategoryID, &event.CreatedByUserID, &event.CreatedAt, &event.UpdatedAt,
	)
	if err != nil {
		return models.Event{}, fmt.Errorf("finding event by id: %w", err)
	}
	return event, nil
}

func (repository *SQLiteEventRepository) FindAll(ctx context.Context, filter EventFilter) ([]models.Event, error) {
	query := `SELECT id, title, description, location, start_time, end_time, all_day,
		category_id, created_by_user_id, created_at, updated_at
	FROM events WHERE 1=1`

	var args []interface{}

	if filter.StartAfter != nil {
		query += " AND start_time >= ?"
		args = append(args, *filter.StartAfter)
	}
	if filter.StartBefore != nil {
		query += " AND start_time <= ?"
		args = append(args, *filter.StartBefore)
	}
	if filter.CategoryID != nil {
		query += " AND category_id = ?"
		args = append(args, *filter.CategoryID)
	}

	query += " ORDER BY start_time ASC"

	rows, err := repository.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var event models.Event
		if err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.Location,
			&event.StartTime, &event.EndTime, &event.AllDay,
			&event.CategoryID, &event.CreatedByUserID, &event.CreatedAt, &event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (repository *SQLiteEventRepository) Create(ctx context.Context, event models.Event) (models.Event, error) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO events (id, title, description, location, start_time, end_time, all_day,
			category_id, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.AllDay,
		event.CategoryID, event.CreatedByUserID, event.CreatedAt, event.UpdatedAt,
	)
	if err != nil {
		return models.Event{}, fmt.Errorf("creating event: %w", err)
	}
	return event, nil
}

func (repository *SQLiteEventRepository) Update(ctx context.Context, event models.Event) error {
	event.UpdatedAt = time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE events SET title = ?, description = ?, location = ?,
			start_time = ?, end_time = ?, all_day = ?, category_id = ?, updated_at = ?
		WHERE id = ?`,
		event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.AllDay,
		event.CategoryID, event.UpdatedAt, event.ID,
	)
	if err != nil {
		return fmt.Errorf("updating event: %w", err)
	}
	return nil
}

func (repository *SQLiteEventRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM events WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting event: %w", err)
	}
	return nil
}
