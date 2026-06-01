package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

var (
	ErrUserHasOverdueChores = errors.New("user has overdue chores")
	ErrChoreAlreadyComplete = errors.New("chore is already completed")
)

// SeedHorizon is how far ahead fixed-schedule recurring chores are materialized.
// A background top-up (TopUpAllSeries) keeps this window full, so it stays
// bounded to keep the row count and per-seed work small rather than a full year.
const SeedHorizon = 75 * 24 * time.Hour

// SeedHorizonFrom returns the materialization cutoff relative to now.
func SeedHorizonFrom(now time.Time) time.Time { return now.Add(SeedHorizon) }

type ChoreService struct {
	choreRepo      repository.ChoreRepository
	assignmentRepo repository.ChoreAssignmentRepository
	userRepo       repository.UserRepository
	seriesRepo     repository.ChoreSeriesRepository
}

func NewChoreService(
	choreRepo repository.ChoreRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	userRepo repository.UserRepository,
	seriesRepo repository.ChoreSeriesRepository,
) *ChoreService {
	return &ChoreService{
		choreRepo:      choreRepo,
		assignmentRepo: assignmentRepo,
		userRepo:       userRepo,
		seriesRepo:     seriesRepo,
	}
}

func (service *ChoreService) AssignNextUser(ctx context.Context, chore models.Chore) (models.Chore, error) {
	return service.assignNextUser(ctx, chore, chore.AssignedToUserID)
}

// assignNextUser assigns the chore to the next user in rotation. Rotation is
// anchored to lastAssignedUserID — the previous occupant, or the previous
// occurrence's assignee when seeding a series — so it survives users being
// added, removed, or renamed. It falls back to the positional LastAssignedIndex
// only when no previous assignee is known (e.g. the first assignment of a
// brand-new chore). A user with outstanding overdue chores is skipped within a
// single lap, but if everyone is overdue the rotation still advances rather than
// always landing on the same person.
func (service *ChoreService) assignNextUser(ctx context.Context, chore models.Chore, lastAssignedUserID *string) (models.Chore, error) {
	series := service.loadSeries(ctx, chore.SeriesID)

	candidates, err := service.findCandidates(ctx, chore.ID, series)
	if err != nil {
		return chore, err
	}

	if len(candidates) == 0 {
		return chore, errors.New("no users available for assignment")
	}

	// When a series definition exists, its rotation cursor is the authoritative
	// anchor so rotation is durable across the whole series; otherwise fall back
	// to the caller-supplied previous assignee.
	if series != nil {
		lastAssignedUserID = series.RotationCursorUserID
	}

	start := rotationStart(candidates, lastAssignedUserID, chore.LastAssignedIndex)

	chosenIndex := start
	for attempts := 0; attempts < len(candidates); attempts++ {
		candidateIndex := (start + attempts) % len(candidates)

		overdueCount, err := service.choreRepo.CountByStatusAndUser(ctx, models.ChoreStatusOverdue, candidates[candidateIndex].ID)
		if err != nil {
			return chore, fmt.Errorf("checking overdue chores: %w", err)
		}

		if overdueCount == 0 {
			chosenIndex = candidateIndex
			break
		}
	}

	assignedUser := candidates[chosenIndex]

	if chore.AssignedToUserID != nil {
		if err := service.assignmentRepo.MarkReassigned(ctx, chore.ID); err != nil {
			return chore, fmt.Errorf("marking old assignment: %w", err)
		}
	}

	chore.AssignedToUserID = &assignedUser.ID
	chore.LastAssignedIndex = chosenIndex

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

	// Advance the durable rotation cursor on the series so the next occurrence
	// continues from this assignee regardless of which row it lives on.
	if series != nil {
		if err := service.seriesRepo.SetRotationCursor(ctx, series.ID, assignedUser.ID); err != nil {
			return chore, fmt.Errorf("updating rotation cursor: %w", err)
		}
	}

	return chore, nil
}

// rotationStart returns the index of the candidate that should be tried first.
// When the previously assigned user is still in the candidate list, rotation
// continues from the person after them (robust to membership changes). Otherwise
// it falls back to advancing the stored positional index.
func rotationStart(candidates []models.User, lastAssignedUserID *string, lastIndex int) int {
	n := len(candidates)
	if lastAssignedUserID != nil {
		for i, user := range candidates {
			if user.ID == *lastAssignedUserID {
				return (i + 1) % n
			}
		}
	}
	return ((lastIndex+1)%n + n) % n
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

	// The series definition is authoritative for the recurrence rule, so a rule
	// edit is honored even by an in-flight occurrence created before the edit.
	rule := applySeriesRule(chore, service.loadSeries(ctx, chore.SeriesID))

	if rule.RecurrenceType == models.RecurrenceNone {
		return nil
	}

	if rule.RecurOnComplete {
		return service.createNextRecurrence(ctx, rule, now)
	}
	return service.SeedFutureOccurrences(ctx, rule, SeedHorizonFrom(now))

}

