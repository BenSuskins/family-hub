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
	"github.com/google/uuid"
)

var (
	ErrChoreAlreadyComplete = errors.New("chore is already completed")
)

type ChoreService struct {
	choreRepo      repository.ChoreRepository
	assignmentRepo repository.ChoreAssignmentRepository
	userRepo       repository.UserRepository
	seriesRepo     repository.ChoreSeriesRepository
	now            func() time.Time
	settings       repository.SettingsRepository
}

func NewChoreService(
	choreRepo repository.ChoreRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	userRepo repository.UserRepository,
	seriesRepo repository.ChoreSeriesRepository,
	options ...ChoreOption,
) *ChoreService {
	service := &ChoreService{
		choreRepo:      choreRepo,
		assignmentRepo: assignmentRepo,
		userRepo:       userRepo,
		seriesRepo:     seriesRepo,
		now:            time.Now,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func (service *ChoreService) AssignNextUser(ctx context.Context, chore models.Chore) (models.Chore, error) {
	return service.assignNextUser(ctx, chore, chore.AssignedToUserID)
}

// Anchor rotation to immutable user IDs so name changes do not change the order.
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

type ChoreOption func(*ChoreService)

func WithChoreClock(clock func() time.Time) ChoreOption {
	return func(service *ChoreService) { service.now = clock }
}
func WithChoreSettings(settings repository.SettingsRepository) ChoreOption {
	return func(service *ChoreService) { service.settings = settings }
}

func (service *ChoreService) CompleteChore(ctx context.Context, choreID, userID string) error {
	return service.resolve(ctx, choreID, userID, models.ChoreStatusCompleted)
}
func (service *ChoreService) SkipChore(ctx context.Context, choreID, userID string) error {
	return service.resolve(ctx, choreID, userID, models.ChoreStatusSkipped)
}
func (service *ChoreService) householdNow(ctx context.Context) (time.Time, error) {
	timezone := "Europe/London"
	if service.settings != nil {
		settings, err := LoadChoreReminders(ctx, service.settings)
		if err != nil {
			return time.Time{}, err
		}
		timezone = settings.Timezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, err
	}
	return service.now().In(location), nil
}
func (service *ChoreService) resolve(ctx context.Context, choreID, userID string, status models.ChoreStatus) error {
	now, err := service.householdNow(ctx)
	if err != nil {
		return err
	}
	return service.choreRepo.Transact(ctx, func(chores repository.ChoreRepository, assignments repository.ChoreAssignmentRepository, users repository.UserRepository, definitions repository.ChoreSeriesRepository) error {
		scoped := *service
		scoped.choreRepo = chores
		scoped.assignmentRepo = assignments
		scoped.userRepo = users
		scoped.seriesRepo = definitions
		task, err := chores.FindByID(ctx, choreID)
		if err != nil {
			return err
		}
		if task.Status != models.ChoreStatusPending && task.Status != models.ChoreStatusOverdue {
			return ErrChoreAlreadyComplete
		}
		var definition *models.ChoreSeries
		if task.SeriesID != nil {
			definition, err = definitions.FindByID(ctx, *task.SeriesID)
			if err != nil {
				return err
			}
		}
		if definition != nil && definition.DeletedAt != nil {
			return ErrChoreAlreadyComplete
		}
		task.Status = status
		task.ResolvedAt = &now
		task.ResolvedByUserID = &userID
		if status == models.ChoreStatusCompleted {
			task.CompletedAt = &now
			task.CompletedByUserID = &userID
		}
		if err := chores.Update(ctx, task); err != nil {
			return err
		}
		if status == models.ChoreStatusCompleted {
			if err := assignments.MarkReassigned(ctx, choreID); err != nil {
				return err
			}
			if _, err := assignments.Create(ctx, models.ChoreAssignment{ChoreID: choreID, UserID: userID, Status: models.AssignmentStatusCompleted, CompletedAt: &now}); err != nil {
				return err
			}
		} else if err := assignments.MarkReassigned(ctx, choreID); err != nil {
			return err
		}
		rule := applySeriesRule(task, definition)
		if definition != nil {
			rule.Name = definition.Name
			rule.Description = definition.Description
		}
		next, err := nextTaskDate(rule, now)
		if err != nil {
			return err
		}
		if next == nil {
			return nil
		}
		if rule.RecurrenceUntil != nil && next.Format("2006-01-02") > rule.RecurrenceUntil.Format("2006-01-02") {
			return nil
		}
		if err := scoped.ensureSeriesID(ctx, &rule); err != nil {
			return err
		}
		if rule.RecurrenceCount != nil {
			count, err := chores.CountBySeries(ctx, *rule.SeriesID)
			if err != nil {
				return err
			}
			if count >= *rule.RecurrenceCount {
				return nil
			}
		}
		if definition != nil && task.AssignedToUserID != nil {
			if err := definitions.SetRotationCursor(ctx, definition.ID, *task.AssignedToUserID); err != nil {
				return err
			}
		}
		created, err := chores.Create(ctx, newChoreFromTemplate(rule, next, task.LastAssignedIndex))
		if err != nil {
			return err
		}
		if definition == nil {
			if err := scoped.copyEligibleAssignees(ctx, task.ID, created.ID); err != nil {
				return err
			}
		}
		_, err = scoped.assignNextUser(ctx, created, task.AssignedToUserID)
		return err
	})
}

func nextTaskDate(task models.Chore, now time.Time) (*time.Time, error) {
	if task.RecurrenceType == models.RecurrenceNone || task.RecurrenceType == "" {
		return nil, nil
	}
	config, err := parseConfig(task.RecurrenceValue)
	if err != nil {
		return nil, err
	}
	if task.RecurOnComplete {
		return CalculateNextDueDate(task, now)
	}
	if task.DueDate == nil {
		task.DueDate = &now
	}
	date := *task.DueDate
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, now.Location())
	for iteration := 0; iteration < 36600; iteration++ {
		next := advanceToNextOccurrence(date, task.RecurrenceType, config)
		if !next.After(date) {
			return nil, errors.New("recurrence does not advance")
		}
		date = next
		if date.Format("2006-01-02") > now.Format("2006-01-02") {
			return &date, nil
		}
	}
	return nil, errors.New("recurrence exceeds date limit")
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
// for behaviour. Historical task names and owners remain unchanged.
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

func (service *ChoreService) CreateChore(ctx context.Context, task models.Chore, eligible []string) (models.Chore, error) {
	var created models.Chore
	err := service.choreRepo.Transact(ctx, func(chores repository.ChoreRepository, assignments repository.ChoreAssignmentRepository, users repository.UserRepository, definitions repository.ChoreSeriesRepository) error {
		scoped := *service
		scoped.choreRepo = chores
		scoped.assignmentRepo = assignments
		scoped.userRepo = users
		scoped.seriesRepo = definitions
		identifier := uuid.NewString()
		task.ID = identifier
		task.SeriesID = &identifier
		task.LastAssignedIndex = -1
		if err := scoped.SyncSeriesDefinition(ctx, task, eligible); err != nil {
			return err
		}
		var err error
		created, err = chores.Create(ctx, task)
		if err != nil {
			return err
		}
		created, err = scoped.AssignNextUser(ctx, created)
		return err
	})
	return created, err
}

func (service *ChoreService) UpdateDefinition(ctx context.Context, task models.Chore, eligible []string) error {
	return service.choreRepo.Transact(ctx, func(chores repository.ChoreRepository, assignments repository.ChoreAssignmentRepository, users repository.UserRepository, definitions repository.ChoreSeriesRepository) error {
		scoped := *service
		scoped.choreRepo = chores
		scoped.seriesRepo = definitions
		if err := scoped.SyncSeriesDefinition(ctx, task, eligible); err != nil {
			return err
		}
		current, err := chores.FindAll(ctx, repository.ChoreFilter{SeriesID: task.SeriesID, Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
		if err != nil {
			return err
		}
		for _, item := range current {
			item.Name = task.Name
			item.Description = task.Description
			item.CategoryID = task.CategoryID
			item.DueTime = task.DueTime
			if err := chores.Update(ctx, item); err != nil {
				return err
			}
		}
		return nil
	})
}
