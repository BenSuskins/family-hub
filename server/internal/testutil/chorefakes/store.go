package chorefakes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/chores"
	domain "github.com/bensuskins/family-hub/internal/domain/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type state struct {
	Chores      map[string]models.Chore
	Series      map[string]models.ChoreSeries
	Assignments map[string]models.ChoreAssignment
	Eligible    map[string][]string
	Users       []models.User
	Categories  []models.Category
}
type root struct {
	mutex sync.Mutex
	state state
}
type transactionGuard struct {
	mutex  sync.Mutex
	closed bool
}
type Store struct {
	root        *root
	transaction *state
	guard       *transactionGuard
}
type Assignments struct{ store *Store }
type Series struct{ store *Store }
type Users struct{ store *Store }

func New(users []models.User, categories ...models.Category) *Store {
	return &Store{root: &root{state: state{Chores: map[string]models.Chore{}, Series: map[string]models.ChoreSeries{}, Assignments: map[string]models.ChoreAssignment{}, Eligible: map[string][]string{}, Users: clone(users), Categories: clone(categories)}}}
}
func (store *Store) Assignments() *Assignments { return &Assignments{store} }
func (store *Store) Series() *Series           { return &Series{store} }
func (store *Store) Users() *Users             { return &Users{store} }
func clone[T any](value T) T {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var copied T
	if err := json.Unmarshal(data, &copied); err != nil {
		panic(err)
	}
	return copied
}
func (store *Store) access(ctx context.Context, operation func(*state) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store.transaction != nil {
		store.guard.mutex.Lock()
		defer store.guard.mutex.Unlock()
		if store.guard.closed {
			return errors.New("transaction is closed")
		}
		return operation(store.transaction)
	}
	store.root.mutex.Lock()
	defer store.root.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	return operation(&store.root.state)
}
func (store *Store) Transact(ctx context.Context, operation func(app.ChoreRepository, app.ChoreAssignmentRepository, app.UserRepository, app.ChoreSeriesRepository) error) error {
	if store.transaction != nil {
		return operation(store, store.Assignments(), store.Users(), store.Series())
	}
	return store.access(ctx, func(current *state) error {
		snapshot := clone(*current)
		guard := &transactionGuard{}
		scoped := &Store{root: store.root, transaction: &snapshot, guard: guard}
		defer func() { guard.mutex.Lock(); guard.closed = true; guard.mutex.Unlock() }()
		if err := operation(scoped, scoped.Assignments(), scoped.Users(), scoped.Series()); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		guard.mutex.Lock()
		*current = clone(snapshot)
		guard.closed = true
		guard.mutex.Unlock()
		return nil
	})
}
func (store *Store) FindByID(ctx context.Context, id string) (models.Chore, error) {
	var result models.Chore
	err := store.access(ctx, func(current *state) error {
		task, ok := current.Chores[id]
		if !ok {
			return app.ErrChoreNotFound
		}
		result = project(current, task)
		return nil
	})
	return result, err
}
func project(current *state, task models.Chore) models.Chore {
	task = clone(task)
	task.RecurrenceType = models.RecurrenceNone
	task.RecurrenceValue = ""
	task.RecurOnComplete = false
	task.RecurrenceUntil = nil
	task.RecurrenceCount = nil
	task.EligibleAssignees = nil
	if task.SeriesID != nil {
		if series, ok := current.Series[*task.SeriesID]; ok {
			task.RecurrenceType = series.RecurrenceType
			task.RecurrenceValue = series.RecurrenceValue
			task.RecurOnComplete = series.RecurOnComplete
			task.RecurrenceUntil = clone(series.RecurrenceUntil)
			task.RecurrenceCount = clone(series.RecurrenceCount)
		}
	}
	return task
}
func hasUser(current *state, identifier string) bool {
	for _, user := range current.Users {
		if user.ID == identifier {
			return true
		}
	}
	return false
}
func validEligible(current *state, identifiers []string) bool {
	seen := map[string]bool{}
	for _, id := range identifiers {
		if seen[id] || !hasUser(current, id) {
			return false
		}
		seen[id] = true
	}
	return true
}