// loadSeries returns the chore_series definition for a chore, or nil when there
// is no series repository, no series id, or no definition row yet (the latter
// keeps pre-Phase-4 occurrences working via per-occurrence fields).
func (service *ChoreService) loadSeries(ctx context.Context, seriesID *string) *models.ChoreSeries {
	if service.seriesRepo == nil || seriesID == nil {
		return nil
	}
	series, err := service.seriesRepo.FindByID(ctx, *seriesID)
	if err != nil {
		slog.Error("loading chore series", "series_id", *seriesID, "error", err)
		return nil
	}
	return series
}

// SeriesByID exposes the series definition to handlers (e.g. to render the
// authoritative eligible pool on the edit form). Returns (nil, nil) when absent.
func (service *ChoreService) SeriesByID(ctx context.Context, seriesID string) (*models.ChoreSeries, error) {
	if service.seriesRepo == nil {
		return nil, nil
	}
	return service.seriesRepo.FindByID(ctx, seriesID)
}

// applySeriesRule overlays the recurrence rule, end conditions and timing from
// the series definition onto a chore copy, making the series the source of truth
// for behaviour. Occurrence rule columns remain as a denormalized read cache for
// display and filtering. No-op when there is no series definition.
func applySeriesRule(chore models.Chore, series *models.ChoreSeries) models.Chore {
	if series == nil {
		return chore
	}
	chore.RecurrenceType = series.RecurrenceType
	chore.RecurrenceValue = series.RecurrenceValue
	chore.RecurOnComplete = series.RecurOnComplete
	chore.RecurrenceUntil = series.RecurrenceUntil
	chore.RecurrenceCount = series.RecurrenceCount
	chore.DueTime = series.DueTime
	chore.CategoryID = series.CategoryID
	return chore
}

// findCandidates resolves the eligible assignee pool for a chore. When the chore
// belongs to a series with a definition row, the series pool is authoritative
// (so pool edits apply to every occurrence, including already-seeded ones);
// otherwise it falls back to the per-occurrence pool.
func (service *ChoreService) findCandidates(ctx context.Context, choreID string, series *models.ChoreSeries) ([]models.User, error) {
	var eligibleIDs []string
	if series != nil {
		eligibleIDs = series.EligibleAssignees
	} else {
		ids, err := service.choreRepo.GetEligibleAssignees(ctx, choreID)
		if err != nil {
			return nil, fmt.Errorf("getting eligible assignees: %w", err)
		}
		eligibleIDs = ids
	}

	allUsers, err := service.userRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	candidates := allUsers
	if len(eligibleIDs) > 0 {
		eligibleSet := make(map[string]bool, len(eligibleIDs))
		for _, id := range eligibleIDs {
			eligibleSet[id] = true
		}

		candidates = nil
		for _, user := range allUsers {
			if eligibleSet[user.ID] {
				candidates = append(candidates, user)
			}
		}
	}

	// Rotation anchors on candidate position, so the order must be stable and
	// independent of query plan or display name. Sort by immutable ID.
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
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
		// A series row must exist before chores.series_id can reference it (FK).
		return service.ensureSeriesRow(ctx, *chore)
	}
	chore.SeriesID = &chore.ID
	if err := service.ensureSeriesRow(ctx, *chore); err != nil {
		return err
	}
	return service.choreRepo.Update(ctx, *chore)
}

