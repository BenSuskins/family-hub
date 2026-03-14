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
	candidates, err := service.findCandidates(ctx, chore.ID)
	if err != nil {
		return chore, err
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

	if chore.RecurrenceType == models.RecurrenceNone {
		return nil
	}

	if chore.RecurOnComplete {
		return service.createNextRecurrence(ctx, chore, now)
	}
	return service.SeedFutureOccurrences(ctx, chore, now.AddDate(1, 0, 0))

}

func (service *ChoreService) findCandidates(ctx context.Context, choreID string) ([]models.User, error) {
	eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, choreID)
	if err != nil {
		return nil, fmt.Errorf("getting eligible assignees: %w", err)
	}

	allUsers, err := service.userRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	if len(eligibleIDs) == 0 {
		return allUsers, nil
	}

	eligibleSet := make(map[string]bool, len(eligibleIDs))
	for _, id := range eligibleIDs {
		eligibleSet[id] = true
	}

	var candidates []models.User
	for _, user := range allUsers {
		if eligibleSet[user.ID] {
			candidates = append(candidates, user)
		}
	}
	return candidates, nil
}

func (service *ChoreService) copyEligibleAssignees(ctx context.Context, sourceChoreID, targetChoreID string) error {
	eligibleIDs, err := service.choreRepo.GetEligibleAssignees(ctx, sourceChoreID)
	if err != nil {
		return fmt.Errorf("getting eligible assignees for recurrence: %w", err)
	}
	if len(eligibleIDs) > 0 {
		if err := service.choreRepo.SetEligibleAssignees(ctx, targetChoreID, eligibleIDs); err != nil {
			return fmt.Errorf("copying eligible assignees: %w", err)
		}
	}
	return nil
}

func (service *ChoreService) ensureSeriesID(ctx context.Context, chore *models.Chore) error {
	if chore.SeriesID != nil {
		return nil
	}
	chore.SeriesID = &chore.ID
	return service.choreRepo.Update(ctx, *chore)
}

func newChoreFromTemplate(template models.Chore, dueDate *time.Time, lastAssignedIndex int) models.Chore {
	return models.Chore{
		Name:              template.Name,
		Description:       template.Description,
		CreatedByUserID:   template.CreatedByUserID,
		CategoryID:        template.CategoryID,
		SeriesID:          template.SeriesID,
		LastAssignedIndex: lastAssignedIndex,
		DueDate:           dueDate,
		DueTime:           template.DueTime,
		RecurrenceType:    template.RecurrenceType,
		RecurrenceValue:   template.RecurrenceValue,
		RecurOnComplete:   template.RecurOnComplete,
		Status:            models.ChoreStatusPending,
	}
}

func (service *ChoreService) createNextRecurrence(ctx context.Context, chore models.Chore, completedAt time.Time) error {
	nextDueDate, err := CalculateNextDueDate(chore, completedAt)
	if err != nil {
		return fmt.Errorf("calculating next due date: %w", err)
	}
	if nextDueDate == nil {
		return nil
	}

	if err := service.ensureSeriesID(ctx, &chore); err != nil {
		return fmt.Errorf("setting series_id on legacy chore: %w", err)
	}

	createdChore, err := service.choreRepo.Create(ctx, newChoreFromTemplate(chore, nextDueDate, chore.LastAssignedIndex))
	if err != nil {
		return fmt.Errorf("creating next chore instance: %w", err)
	}

	if err := service.copyEligibleAssignees(ctx, chore.ID, createdChore.ID); err != nil {
		return err
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
	if chore.RecurrenceType == models.RecurrenceNone || chore.RecurOnComplete || chore.DueDate == nil {
		return nil
	}

	if err := service.ensureSeriesID(ctx, &chore); err != nil {
		return fmt.Errorf("setting series_id: %w", err)
	}

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
			continue
		}

		created, err := service.choreRepo.Create(ctx, newChoreFromTemplate(chore, &nextDate, currentChore.LastAssignedIndex))
		if err != nil {
			return fmt.Errorf("creating seeded chore instance: %w", err)
		}

		if err := service.copyEligibleAssignees(ctx, chore.ID, created.ID); err != nil {
			return err
		}

		assigned, err := service.AssignNextUser(ctx, created)
		if err != nil {
			return fmt.Errorf("assigning seeded chore: %w", err)
		}
		currentChore = assigned
	}

	return nil
}

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
		if chore.SeriesID != nil || chore.RecurOnComplete || chore.DueDate == nil {
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
