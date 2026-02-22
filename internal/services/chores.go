package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

var (
	ErrUserHasOverdueChores = errors.New("user has overdue chores")
	ErrChoreAlreadyComplete = errors.New("chore is already completed")
)

type ChoreService struct {
	choreRepo      repository.ChoreRepository
	assignmentRepo repository.ChoreAssignmentRepository
	userRepo       repository.UserRepository
}

func NewChoreService(
	choreRepo repository.ChoreRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	userRepo repository.UserRepository,
) *ChoreService {
	return &ChoreService{
		choreRepo:      choreRepo,
		assignmentRepo: assignmentRepo,
		userRepo:       userRepo,
	}
}

func (service *ChoreService) AssignNextUser(ctx context.Context, chore models.Chore) (models.Chore, error) {
	eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		return chore, fmt.Errorf("getting eligible assignees: %w", err)
	}

	var candidates []models.User
	if len(eligibleIDs) > 0 {
		eligibleSet := make(map[string]bool, len(eligibleIDs))
		for _, id := range eligibleIDs {
			eligibleSet[id] = true
		}
		allUsers, err := service.userRepo.FindAll(ctx)
		if err != nil {
			return chore, fmt.Errorf("finding users: %w", err)
		}
		for _, user := range allUsers {
			if eligibleSet[user.ID] {
				candidates = append(candidates, user)
			}
		}
	} else {
		candidates, err = service.userRepo.FindAll(ctx)
		if err != nil {
			return chore, fmt.Errorf("finding users: %w", err)
		}
	}

	if len(candidates) == 0 {
		return chore, errors.New("no users available for assignment")
	}

	nextIndex := (chore.LastAssignedIndex + 1) % len(candidates)

	var assignedUser models.User
	found := false
	for attempts := 0; attempts < len(candidates); attempts++ {
		candidateIndex := (nextIndex + attempts) % len(candidates)
		candidate := candidates[candidateIndex]

		overdueCount, err := service.choreRepo.CountByStatusAndUser(ctx, models.ChoreStatusOverdue, candidate.ID)
		if err != nil {
			return chore, fmt.Errorf("checking overdue chores: %w", err)
		}

		if overdueCount == 0 {
			assignedUser = candidate
			nextIndex = candidateIndex
			found = true
			break
		}
	}

	if !found {
		assignedUser = candidates[nextIndex%len(candidates)]
	}

	if chore.AssignedToUserID != nil {
		if err := service.assignmentRepo.MarkReassigned(ctx, chore.ID); err != nil {
			return chore, fmt.Errorf("marking old assignment: %w", err)
		}
	}

	chore.AssignedToUserID = &assignedUser.ID
	chore.LastAssignedIndex = nextIndex

	_, err = service.assignmentRepo.Create(ctx, models.ChoreAssignment{
		ChoreID: chore.ID,
		UserID:  assignedUser.ID,
		Status:  models.AssignmentStatusAssigned,
	})
	if err != nil {
		return chore, fmt.Errorf("creating assignment: %w", err)
	}

	if err := service.choreRepo.Update(ctx, chore); err != nil {
		return chore, fmt.Errorf("updating chore assignment: %w", err)
	}

	return chore, nil
}

func (service *ChoreService) CompleteChore(ctx context.Context, choreID string, userID string) error {
	chore, err := service.choreRepo.FindByID(ctx, choreID)
	if err != nil {
		return fmt.Errorf("finding chore: %w", err)
	}

	if chore.Status == models.ChoreStatusCompleted {
		return ErrChoreAlreadyComplete
	}

	now := time.Now()
	chore.Status = models.ChoreStatusCompleted
	chore.CompletedAt = &now
	chore.CompletedByUserID = &userID

	if err := service.choreRepo.Update(ctx, chore); err != nil {
		return fmt.Errorf("updating chore: %w", err)
	}

	if err := service.assignmentRepo.MarkCompleted(ctx, choreID, userID); err != nil {
		return fmt.Errorf("marking assignment completed: %w", err)
	}

	if chore.RecurrenceType != models.RecurrenceNone {
		if chore.RecurOnComplete {
			if err := service.createNextRecurrence(ctx, chore, now); err != nil {
				return fmt.Errorf("creating next recurrence: %w", err)
			}
		} else {
			if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(1, 0, 0)); err != nil {
				return fmt.Errorf("seeding future occurrences: %w", err)
			}
		}
	}

	return nil
}

func (service *ChoreService) createNextRecurrence(ctx context.Context, chore models.Chore, completedAt time.Time) error {
	nextDueDate, err := CalculateNextDueDate(chore, completedAt)
	if err != nil {
		return fmt.Errorf("calculating next due date: %w", err)
	}

	if nextDueDate == nil {
		return nil
	}

	if chore.SeriesID == nil {
		chore.SeriesID = &chore.ID
		if err := service.choreRepo.Update(ctx, chore); err != nil {
			return fmt.Errorf("setting series_id on legacy chore: %w", err)
		}
	}

	newChore := models.Chore{
		Name:              chore.Name,
		Description:       chore.Description,
		CreatedByUserID:   chore.CreatedByUserID,
		CategoryID:        chore.CategoryID,
		SeriesID:          chore.SeriesID,
		LastAssignedIndex: chore.LastAssignedIndex,
		DueDate:           nextDueDate,
		DueTime:           chore.DueTime,
		RecurrenceType:    chore.RecurrenceType,
		RecurrenceValue:   chore.RecurrenceValue,
		RecurOnComplete:   chore.RecurOnComplete,
		Status:            models.ChoreStatusPending,
	}

	createdChore, err := service.choreRepo.Create(ctx, newChore)
	if err != nil {
		return fmt.Errorf("creating next chore instance: %w", err)
	}

	eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		return fmt.Errorf("getting eligible assignees for recurrence: %w", err)
	}
	if len(eligibleIDs) > 0 {
		if err := service.choreRepo.SetEligibleAssignees(ctx, createdChore.ID, eligibleIDs); err != nil {
			return fmt.Errorf("copying eligible assignees: %w", err)
		}
	}

	_, err = service.AssignNextUser(ctx, createdChore)
	if err != nil {
		return fmt.Errorf("assigning next user: %w", err)
	}

	return nil
}