func validCategory(current *state, id *string) bool {
	if id == nil {
		return true
	}
	for _, category := range current.Categories {
		if category.ID == *id {
			return true
		}
	}
	return false
}
func validUser(current *state, id *string) bool { return id == nil || hasUser(current, *id) }
func validStatus(status models.ChoreStatus) bool {
	switch status {
	case models.ChoreStatusPending, models.ChoreStatusCompleted, models.ChoreStatusOverdue, models.ChoreStatusSkipped, models.ChoreStatusSuperseded:
		return true
	}
	return false
}
func validRecurrence(recurrence models.RecurrenceType) bool {
	switch recurrence {
	case models.RecurrenceNone, models.RecurrenceDaily, models.RecurrenceWeekly, models.RecurrenceMonthly, models.RecurrenceCustom, models.RecurrenceCalendar:
		return true
	}
	return false
}
func checkOutstanding(current *state, task models.Chore) error {
	if task.SeriesID == nil || (task.Status != models.ChoreStatusPending && task.Status != models.ChoreStatusOverdue) {
		return nil
	}
	for _, existing := range current.Chores {
		if existing.ID != task.ID && existing.SeriesID != nil && *existing.SeriesID == *task.SeriesID && (existing.Status == models.ChoreStatusPending || existing.Status == models.ChoreStatusOverdue) {
			return errors.New("series already has an outstanding chore")
		}
	}
	return nil
}

