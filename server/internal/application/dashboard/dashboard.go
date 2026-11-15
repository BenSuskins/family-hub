package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrLocation    = errors.New("load household time zone")
	ErrBins        = errors.New("load bin schedule")
	ErrChores      = errors.New("load due chores")
	ErrWeeklyMeals = errors.New("load weekly meals")
	ErrTodayMeals  = errors.New("load today's meals")
)

type BinCollection struct{ Date, Type, Name string }
type Chore struct {
	IsForecast                             bool
	ID, Name, Description, CreatedByUserID string
	CategoryID, AssignedToUserID           *string
	LastAssignedIndex                      int
	EligibleAssignees                      []string
	DueDate                                *time.Time
	DueTime                                *string
	RecurrenceType, RecurrenceValue        string
	RecurOnComplete                        bool
	SeriesID                               *string
	RecurrenceUntil                        *time.Time
	RecurrenceCount                        *int
	Status                                 string
	ResolvedAt                             *time.Time
	ResolvedByUserID                       *string
	CompletedAt                            *time.Time
	CompletedByUserID                      *string
	CreatedAt, UpdatedAt                   time.Time
}
type Meal struct {
	Date, MealType               string
	RecipeID                     *string
	Name, Notes, CreatedByUserID string
	CreatedAt, UpdatedAt         time.Time
}
type LocationSource interface {
	Location(context.Context) (*time.Location, error)
}
type BinSource interface {
	Collections(context.Context) ([]BinCollection, error)
}
type ChoreSource interface {
	Due(context.Context) ([]Chore, []Chore, error)
}
type MealSource interface {
	Between(context.Context, string, string) ([]Meal, error)
	On(context.Context, string) ([]Meal, error)
}
type Stats struct {
	ChoresToday, ChoresOverdue []Chore
	MealsThisWeek              int
	TodayMeals                 []Meal
	BinCollectionsTomorrow     []BinCollection
}
type Service struct {
	location LocationSource
	bins     BinSource
	chores   ChoreSource
	meals    MealSource
	now      func() time.Time
}

func New(location LocationSource, bins BinSource, chores ChoreSource, meals MealSource, now func() time.Time) *Service {
	return &Service{location: location, bins: bins, chores: chores, meals: meals, now: now}
}
func (service *Service) Stats(ctx context.Context) (Stats, error) {
	now := service.now()
	result := Stats{ChoresToday: []Chore{}, ChoresOverdue: []Chore{}, TodayMeals: []Meal{}, BinCollectionsTomorrow: []BinCollection{}}
	if service.location != nil {
		location, err := service.location.Location(ctx)
		if err != nil {
			return Stats{}, fmt.Errorf("%w: %w", ErrLocation, err)
		}
		collections, err := service.bins.Collections(ctx)
		if err != nil {
			return Stats{}, fmt.Errorf("%w: %w", ErrBins, err)
		}
		tomorrow := now.In(location).AddDate(0, 0, 1).Format("2006-01-02")
		for _, collection := range collections {
			if collection.Date == tomorrow {
				result.BinCollectionsTomorrow = append(result.BinCollectionsTomorrow, collection)
			}
		}
	}
	today, overdue, err := service.chores.Due(ctx)
	if err != nil {
		return Stats{}, fmt.Errorf("%w: %w", ErrChores, err)
	}
	for _, chore := range today {
		result.ChoresToday = append(result.ChoresToday, copyChore(chore))
	}
	for _, chore := range overdue {
		result.ChoresOverdue = append(result.ChoresOverdue, copyChore(chore))
	}
	if service.meals != nil {
		start := now.Truncate(24 * time.Hour)
		week, err := service.meals.Between(ctx, start.Format("2006-01-02"), start.AddDate(0, 0, 7).Format("2006-01-02"))
		if err != nil {
			return Stats{}, fmt.Errorf("%w: %w", ErrWeeklyMeals, err)
		}
		result.MealsThisWeek = len(week)
		meals, err := service.meals.On(ctx, now.Format("2006-01-02"))
		if err != nil {
			return Stats{}, fmt.Errorf("%w: %w", ErrTodayMeals, err)
		}
		for _, meal := range meals {
			meal.RecipeID = copyPointer(meal.RecipeID)
			result.TodayMeals = append(result.TodayMeals, meal)
		}
	}
	return result, nil
}
func copyPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
func copyChore(chore Chore) Chore {
	chore.CategoryID = copyPointer(chore.CategoryID)
	chore.AssignedToUserID = copyPointer(chore.AssignedToUserID)
	if chore.EligibleAssignees != nil {
		chore.EligibleAssignees = append([]string{}, chore.EligibleAssignees...)
	}
	chore.DueDate = copyPointer(chore.DueDate)
	chore.DueTime = copyPointer(chore.DueTime)
	chore.SeriesID = copyPointer(chore.SeriesID)
	chore.RecurrenceUntil = copyPointer(chore.RecurrenceUntil)
	chore.RecurrenceCount = copyPointer(chore.RecurrenceCount)
	chore.ResolvedAt = copyPointer(chore.ResolvedAt)
	chore.ResolvedByUserID = copyPointer(chore.ResolvedByUserID)
	chore.CompletedAt = copyPointer(chore.CompletedAt)
	chore.CompletedByUserID = copyPointer(chore.CompletedByUserID)
	return chore
}
