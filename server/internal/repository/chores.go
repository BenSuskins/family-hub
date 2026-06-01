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
	Status             *models.ChoreStatus
	Statuses           []models.ChoreStatus
	RecurrenceTypes    []models.RecurrenceType
	AssignedToUser     *string
	CategoryID         *string
	DueBefore          *time.Time
	DueAfter           *time.Time
	OrderBy            string
	Limit              int
	OnlyNextPerSeries  bool
}

type ChoreRepository interface {
	FindByID(ctx context.Context, id string) (models.Chore, error)
	FindAll(ctx context.Context, filter ChoreFilter) ([]models.Chore, error)
	Create(ctx context.Context, chore models.Chore) (models.Chore, error)
	Update(ctx context.Context, chore models.Chore) error
	Delete(ctx context.Context, id string) error
	FindOverdueChores(ctx context.Context) ([]models.Chore, error)
	MarkOverdue(ctx context.Context, choreID string) error
	FindDueToday(ctx context.Context) ([]models.Chore, error)
	CountByStatusAndUser(ctx context.Context, status models.ChoreStatus, userID string) (int, error)
	SetEligibleAssignees(ctx context.Context, choreID string, userIDs []string) error
	GetEligibleAssignees(ctx context.Context, choreID string) ([]string, error)
	DeleteFuturePendingBySeries(ctx context.Context, seriesID string) error
	FindLastFuturePendingInSeries(ctx context.Context, seriesID string) (*models.Chore, error)
	CountBySeries(ctx context.Context, seriesID string) (int, error)
	DeleteCompletedByName(ctx context.Context, name string) error
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
		fmt.Sprintf("SELECT %s %s WHERE c.id = ?", choreSelectColumns, choreJoin), id,
	).Scan(
		&chore.ID, &chore.Name, &chore.Description, &chore.CreatedByUserID, &chore.CategoryID,
		&chore.AssignedToUserID, &chore.LastAssignedIndex,
		&chore.DueDate, &chore.DueTime,
		&chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete, &chore.SeriesID,
		&chore.RecurrenceUntil, &chore.RecurrenceCount,
		&chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
		&chore.CreatedAt, &chore.UpdatedAt,
	)
	if err != nil {
		return models.Chore{}, fmt.Errorf("finding chore by id: %w", err)
	}
	return chore, nil
}

// As of migration 016 the recurrence rule lives in chore_series, not on chores.
// Reads LEFT JOIN the series and project the rule fields back into the same
// column order the model scans, so the rest of the code is unaffected by the
// normalization. choreColumnNames is the bare list for the OnlyNextPerSeries
// subquery wrapper.
const choreSelectColumns = `c.id AS id, c.name AS name, c.description AS description,
		c.created_by_user_id AS created_by_user_id, c.category_id AS category_id,
		c.assigned_to_user_id AS assigned_to_user_id, c.last_assigned_index AS last_assigned_index,
		c.due_date AS due_date, c.due_time AS due_time,
		COALESCE(cs.recurrence_type, 'none') AS recurrence_type,
		COALESCE(cs.recurrence_value, '') AS recurrence_value,
		COALESCE(cs.recur_on_complete, 0) AS recur_on_complete,
		c.series_id AS series_id,
		cs.recurrence_until AS recurrence_until, cs.recurrence_count AS recurrence_count,
		c.status AS status, c.completed_at AS completed_at, c.completed_by_user_id AS completed_by_user_id,
		c.created_at AS created_at, c.updated_at AS updated_at`

const choreColumnNames = `id, name, description, created_by_user_id, category_id,
		assigned_to_user_id, last_assigned_index, due_date, due_time,
		recurrence_type, recurrence_value, recur_on_complete, series_id,
		recurrence_until, recurrence_count, status, completed_at, completed_by_user_id,
		created_at, updated_at`

const choreJoin = `FROM chores c LEFT JOIN chore_series cs ON cs.id = c.series_id`