func (store *Store) Create(ctx context.Context, task models.Chore) (models.Chore, error) {
	err := store.access(ctx, func(current *state) error {
		if !hasUser(current, task.CreatedByUserID) {
			return errors.New("creator does not exist")
		}
		if task.ID == "" {
			task.ID = uuid.NewString()
		}
		if _, ok := current.Chores[task.ID]; ok {
			return errors.New("chore already exists")
		}
		if task.SeriesID != nil {
			if _, ok := current.Series[*task.SeriesID]; !ok {
				return errors.New("series does not exist")
			}
		}
		task.CreatedAt = time.Now()
		task.UpdatedAt = task.CreatedAt
		if task.Status == "" {
			task.Status = models.ChoreStatusPending
		}
		if task.RecurrenceType == "" {
			task.RecurrenceType = models.RecurrenceNone
		}
		if !validStatus(task.Status) || !validCategory(current, task.CategoryID) || !validUser(current, task.AssignedToUserID) || !validUser(current, task.CompletedByUserID) {
			return errors.New("invalid chore")
		}
		if err := checkOutstanding(current, task); err != nil {
			return err
		}
		persisted := clone(task)
		persisted.ResolvedAt = nil
		persisted.ResolvedByUserID = nil
		current.Chores[task.ID] = persisted
		return nil
	})
	if err != nil {
		return models.Chore{}, err
	}
	return clone(task), nil
}
func (store *Store) Update(ctx context.Context, task models.Chore) error {
	return store.access(ctx, func(current *state) error {
		old, ok := current.Chores[task.ID]
		if !ok {
			return nil
		}
		task.CreatedAt = old.CreatedAt
		task.CreatedByUserID = old.CreatedByUserID
		if task.SeriesID != nil {
			if _, ok := current.Series[*task.SeriesID]; !ok {
				return errors.New("series does not exist")
			}
		}
		if !validStatus(task.Status) || !validCategory(current, task.CategoryID) || !validUser(current, task.AssignedToUserID) || !validUser(current, task.CompletedByUserID) || !validUser(current, task.ResolvedByUserID) {
			return errors.New("invalid chore")
		}
		if err := checkOutstanding(current, task); err != nil {
			return err
		}
		task.UpdatedAt = time.Now()
		current.Chores[task.ID] = clone(task)
		return nil
	})
}
func (store *Store) FindAll(ctx context.Context, filter app.ChoreFilter) ([]models.Chore, error) {
	var result []models.Chore
	err := store.access(ctx, func(current *state) error {
		if filter.OrderBy != "" && filter.OrderBy != app.OrderDueDate && filter.OrderBy != app.OrderCompleted && filter.OrderBy != app.OrderHistory {
			return fmt.Errorf("unknown chore ordering: %q", filter.OrderBy)
		}
		for _, raw := range current.Chores {
			task := project(current, raw)
			deleted := false
			if task.SeriesID != nil {
				series, ok := current.Series[*task.SeriesID]
				deleted = ok && series.DeletedAt != nil
			}
			if !filter.IncludeHistory && (deleted || (task.Status != models.ChoreStatusPending && task.Status != models.ChoreStatusOverdue && task.Status != models.ChoreStatusCompleted)) {
				continue
			}
			if filter.HistoryOnly && !deleted && task.Status != models.ChoreStatusCompleted && task.Status != models.ChoreStatusSkipped && task.Status != models.ChoreStatusSuperseded {
				continue
			}
			if filter.SeriesID != nil && (task.SeriesID == nil || *task.SeriesID != *filter.SeriesID) {
				continue
			}
			if filter.Status != nil && task.Status != *filter.Status {
				continue
			}
			if len(filter.Statuses) > 0 && !contains(filter.Statuses, task.Status) {
				continue
			}
			if len(filter.RecurrenceTypes) > 0 && !contains(filter.RecurrenceTypes, task.RecurrenceType) {
				continue
			}
			if filter.AssignedToUser != nil && (task.AssignedToUserID == nil || *task.AssignedToUserID != *filter.AssignedToUser) {
				continue
			}
			if filter.CategoryID != nil && (task.CategoryID == nil || *task.CategoryID != *filter.CategoryID) {
				continue
			}
			if filter.DueBefore != nil && (task.DueDate == nil || task.DueDate.After(*filter.DueBefore)) {
				continue
			}
			if filter.DueAfter != nil && (task.DueDate == nil || task.DueDate.Before(*filter.DueAfter)) {
				continue
			}
			if filter.AfterID != "" {
				cursor, ok := current.Chores[filter.AfterID]
				if !ok {
					continue
				}
				left, right := historyDate(task), historyDate(cursor)
				if left.After(right) || (left.Equal(right) && task.ID >= cursor.ID) {
					continue
				}
			}
			result = append(result, task)
		}
		if filter.OnlyNextPerSeries {
			earliest := map[string]models.Chore{}
			for _, task := range result {
				key := task.ID
				if task.SeriesID != nil {
					key = *task.SeriesID
				}
				existing, ok := earliest[key]
				if !ok || earlier(task.DueDate, existing.DueDate) {
					earliest[key] = task
				}
			}
			result = nil
			for _, task := range earliest {
				result = append(result, task)
			}
		}
		sort.SliceStable(result, func(i, j int) bool {
			a, b := result[i], result[j]
			switch filter.OrderBy {
			case app.OrderHistory:
				left, right := historyDate(a), historyDate(b)
				if left.Equal(right) {
					return a.ID > b.ID
				}
				return left.After(right)
			case app.OrderCompleted:
				if datesEqual(a.CompletedAt, b.CompletedAt) {
					return a.Name < b.Name
				}
				return later(a.CompletedAt, b.CompletedAt)
			default:
				if datesEqual(a.DueDate, b.DueDate) {
					return a.Name < b.Name
				}
				return earlier(a.DueDate, b.DueDate)
			}
		})
		if filter.Limit > 0 && len(result) > filter.Limit {
			result = result[:filter.Limit]
		}
		return nil
	})
	return clone(result), err
}
func contains[T comparable](values []T, value T) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func datesEqual(a, b *time.Time) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Equal(*b)
}
func earlier(a, b *time.Time) bool { return a != nil && (b == nil || a.Before(*b)) }
func later(a, b *time.Time) bool   { return a != nil && (b == nil || a.After(*b)) }
func historyDate(task models.Chore) time.Time {
	if task.ResolvedAt != nil {
		return *task.ResolvedAt
	}
	return task.CreatedAt
}
func (store *Store) FindOverdueChores(ctx context.Context) ([]models.Chore, error) {
	tasks, err := store.FindAll(ctx, app.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
	if err != nil {
		return nil, err
	}
	var result []models.Chore
	for _, task := range tasks {
		if domain.IsOverdue(task, time.Now()) {
			result = append(result, task)
		}
	}
	return result, nil
}
func (store *Store) MarkOverdue(ctx context.Context, id string) error {
	return store.access(ctx, func(current *state) error {
		task, ok := current.Chores[id]
		if ok && task.Status == models.ChoreStatusPending {
			task.Status = models.ChoreStatusOverdue
			task.UpdatedAt = time.Now()
			current.Chores[id] = task
		}
		return nil
	})
}
func (store *Store) SetEligibleAssignees(ctx context.Context, id string, users []string) error {
	return store.access(ctx, func(current *state) error {
		if len(users) > 0 {
			if _, ok := current.Chores[id]; !ok {
				return errors.New("chore does not exist")
			}
			if !validEligible(current, users) {
				return errors.New("invalid eligible users")
			}
		}
		current.Eligible[id] = clone(users)
		return nil
	})
}
func (store *Store) GetEligibleAssignees(ctx context.Context, id string) ([]string, error) {
	var result []string
	err := store.access(ctx, func(current *state) error { result = clone(current.Eligible[id]); sort.Strings(result); return nil })
	return result, err
}
func (store *Store) CountBySeries(ctx context.Context, id string) (int, error) {
	count := 0
	err := store.access(ctx, func(current *state) error {
		for _, task := range current.Chores {
			if task.SeriesID != nil && *task.SeriesID == id && task.Status != models.ChoreStatusSuperseded {
				count++
			}
		}
		return nil
	})
	return count, err
}
func (users *Users) FindAll(ctx context.Context) ([]models.User, error) {
	var result []models.User
	err := users.store.access(ctx, func(current *state) error { result = clone(current.Users); return nil })
	return result, err
}
func (assignments *Assignments) Create(ctx context.Context, assignment models.ChoreAssignment) (models.ChoreAssignment, error) {
	err := assignments.store.access(ctx, func(current *state) error {
		if _, ok := current.Chores[assignment.ChoreID]; !ok {
			return errors.New("chore does not exist")
		}
		if !hasUser(current, assignment.UserID) {
			return errors.New("user does not exist")
		}
		if assignment.ID == "" {
			assignment.ID = uuid.NewString()
		}
		assignment.AssignedAt = time.Now()
		if assignment.Status == "" {
			assignment.Status = models.AssignmentStatusAssigned
		}
		if assignment.Status != models.AssignmentStatusAssigned && assignment.Status != models.AssignmentStatusCompleted && assignment.Status != models.AssignmentStatusReassigned {
			return errors.New("invalid assignment status")
		}
		if _, exists := current.Assignments[assignment.ID]; exists {
			return errors.New("assignment already exists")
		}
		current.Assignments[assignment.ID] = clone(assignment)
		return nil
	})
	if err != nil {
		return models.ChoreAssignment{}, err
	}
	return clone(assignment), nil
}
func (assignments *Assignments) FindByChoreID(ctx context.Context, id string) ([]models.ChoreAssignment, error) {
	var result []models.ChoreAssignment
	err := assignments.store.access(ctx, func(current *state) error {
		for _, assignment := range current.Assignments {
			if assignment.ChoreID == id {
				result = append(result, clone(assignment))
			}
		}
		sort.SliceStable(result, func(i, j int) bool { return result[i].AssignedAt.After(result[j].AssignedAt) })
		return nil
	})
	return result, err
}

func (assignments *Assignments) MarkReassigned(ctx context.Context, id string) error {
	return assignments.store.access(ctx, func(current *state) error {
		for key, assignment := range current.Assignments {
			if assignment.ChoreID == id && assignment.Status == models.AssignmentStatusAssigned {
				assignment.Status = models.AssignmentStatusReassigned
				current.Assignments[key] = assignment
			}
		}
		return nil
	})
}
func (series *Series) FindByID(ctx context.Context, id string) (*models.ChoreSeries, error) {
	var result *models.ChoreSeries
	err := series.store.access(ctx, func(current *state) error {
		definition, ok := current.Series[id]
		if ok {
			copied := clone(definition)
			result = &copied
		}
		return nil
	})
	return result, err
}
func (series *Series) FindAll(ctx context.Context) ([]models.ChoreSeries, error) {
	result := []models.ChoreSeries{}
	err := series.store.access(ctx, func(current *state) error {
		for _, definition := range current.Series {
			result = append(result, clone(definition))
		}
		sort.Slice(result, func(i, j int) bool {
			left, right := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
			if left == right {
				return result[i].ID < result[j].ID
			}
			return left < right
		})
		return nil
	})
	return result, err
}
func (series *Series) Create(ctx context.Context, definition models.ChoreSeries) (models.ChoreSeries, error) {
	err := series.store.access(ctx, func(current *state) error {
		if !hasUser(current, definition.CreatedByUserID) {
			return errors.New("creator does not exist")
		}
		if definition.ID == "" {
			definition.ID = uuid.NewString()
		}
		if _, ok := current.Series[definition.ID]; ok {
			return errors.New("series already exists")
		}
		definition.CreatedAt = time.Now()
		definition.UpdatedAt = definition.CreatedAt
		if definition.RecurrenceType == "" {
			definition.RecurrenceType = models.RecurrenceNone
		}
		if !validRecurrence(definition.RecurrenceType) || !validCategory(current, definition.CategoryID) || !validUser(current, definition.RotationCursorUserID) {
			return errors.New("invalid series")
		}
		persisted := clone(definition)
		persisted.EligibleAssignees = nil
		current.Series[definition.ID] = persisted
		return nil
	})
	if err != nil {
		return models.ChoreSeries{}, err
	}
	return clone(definition), nil
}
func (series *Series) Update(ctx context.Context, definition models.ChoreSeries) error {
	return series.store.access(ctx, func(current *state) error {
		old, ok := current.Series[definition.ID]
		if !ok {
			return nil
		}
		if !validRecurrence(definition.RecurrenceType) || !validCategory(current, definition.CategoryID) || !validUser(current, definition.RotationCursorUserID) {
			return errors.New("invalid series")
		}
		definition.CreatedByUserID = old.CreatedByUserID
		definition.CreatedAt = old.CreatedAt
		definition.EligibleAssignees = old.EligibleAssignees
		definition.UpdatedAt = time.Now()
		current.Series[definition.ID] = clone(definition)
		return nil
	})
}
func (series *Series) SetRotationCursor(ctx context.Context, id, userID string) error {
	return series.store.access(ctx, func(current *state) error {
		definition, ok := current.Series[id]
		if ok && !hasUser(current, userID) {
			return errors.New("user does not exist")
		}
		if ok {
			definition.RotationCursorUserID = &userID
			current.Series[id] = definition
		}
		return nil
	})
}
func (series *Series) MarkDeleted(ctx context.Context, id string) error {
	return series.store.access(ctx, func(current *state) error {
		definition, ok := current.Series[id]
		if ok && definition.DeletedAt == nil {
			now := time.Now()
			definition.DeletedAt = &now
			current.Series[id] = definition
		}
		return nil
	})
}
func (series *Series) SetEligibleAssignees(ctx context.Context, id string, users []string) error {
	return series.store.access(ctx, func(current *state) error {
		definition, ok := current.Series[id]
		if len(users) > 0 {
			if !ok {
				return errors.New("series does not exist")
			}
			if !validEligible(current, users) {
				return errors.New("invalid eligible users")
			}
		}
		if ok {
			definition.EligibleAssignees = clone(users)
			sort.Strings(definition.EligibleAssignees)
			current.Series[id] = definition
		}
		return nil
	})
}