// ensureSeriesRow creates a minimal chore_series definition from the chore when
// one does not yet exist, satisfying the chores.series_id foreign key. The rule
// is taken from the chore's own fields (which, for in-memory chores, still carry
// the intended rule). No-op when the definition already exists.
func (service *ChoreService) ensureSeriesRow(ctx context.Context, chore models.Chore) error {
	if service.seriesRepo == nil || chore.SeriesID == nil {
		return nil
	}
	existing, err := service.seriesRepo.FindByID(ctx, *chore.SeriesID)
	if err != nil {
		return fmt.Errorf("checking series row: %w", err)
	}
	if existing != nil {
		return nil
	}
	_, err = service.seriesRepo.Create(ctx, models.ChoreSeries{
		ID:                   *chore.SeriesID,
		Name:                 chore.Name,
		Description:          chore.Description,
		CreatedByUserID:      chore.CreatedByUserID,
		CategoryID:           chore.CategoryID,
		DueTime:              chore.DueTime,
		RecurrenceType:       chore.RecurrenceType,
		RecurrenceValue:      chore.RecurrenceValue,
		RecurOnComplete:      chore.RecurOnComplete,
		RecurrenceUntil:      chore.RecurrenceUntil,
		RecurrenceCount:      chore.RecurrenceCount,
		RotationCursorUserID: chore.AssignedToUserID,
	})
	if err != nil {
		return fmt.Errorf("creating series row: %w", err)
	}
	return nil
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
		RecurrenceUntil:   template.RecurrenceUntil,
		RecurrenceCount:   template.RecurrenceCount,
		Status:            models.ChoreStatusPending,
	}
}

func (service *ChoreService) createNextRecurrence(ctx context.Context, chore models.Chore, completedAt time.Time) error {
	series := service.loadSeries(ctx, chore.SeriesID)
	chore = applySeriesRule(chore, series)

	nextDueDate, err := CalculateNextDueDate(chore, completedAt)
	if err != nil {
		return fmt.Errorf("calculating next due date: %w", err)
	}
	if nextDueDate == nil {
		return nil
	}

	// Stop the series once it reaches its end date.
	if chore.RecurrenceUntil != nil && nextDueDate.After(*chore.RecurrenceUntil) {
		return nil
	}

	if err := service.ensureSeriesID(ctx, &chore); err != nil {
		return fmt.Errorf("setting series_id on legacy chore: %w", err)
	}

	// Stop the series once it reaches its occurrence cap.
	if chore.RecurrenceCount != nil {
		existing, err := service.choreRepo.CountBySeries(ctx, *chore.SeriesID)
		if err != nil {
			return fmt.Errorf("counting series occurrences: %w", err)
		}
		if existing >= *chore.RecurrenceCount {
			return nil
		}
	}

	createdChore, err := service.choreRepo.Create(ctx, newChoreFromTemplate(chore, nextDueDate, chore.LastAssignedIndex))
	if err != nil {
		return fmt.Errorf("creating next chore instance: %w", err)
	}

	// The series owns the eligible pool; only fall back to copying the
	// per-occurrence pool when there is no series definition yet.
	if series == nil {
		if err := service.copyEligibleAssignees(ctx, chore.ID, createdChore.ID); err != nil {
			return err
		}
	}

	_, err = service.assignNextUser(ctx, createdChore, chore.AssignedToUserID)
	if err != nil {
		return fmt.Errorf("assigning next user: %w", err)
	}
	return nil
}

// SeedFutureOccurrences creates pending chore instances from the chore's series ahead to `until`.
// No-op for RecurOnComplete chores (can't predict completion dates) or chores without a DueDate.
// Idempotent: starts from the last existing future pending instance in the series.
func (service *ChoreService) SeedFutureOccurrences(ctx context.Context, chore models.Chore, until time.Time) error {
	series := service.loadSeries(ctx, chore.SeriesID)
	chore = applySeriesRule(chore, series)

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

	// Never materialize past the series end date.
	if chore.RecurrenceUntil != nil && chore.RecurrenceUntil.Before(until) {
		until = *chore.RecurrenceUntil
	}

	// Track how many occurrences already exist so we can honor the cap.
	existing := 0
	if chore.RecurrenceCount != nil {
		existing, err = service.choreRepo.CountBySeries(ctx, *chore.SeriesID)
		if err != nil {
			return fmt.Errorf("counting series occurrences: %w", err)
		}
	}

	current := *startChore.DueDate
	currentChore := startChore
	now := time.Now()

	for i := 0; i < maxExpansionIterations; i++ {
		if chore.RecurrenceCount != nil && existing >= *chore.RecurrenceCount {
			break
		}

		nextDate := advanceToNextOccurrence(current, chore.RecurrenceType, config)
		if !nextDate.Before(until) {
			break
		}
		current = nextDate

		// Catch-up policy: occurrences whose due date has already passed are
		// skipped, not back-filled. The cursor advances so the series resumes
		// at the next future date.
		if nextDate.Before(now) {
			continue
		}

		created, err := service.choreRepo.Create(ctx, newChoreFromTemplate(chore, &nextDate, currentChore.LastAssignedIndex))
		if err != nil {
			return fmt.Errorf("creating seeded chore instance: %w", err)
		}

		// The series owns the eligible pool; only fall back to per-occurrence
		// copies when there is no series definition yet.
		if series == nil {
			if err := service.copyEligibleAssignees(ctx, chore.ID, created.ID); err != nil {
				return err
			}
		}

		assigned, err := service.assignNextUser(ctx, created, currentChore.AssignedToUserID)
		if err != nil {
			return fmt.Errorf("assigning seeded chore: %w", err)
		}
		currentChore = assigned
		existing++
	}

	return nil
}