func (repository *SQLiteChoreRepository) FindAll(ctx context.Context, filter ChoreFilter) ([]models.Chore, error) {
	where := "WHERE 1=1"
	var args []interface{}

	if filter.Status != nil {
		where += " AND c.status = ?"
		args = append(args, *filter.Status)
	}
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		where += " AND c.status IN (" + strings.Join(placeholders, ",") + ")"
	}
	if len(filter.RecurrenceTypes) > 0 {
		placeholders := make([]string, len(filter.RecurrenceTypes))
		for i, rt := range filter.RecurrenceTypes {
			placeholders[i] = "?"
			args = append(args, string(rt))
		}
		where += " AND COALESCE(cs.recurrence_type, 'none') IN (" + strings.Join(placeholders, ",") + ")"
	}
	if filter.AssignedToUser != nil {
		where += " AND c.assigned_to_user_id = ?"
		args = append(args, *filter.AssignedToUser)
	}
	if filter.CategoryID != nil {
		where += " AND c.category_id = ?"
		args = append(args, *filter.CategoryID)
	}
	if filter.DueBefore != nil {
		where += " AND c.due_date <= ?"
		args = append(args, *filter.DueBefore)
	}
	if filter.DueAfter != nil {
		where += " AND c.due_date >= ?"
		args = append(args, *filter.DueAfter)
	}

	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = OrderByDueDateAsc
	}

	var query string
	if filter.OnlyNextPerSeries {
		query = fmt.Sprintf(
			`SELECT %s FROM (
				SELECT %s,
					ROW_NUMBER() OVER (PARTITION BY COALESCE(c.series_id, c.id) ORDER BY c.due_date ASC NULLS LAST) AS _rn
				%s %s
			) WHERE _rn = 1
			ORDER BY %s`,
			choreColumnNames, choreSelectColumns, choreJoin, where, orderBy,
		)
	} else {
		query = fmt.Sprintf("SELECT %s %s %s ORDER BY %s", choreSelectColumns, choreJoin, where, orderBy)
	}

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

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

	// The recurrence rule lives in chore_series (migration 016); only occurrence
	// state is stored here. series_id has a foreign key, so the series row must
	// already exist when it is non-null.
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO chores (id, name, description, created_by_user_id, category_id,
			assigned_to_user_id, last_assigned_index,
			due_date, due_time, series_id,
			status, completed_at, completed_by_user_id,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		chore.ID, chore.Name, chore.Description, chore.CreatedByUserID, chore.CategoryID,
		chore.AssignedToUserID, chore.LastAssignedIndex,
		chore.DueDate, chore.DueTime, chore.SeriesID,
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
			due_date = ?, due_time = ?, series_id = ?,
			status = ?, completed_at = ?, completed_by_user_id = ?,
			updated_at = ?
		WHERE id = ?`,
		chore.Name, chore.Description, chore.CategoryID,
		chore.AssignedToUserID, chore.LastAssignedIndex,
		chore.DueDate, chore.DueTime, chore.SeriesID,
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
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Add(24 * time.Hour)

	rows, err := repository.database.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s %s
		WHERE c.status IN ('pending', 'overdue') AND c.due_date IS NOT NULL AND c.due_date < ?
		ORDER BY c.due_date ASC`, choreSelectColumns, choreJoin),
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

// MarkOverdue flips a chore to overdue only if it is still pending. The
// status guard makes the call idempotent so a chore already marked overdue is
// left untouched (no needless updated_at churn).
func (repository *SQLiteChoreRepository) MarkOverdue(ctx context.Context, choreID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE chores SET status = 'overdue', updated_at = ? WHERE id = ? AND status = 'pending'",
		time.Now(), choreID,
	)
	if err != nil {
		return fmt.Errorf("marking chore overdue: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreRepository) FindDueToday(ctx context.Context) ([]models.Chore, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	rows, err := repository.database.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s %s
		WHERE c.due_date >= ? AND c.due_date < ? AND c.status = 'pending'
		ORDER BY c.due_date ASC`, choreSelectColumns, choreJoin),
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

func (repository *SQLiteChoreRepository) DeleteFuturePendingBySeries(ctx context.Context, seriesID string) error {
	_, err := repository.database.ExecContext(ctx,
		`DELETE FROM chores WHERE series_id = ? AND status = 'pending' AND due_date > CURRENT_TIMESTAMP`,
		seriesID,
	)
	if err != nil {
		return fmt.Errorf("deleting future pending by series: %w", err)
	}
	return nil
}

func (repository *SQLiteChoreRepository) FindLastFuturePendingInSeries(ctx context.Context, seriesID string) (*models.Chore, error) {
	rows, err := repository.database.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s %s
		WHERE c.series_id = ? AND c.status = 'pending' AND c.due_date > CURRENT_TIMESTAMP
		ORDER BY c.due_date DESC
		LIMIT 1`, choreSelectColumns, choreJoin),
		seriesID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding last future pending in series: %w", err)
	}
	defer rows.Close()

	chores, err := scanChores(rows)
	if err != nil {
		return nil, err
	}
	if len(chores) == 0 {
		return nil, nil
	}
	return &chores[0], nil
}

// CountBySeries returns the total number of chore rows (any status) belonging to
// the series. Used to enforce a recurrence occurrence cap.
func (repository *SQLiteChoreRepository) CountBySeries(ctx context.Context, seriesID string) (int, error) {
	var count int
	err := repository.database.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM chores WHERE series_id = ?", seriesID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting chores by series: %w", err)
	}
	return count, nil
}

func (repository *SQLiteChoreRepository) DeleteCompletedByName(ctx context.Context, name string) error {
	_, err := repository.database.ExecContext(ctx,
		`DELETE FROM chores WHERE name = ? AND status = 'completed'`,
		name,
	)
	if err != nil {
		return fmt.Errorf("deleting completed chores by name: %w", err)
	}
	return nil
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
			&chore.RecurrenceType, &chore.RecurrenceValue, &chore.RecurOnComplete, &chore.SeriesID,
			&chore.RecurrenceUntil, &chore.RecurrenceCount,
			&chore.Status, &chore.CompletedAt, &chore.CompletedByUserID,
			&chore.CreatedAt, &chore.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning chore: %w", err)
		}
		chores = append(chores, chore)
	}
	return chores, rows.Err()
}
