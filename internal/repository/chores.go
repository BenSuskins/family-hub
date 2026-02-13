package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

const (
	OrderByDueDateAsc      = "due_date ASC NULLS LAST, name ASC"
	OrderByCompletedAtDesc = "completed_at DESC NULLS LAST, name ASC"
)

type ChoreFilter struct {
	Status          *models.ChoreStatus
	Statuses        []models.ChoreStatus
	RecurrenceTypes []models.RecurrenceType
	AssignedToUser  *string
	CategoryID      *string
	DueBefore       *time.Time
	DueAfter        *time.Time
	OrderBy         string
}

type ChoreRepository interface {
	FindByID(ctx context.Context, id string) (models.Chore, error)
	FindAll(ctx context.Context, filter ChoreFilter) ([]models.Chore, error)
	Create(ctx context.Context, chore models.Chore) (models.Chore, error)
	Update(ctx context.Context, chore models.Chore) error
	Delete(ctx context.Context, id string) error
	FindOverdueChores(ctx context.Context) ([]models.Chore, error)
	FindDueToday(ctx context.Context) ([]models.Chore, error)
	CountByStatusAndUser(ctx context.Context, status models.ChoreStatus, userID string) (int, error)
	SetEligibleAssignees(ctx context.Context, choreID string, userIDs []string) error
	GetEligibleAssignees(ctx context.Context, choreID string) ([]string, error)
}

type SQLiteChoreRepository struct {
	database *sql.DB
}

func NewChoreRepository(database *sql.DB) *SQLiteChoreRepository {
	return &SQLiteChoreRepository{database: database}
}

func (repository *SQLiteChoreRepository) FindByID(ctx context.Context, id string) (models.Chore, error) {
	var chore models.Chore
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, name, description, created_by_user_id, category_id,
			assigned_to_user_id, last_assigned_index,
			due_date, due_time,
			recurrence_type, recurrence_value, recur_on_complete,
			status, completed_at, completed_by_user_id,
			created_at, updated_at
		FROM chores WHERE id = ?`, id,
	).Scan(
		&chore.ID, &chore.Name, &chore.Description, &chore.CreatedByUserID, &chore.CategoryID,
		&chore.AssignedToUserID, &chore.LastAssignedIndex,
		&chore.DueDate, &chore.DueTime,
		&chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete,
		&chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
		&chore.CreatedAt, &chore.UpdatedAt,
	)
	if err != nil {
		return models.Chore{}, fmt.Errorf("finding chore by id: %w", err)
	}
	return chore, nil
}

func (repository *SQLiteChoreRepository) FindAll(ctx context.Context, filter ChoreFilter) ([]models.Chore, error) {
	query := `SELECT id, name, description, created_by_user_id, category_id,
		assigned_to_user_id, last_assigned_index,
		due_date, due_time,
		recurrence_type, recurrence_value, recur_on_complete,
		status, completed_at, completed_by_user_id,
		created_at, updated_at
	FROM chores WHERE 1=1`

	var args []interface{}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		query += " AND status IN (" + strings.Join(placeholders, ",") + ")"
	}
	if len(filter.RecurrenceTypes) > 0 {
		placeholders := make([]string, len(filter.RecurrenceTypes))
		for i, rt := range filter.RecurrenceTypes {
			placeholders[i] = "?"
			args = append(args, string(rt))
		}
		query += " AND recurrence_type IN (" + strings.Join(placeholders, ",") + ")"
	}
	if filter.AssignedToUser != nil {
		query += " AND assigned_to_user_id = ?"
		args = append(args, *filter.AssignedToUser)
	}
	if filter.CategoryID != nil {
		query += " AND category_id = ?"
		args = append(args, *filter.CategoryID)
	}
	if filter.DueBefore != nil {
		query += " AND due_date <= ?"
		args = append(args, *filter.DueBefore)
	}
	if filter.DueAfter != nil {
		query += " AND due_date >= ?"
		args = append(args, *filter.DueAfter)
	}

	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = OrderByDueDateAsc
	}
	query += " ORDER BY " + orderBy

	rows, err := repository.database.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finding chores: %w", err)
	}
	defer rows.Close()

	return scanChores(rows)
}

func (repository *SQLiteChoreRepository) Create(ctx context.Context, chore models.Chore) (models.Chore, error) {
	if chore.ID == "" {
		chore.ID = uuid.New().String()
	}
	now := time.Now()
	chore.CreatedAt = now
	chore.UpdatedAt = now
	if chore.Status == "" {
		chore.Status = models.ChoreStatusPending
	}
	if chore.RecurrenceType == "" {
		chore.RecurrenceType = models.RecurrenceNone
	}

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO chores (id, name, description, created_by_user_id, category_id,
			assigned_to_user_id, last_assigned_index,
			due_date, due_time,
			recurrence_type, recurrence_value, recur_on_complete,
			status, completed_at, completed_by_user_id,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chore.ID, chore.Name, chore.Description, chore.CreatedByUserID, chore.CategoryID,
		chore.AssignedToUserID, chore.LastAssignedIndex,
		chore.DueDate, chore.DueTime,
		chore.RecurrenceType, chore.RecurrenceValue, chore.RecurOnComplete,
		chore.Status, chore.CompletedAt, chore.CompletedByUserID,
		chore.CreatedAt, chore.UpdatedAt,
	)
	if err != nil {
		return models.Chore{}, fmt.Errorf("creating chore: %w", err)
	}
	return chore, nil
}

func (repository *SQLiteChoreRepository) Update(ctx context.Context, chore models.Chore) error {
	chore.UpdatedAt = time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE chores SET name = ?, description = ?, category_id = ?,
			assigned_to_user_id = ?, last_assigned_index = ?,
			due_date = ?, due_time = ?,
			recurrence_type = ?, recurrence_value = ?, recur_on_complete = ?,
			status = ?, completed_at = ?, completed_by_user_id = ?,
			updated_at = ?
		WHERE id = ?`,
		chore.Name, chore.Description, chore.CategoryID,
		chore.AssignedToUserID, chore.LastAssignedIndex,
		chore.DueDate, chore.DueTime,
		chore.RecurrenceType, chore.RecurrenceValue, chore.RecurOnComplete,
		chore.Status, chore.CompletedAt, chore.CompletedByUserID,
		chore.UpdatedAt, chore.ID,
	)
	if err != nil {
		return fmt.Errorf("updating chore: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM chores WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting chore: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreRepository) FindOverdueChores(ctx context.Context) ([]models.Chore, error) {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)

	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, description, created_by_user_id, category_id,
			assigned_to_user_id, last_assigned_index,
			due_date, due_time,
			recurrence_type, recurrence_value, recur_on_complete,
			status, completed_at, completed_by_user_id,
			created_at, updated_at
		FROM chores
		WHERE status IN ('pending', 'overdue') AND due_date IS NOT NULL AND due_date < ?
		ORDER BY due_date ASC`,
		endOfToday,
	)
	if err != nil {
		return nil, fmt.Errorf("finding overdue chores: %w", err)
	}
	defer rows.Close()

	candidates, err := scanChores(rows)
	if err != nil {
		return nil, err
	}

	var overdue []models.Chore
	for _, chore := range candidates {
		if IsOverdue(chore, now) {
			overdue = append(overdue, chore)
		}
	}
	return overdue, nil
}

func (repository *SQLiteChoreRepository) FindDueToday(ctx context.Context) ([]models.Chore, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, description, created_by_user_id, category_id,
			assigned_to_user_id, last_assigned_index,
			due_date, due_time,
			recurrence_type, recurrence_value, recur_on_complete,
			status, completed_at, completed_by_user_id,
			created_at, updated_at
		FROM chores
		WHERE due_date >= ? AND due_date < ? AND status = 'pending'
		ORDER BY due_date ASC`,
		today, tomorrow,
	)
	if err != nil {
		return nil, fmt.Errorf("finding chores due today: %w", err)
	}
	defer rows.Close()

	return scanChores(rows)
}

