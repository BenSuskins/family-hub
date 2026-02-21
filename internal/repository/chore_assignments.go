package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type ChoreAssignmentRepository interface {
	Create(ctx context.Context, assignment models.ChoreAssignment) (models.ChoreAssignment, error)
	FindByChoreID(ctx context.Context, choreID string) ([]models.ChoreAssignment, error)
	MarkCompleted(ctx context.Context, choreID string, userID string) error
	MarkReassigned(ctx context.Context, choreID string) error
	CompletedCountByUser(ctx context.Context, userID string, since time.Time) (int, error)
	RecentCompleted(ctx context.Context, limit int) ([]models.ChoreAssignment, error)
	DeleteCompleted(ctx context.Context) error
}

type SQLiteChoreAssignmentRepository struct {
	database *sql.DB
}

func NewChoreAssignmentRepository(database *sql.DB) *SQLiteChoreAssignmentRepository {
	return &SQLiteChoreAssignmentRepository{database: database}
}

func (repository *SQLiteChoreAssignmentRepository) Create(ctx context.Context, assignment models.ChoreAssignment) (models.ChoreAssignment, error) {
	if assignment.ID == "" {
		assignment.ID = uuid.New().String()
	}
	assignment.AssignedAt = time.Now()
	if assignment.Status == "" {
		assignment.Status = models.AssignmentStatusAssigned
	}

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO chore_assignments (id, chore_id, user_id, assigned_at, completed_at, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		assignment.ID, assignment.ChoreID, assignment.UserID,
		assignment.AssignedAt, assignment.CompletedAt, assignment.Status,
	)
	if err != nil {
		return models.ChoreAssignment{}, fmt.Errorf("creating chore assignment: %w", err)
	}
	return assignment, nil
}

func (repository *SQLiteChoreAssignmentRepository) FindByChoreID(ctx context.Context, choreID string) ([]models.ChoreAssignment, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, chore_id, user_id, assigned_at, completed_at, status
		FROM chore_assignments WHERE chore_id = ? ORDER BY assigned_at DESC`, choreID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding assignments by chore: %w", err)
	}
	defer rows.Close()

	var assignments []models.ChoreAssignment
	for rows.Next() {
		var assignment models.ChoreAssignment
		if err := rows.Scan(&assignment.ID, &assignment.ChoreID, &assignment.UserID,
			&assignment.AssignedAt, &assignment.CompletedAt, &assignment.Status); err != nil {
			return nil, fmt.Errorf("scanning assignment: %w", err)
		}
		assignments = append(assignments, assignment)
	}
	return assignments, rows.Err()
}

func (repository *SQLiteChoreAssignmentRepository) MarkCompleted(ctx context.Context, choreID string, userID string) error {
	now := time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE chore_assignments SET status = ?, completed_at = ?
		WHERE chore_id = ? AND user_id = ? AND status = 'assigned'`,
		models.AssignmentStatusCompleted, now, choreID, userID,
	)
	if err != nil {
		return fmt.Errorf("marking assignment completed: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreAssignmentRepository) MarkReassigned(ctx context.Context, choreID string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE chore_assignments SET status = ?
		WHERE chore_id = ? AND status = 'assigned'`,
		models.AssignmentStatusReassigned, choreID,
	)
	if err != nil {
		return fmt.Errorf("marking assignment reassigned: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreAssignmentRepository) CompletedCountByUser(ctx context.Context, userID string, since time.Time) (int, error) {
	var count int
	err := repository.database.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM chore_assignments
		WHERE user_id = ? AND status = 'completed' AND completed_at >= ?`,
		userID, since,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting completed assignments: %w", err)
	}
	return count, nil
}

func (repository *SQLiteChoreAssignmentRepository) DeleteCompleted(ctx context.Context) error {
	_, err := repository.database.ExecContext(ctx,
		`DELETE FROM chore_assignments WHERE status = 'completed'`)
	if err != nil {
		return fmt.Errorf("deleting completed chore assignments: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreAssignmentRepository) RecentCompleted(ctx context.Context, limit int) ([]models.ChoreAssignment, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, chore_id, user_id, assigned_at, completed_at, status
		FROM chore_assignments WHERE status = 'completed'
		ORDER BY completed_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("finding recent completed: %w", err)
	}
	defer rows.Close()

	var assignments []models.ChoreAssignment
	for rows.Next() {
		var assignment models.ChoreAssignment
		if err := rows.Scan(&assignment.ID, &assignment.ChoreID, &assignment.UserID,
			&assignment.AssignedAt, &assignment.CompletedAt, &assignment.Status); err != nil {
			return nil, fmt.Errorf("scanning assignment: %w", err)
		}
		assignments = append(assignments, assignment)
	}
	return assignments, rows.Err()
}
