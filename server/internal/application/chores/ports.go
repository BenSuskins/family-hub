package chores

import (
	"context"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var ErrChoreNotFound = errors.New("chore not found")

type ChoreOrder string

const (
	OrderDueDate   ChoreOrder = "due_date"
	OrderCompleted ChoreOrder = "completed"
	OrderHistory   ChoreOrder = "history"
)

type ChoreFilter struct {
	Status            *models.ChoreStatus
	Statuses          []models.ChoreStatus
	RecurrenceTypes   []models.RecurrenceType
	AssignedToUser    *string
	CategoryID        *string
	DueBefore         *time.Time
	DueAfter          *time.Time
	OrderBy           ChoreOrder
	Limit             int
	OnlyNextPerSeries bool
	IncludeHistory    bool
	HistoryOnly       bool
	SeriesID          *string
	AfterID           string
}

type ChoreRepository interface {
	Transact(context.Context, func(ChoreRepository, ChoreAssignmentRepository, UserRepository, ChoreSeriesRepository) error) error

	FindByID(ctx context.Context, id string) (models.Chore, error)
	FindAll(ctx context.Context, filter ChoreFilter) ([]models.Chore, error)
	Create(ctx context.Context, chore models.Chore) (models.Chore, error)
	Update(ctx context.Context, chore models.Chore) error
	FindOverdueChores(ctx context.Context) ([]models.Chore, error)
	MarkOverdue(ctx context.Context, choreID string) error
	SetEligibleAssignees(ctx context.Context, choreID string, userIDs []string) error
	GetEligibleAssignees(ctx context.Context, choreID string) ([]string, error)
	CountBySeries(ctx context.Context, seriesID string) (int, error)
}

type ChoreAssignmentRepository interface {
	Create(context.Context, models.ChoreAssignment) (models.ChoreAssignment, error)
	MarkReassigned(context.Context, string) error
}

type UserRepository interface {
	FindAll(context.Context) ([]models.User, error)
}

type ChoreSeriesRepository interface {
	FindAll(context.Context) ([]models.ChoreSeries, error)
	// FindByID returns the series, or (nil, nil) when no such series exists, so
	// callers can fall back to per-occurrence fields during the migration window.
	FindByID(ctx context.Context, id string) (*models.ChoreSeries, error)
	Create(ctx context.Context, series models.ChoreSeries) (models.ChoreSeries, error)
	Update(ctx context.Context, series models.ChoreSeries) error
	SetRotationCursor(ctx context.Context, seriesID string, userID string) error
	MarkDeleted(ctx context.Context, seriesID string) error
	SetEligibleAssignees(ctx context.Context, seriesID string, userIDs []string) error
}
