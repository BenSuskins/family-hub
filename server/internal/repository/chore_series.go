package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

const choreSeriesColumns = `id, name, description, created_by_user_id, category_id,
		due_time,
		recurrence_type, recurrence_value, recur_on_complete, recurrence_until, recurrence_count,
		rotation_cursor_user_id, deleted_at,
		created_at, updated_at`

type ChoreSeriesRepository interface {
	// FindByID returns the series, or (nil, nil) when no such series exists, so
	// callers can fall back to per-occurrence fields during the migration window.
	FindByID(ctx context.Context, id string) (*models.ChoreSeries, error)
	Create(ctx context.Context, series models.ChoreSeries) (models.ChoreSeries, error)
	Update(ctx context.Context, series models.ChoreSeries) error
	SetRotationCursor(ctx context.Context, seriesID string, userID string) error
	MarkDeleted(ctx context.Context, seriesID string) error
	SetEligibleAssignees(ctx context.Context, seriesID string, userIDs []string) error
	GetEligibleAssignees(ctx context.Context, seriesID string) ([]string, error)
}

type SQLiteChoreSeriesRepository struct {
	database *sql.DB
}

func NewChoreSeriesRepository(database *sql.DB) *SQLiteChoreSeriesRepository {
	return &SQLiteChoreSeriesRepository{database: database}
}

func (repository *SQLiteChoreSeriesRepository) FindByID(ctx context.Context, id string) (*models.ChoreSeries, error) {
	var series models.ChoreSeries
	err := repository.database.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM chore_series WHERE id = ?", choreSeriesColumns), id,
	).Scan(
		&series.ID, &series.Name, &series.Description, &series.CreatedByUserID, &series.CategoryID,
		&series.DueTime,
		&series.RecurrenceType, &series.RecurrenceValue, &series.RecurOnComplete, &series.RecurrenceUntil, &series.RecurrenceCount,
		&series.RotationCursorUserID, &series.DeletedAt,
		&series.CreatedAt, &series.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("finding chore series by id: %w", err)
	}

	assignees, err := repository.GetEligibleAssignees(ctx, series.ID)
	if err != nil {
		return nil, err
	}
	series.EligibleAssignees = assignees
	return &series, nil
}

func (repository *SQLiteChoreSeriesRepository) Create(ctx context.Context, series models.ChoreSeries) (models.ChoreSeries, error) {
	if series.ID == "" {
		series.ID = uuid.New().String()
	}
	now := time.Now()
	series.CreatedAt = now
	series.UpdatedAt = now
	if series.RecurrenceType == "" {
		series.RecurrenceType = models.RecurrenceNone
	}

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO chore_series (id, name, description, created_by_user_id, category_id,
			due_time,
			recurrence_type, recurrence_value, recur_on_complete, recurrence_until, recurrence_count,
			rotation_cursor_user_id, deleted_at,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		series.ID, series.Name, series.Description, series.CreatedByUserID, series.CategoryID,
		series.DueTime,
		series.RecurrenceType, series.RecurrenceValue, series.RecurOnComplete, series.RecurrenceUntil, series.RecurrenceCount,
		series.RotationCursorUserID, series.DeletedAt,
		series.CreatedAt, series.UpdatedAt,
	)
	if err != nil {
		return models.ChoreSeries{}, fmt.Errorf("creating chore series: %w", err)
	}
	return series, nil
}

func (repository *SQLiteChoreSeriesRepository) Update(ctx context.Context, series models.ChoreSeries) error {
	series.UpdatedAt = time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE chore_series SET name = ?, description = ?, category_id = ?,
			due_time = ?,
			recurrence_type = ?, recurrence_value = ?, recur_on_complete = ?, recurrence_until = ?, recurrence_count = ?,
			rotation_cursor_user_id = ?, deleted_at = ?,
			updated_at = ?
		WHERE id = ?`,
		series.Name, series.Description, series.CategoryID,
		series.DueTime,
		series.RecurrenceType, series.RecurrenceValue, series.RecurOnComplete, series.RecurrenceUntil, series.RecurrenceCount,
		series.RotationCursorUserID, series.DeletedAt,
		series.UpdatedAt, series.ID,
	)
	if err != nil {
		return fmt.Errorf("updating chore series: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreSeriesRepository) SetRotationCursor(ctx context.Context, seriesID string, userID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE chore_series SET rotation_cursor_user_id = ?, updated_at = ? WHERE id = ?",
		userID, time.Now(), seriesID,
	)
	if err != nil {
		return fmt.Errorf("setting rotation cursor: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreSeriesRepository) MarkDeleted(ctx context.Context, seriesID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE chore_series SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL",
		time.Now(), time.Now(), seriesID,
	)
	if err != nil {
		return fmt.Errorf("marking series deleted: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreSeriesRepository) SetEligibleAssignees(ctx context.Context, seriesID string, userIDs []string) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx, "DELETE FROM chore_series_eligible_assignees WHERE series_id = ?", seriesID); err != nil {
		return fmt.Errorf("clearing series eligible assignees: %w", err)
	}

	for _, userID := range userIDs {
		if _, err := transaction.ExecContext(ctx,
			"INSERT INTO chore_series_eligible_assignees (series_id, user_id) VALUES (?, ?)",
			seriesID, userID,
		); err != nil {
			return fmt.Errorf("inserting series eligible assignee: %w", err)
		}
	}

	return transaction.Commit()
}

func (repository *SQLiteChoreSeriesRepository) GetEligibleAssignees(ctx context.Context, seriesID string) ([]string, error) {
	rows, err := repository.database.QueryContext(ctx,
		"SELECT user_id FROM chore_series_eligible_assignees WHERE series_id = ? ORDER BY user_id",
		seriesID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding series eligible assignees: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scanning series eligible assignee: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}