func (repository *SQLiteChoreRepository) CountByStatusAndUser(ctx context.Context, status models.ChoreStatus, userID string) (int, error) {
	var count int
	err := repository.database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM chores WHERE status = ? AND assigned_to_user_id = ?",
		status, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting chores: %w", err)
	}
	return count, nil
}

func (repository *SQLiteChoreRepository) SetEligibleAssignees(ctx context.Context, choreID string, userIDs []string) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx, "DELETE FROM chore_eligible_assignees WHERE chore_id = ?", choreID); err != nil {
		return fmt.Errorf("clearing eligible assignees: %w", err)
	}

	for _, userID := range userIDs {
		if _, err := transaction.ExecContext(ctx,
			"INSERT INTO chore_eligible_assignees (chore_id, user_id) VALUES (?, ?)",
			choreID, userID,
		); err != nil {
			return fmt.Errorf("inserting eligible assignee: %w", err)
		}
	}

	return transaction.Commit()
}

func (repository *SQLiteChoreRepository) GetEligibleAssignees(ctx context.Context, choreID string) ([]string, error) {
	rows, err := repository.database.QueryContext(ctx,
		"SELECT user_id FROM chore_eligible_assignees WHERE chore_id = ? ORDER BY user_id",
		choreID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding eligible assignees: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scanning eligible assignee: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

func IsOverdue(chore models.Chore, now time.Time) bool {
	if chore.DueDate == nil {
		return false
	}

	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(chore.DueDate.Year(), chore.DueDate.Month(), chore.DueDate.Day(), 0, 0, 0, 0, now.Location())

	if dueDay.Before(startOfToday) {
		return true
	}

	if dueDay.Equal(startOfToday) && chore.DueTime != nil {
		parsed, err := time.Parse("15:04", *chore.DueTime)
		if err != nil {
			return false
		}
		dueAt := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
		return now.After(dueAt)
	}

	return false
}

func scanChores(rows *sql.Rows) ([]models.Chore, error) {
	var chores []models.Chore
	for rows.Next() {
		var chore models.Chore
		if err := rows.Scan(
			&chore.ID, &chore.Name, &chore.Description, &chore.CreatedByUserID, &chore.CategoryID,
			&chore.AssignedToUserID, &chore.LastAssignedIndex,
			&chore.DueDate, &chore.DueTime,
			&chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete,
			&chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
			&chore.CreatedAt, &chore.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning chore: %w", err)
		}
		chores = append(chores, chore)
	}
	return chores, rows.Err()
}