// SeedFutureOccurrences creates pending chore instances from the chore's series ahead to `until`.
// No-op for RecurOnComplete chores (can't predict completion dates) or chores without a DueDate.
// Idempotent: starts from the last existing future pending instance in the series.
func (service *ChoreService) SeedFutureOccurrences(ctx context.Context, chore models.Chore, until time.Time) error {
	if chore.RecurrenceType == models.RecurrenceNone || chore.RecurOnComplete {
		return nil
	}
	if chore.DueDate == nil {
		return nil
	}

	// Ensure series_id is set (handles legacy chores completed for the first time)
	if chore.SeriesID == nil {
		chore.SeriesID = &chore.ID
		if err := service.choreRepo.Update(ctx, chore); err != nil {
			return fmt.Errorf("setting series_id: %w", err)
		}
	}

	// Find the furthest-ahead pending instance — seed from there
	startChore := chore
	lastFuture, err := service.choreRepo.FindLastFuturePendingInSeries(ctx, *chore.SeriesID)
	if err != nil {
		return fmt.Errorf("finding last future pending: %w", err)
	}
	if lastFuture != nil {
		startChore = *lastFuture
	}

	config, err := parseConfig(chore.RecurrenceValue)
	if err != nil {
		return fmt.Errorf("parsing recurrence config: %w", err)
	}

	current := *startChore.DueDate
	currentChore := startChore
	now := time.Now()

	for i := 0; i < maxExpansionIterations; i++ {
		nextDate := advanceToNextOccurrence(current, chore.RecurrenceType, config)
		if !nextDate.Before(until) {
			break
		}
		current = nextDate

		if nextDate.Before(now) {
			continue // skip dates already in the past
		}

		newChore := models.Chore{
			Name:              chore.Name,
			Description:       chore.Description,
			CreatedByUserID:   chore.CreatedByUserID,
			CategoryID:        chore.CategoryID,
			SeriesID:          chore.SeriesID,
			LastAssignedIndex: currentChore.LastAssignedIndex,
			DueDate:           &nextDate,
			DueTime:           chore.DueTime,
			RecurrenceType:    chore.RecurrenceType,
			RecurrenceValue:   chore.RecurrenceValue,
			RecurOnComplete:   chore.RecurOnComplete,
			Status:            models.ChoreStatusPending,
		}

		created, err := service.choreRepo.Create(ctx, newChore)
		if err != nil {
			return fmt.Errorf("creating seeded chore instance: %w", err)
		}

		eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, chore.ID)
		if err == nil && len(eligibleIDs) > 0 {
			if err := service.choreRepo.SetEligibleAssignees(ctx, created.ID, eligibleIDs); err != nil {
				return fmt.Errorf("copying eligible assignees: %w", err)
			}
		}

		assigned, err := service.AssignNextUser(ctx, created)
		if err != nil {
			return fmt.Errorf("assigning seeded chore: %w", err)
		}
		currentChore = assigned
	}

	return nil
}

// SeedExistingRecurringChores seeds future instances for all pending recurring chores
// that have not yet been assigned a series_id. Call once at server startup.
func (service *ChoreService) SeedExistingRecurringChores(ctx context.Context, until time.Time) error {
	chores, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		RecurrenceTypes: []models.RecurrenceType{
			models.RecurrenceDaily,
			models.RecurrenceWeekly,
			models.RecurrenceMonthly,
			models.RecurrenceCustom,
			models.RecurrenceCalendar,
		},
	})
	if err != nil {
		return fmt.Errorf("finding chores to seed: %w", err)
	}

	for _, chore := range chores {
		if chore.SeriesID != nil {
			continue
		}
		if chore.RecurOnComplete || chore.DueDate == nil {
			continue
		}
		if err := service.SeedFutureOccurrences(ctx, chore, until); err != nil {
			slog.Error("seeding existing chore", "chore_id", chore.ID, "error", err)
		}
	}
	return nil
}

func (service *ChoreService) UpdateOverdueChores(ctx context.Context) error {
	overdueChores, err := service.choreRepo.FindOverdueChores(ctx)
	if err != nil {
		return fmt.Errorf("finding overdue chores: %w", err)
	}

	for _, chore := range overdueChores {
		chore.Status = models.ChoreStatusOverdue
		if err := service.choreRepo.Update(ctx, chore); err != nil {
			return fmt.Errorf("updating overdue chore %s: %w", chore.ID, err)
		}
	}

	return nil
}