// TopUpAllSeries refills every active fixed-schedule recurring series up to
// `until`, so a series that is never completed does not run out of future
// occurrences (the materialization horizon is bounded). Safe to call repeatedly:
// SeedFutureOccurrences is idempotent. A failure on one series is logged and
// does not abort the rest.
func (service *ChoreService) TopUpAllSeries(ctx context.Context, until time.Time) error {
	anchors, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		RecurrenceTypes: []models.RecurrenceType{
			models.RecurrenceDaily,
			models.RecurrenceWeekly,
			models.RecurrenceMonthly,
			models.RecurrenceCustom,
			models.RecurrenceCalendar,
		},
		OnlyNextPerSeries: true,
	})
	if err != nil {
		return fmt.Errorf("finding series anchors: %w", err)
	}

	for _, anchor := range anchors {
		if anchor.RecurOnComplete || anchor.DueDate == nil {
			continue
		}
		if err := service.SeedFutureOccurrences(ctx, anchor, until); err != nil {
			slog.Error("topping up series", "chore_id", anchor.ID, "error", err)
		}
	}
	return nil
}

// SyncSeriesDefinition creates or updates the chore_series definition for a
// chore's series from the chore's current fields and eligible pool. On first
// creation the rotation cursor is seeded from the chore's current assignee.
// No-op when there is no series repository or the chore has no series id.
func (service *ChoreService) SyncSeriesDefinition(ctx context.Context, chore models.Chore, eligibleIDs []string) error {
	if service.seriesRepo == nil || chore.SeriesID == nil {
		return nil
	}
	seriesID := *chore.SeriesID

	existing, err := service.seriesRepo.FindByID(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("loading series for sync: %w", err)
	}

	def := models.ChoreSeries{
		ID:              seriesID,
		Name:            chore.Name,
		Description:     chore.Description,
		CreatedByUserID: chore.CreatedByUserID,
		CategoryID:      chore.CategoryID,
		DueTime:         chore.DueTime,
		RecurrenceType:  chore.RecurrenceType,
		RecurrenceValue: chore.RecurrenceValue,
		RecurOnComplete: chore.RecurOnComplete,
		RecurrenceUntil: chore.RecurrenceUntil,
		RecurrenceCount: chore.RecurrenceCount,
	}

	if existing == nil {
		def.RotationCursorUserID = chore.AssignedToUserID
		if _, err := service.seriesRepo.Create(ctx, def); err != nil {
			return fmt.Errorf("creating series definition: %w", err)
		}
	} else {
		def.CreatedAt = existing.CreatedAt
		def.RotationCursorUserID = existing.RotationCursorUserID
		def.DeletedAt = existing.DeletedAt
		if err := service.seriesRepo.Update(ctx, def); err != nil {
			return fmt.Errorf("updating series definition: %w", err)
		}
	}

	if err := service.seriesRepo.SetEligibleAssignees(ctx, seriesID, eligibleIDs); err != nil {
		return fmt.Errorf("setting series eligible assignees: %w", err)
	}
	return nil
}

// DeleteSeriesDefinition soft-deletes the series definition (audit trail).
func (service *ChoreService) DeleteSeriesDefinition(ctx context.Context, seriesID string) error {
	if service.seriesRepo == nil {
		return nil
	}
	return service.seriesRepo.MarkDeleted(ctx, seriesID)
}


func (service *ChoreService) UpdateOverdueChores(ctx context.Context) error {
	overdueChores, err := service.choreRepo.FindOverdueChores(ctx)
	if err != nil {
		return fmt.Errorf("finding overdue chores: %w", err)
	}

	for _, chore := range overdueChores {
		// FindOverdueChores returns rows already marked overdue too; skip them
		// so we don't rewrite unchanged rows every cycle. MarkOverdue is
		// additionally guarded by status = 'pending' in SQL.
		if chore.Status == models.ChoreStatusOverdue {
			continue
		}
		if err := service.choreRepo.MarkOverdue(ctx, chore.ID); err != nil {
			return fmt.Errorf("updating overdue chore %s: %w", chore.ID, err)
		}
	}

	return nil
}
