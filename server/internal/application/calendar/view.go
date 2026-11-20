package calendar

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrViewLocationDecode = errors.New("decode household timezone")
	ErrViewDate           = errors.New("invalid date")
	ErrViewMonth          = errors.New("invalid month")
	ErrViewChores         = errors.New("load chores")
	ErrViewForecast       = errors.New("load chore forecast")
)

type ViewChore struct {
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
type ViewEvent struct {
	ID, Title, Description, Location string
	StartTime                        time.Time
	EndTime                          *time.Time
	AllDay                           bool
	Color                            string
	CategoryID                       *string
	CreatedByUserID                  string
	CreatedAt, UpdatedAt             time.Time
}
type ViewMeal struct {
	Date, MealType               string
	RecipeID                     *string
	Name, Notes, CreatedByUserID string
	CreatedAt, UpdatedAt         time.Time
}
type ViewLocation interface {
	Location(context.Context) (*time.Location, error)
}
type ViewChores interface {
	Chores(context.Context, time.Time, time.Time) ([]ViewChore, error)
	Forecast(context.Context, time.Time, time.Time) ([]ViewChore, error)
}
type ViewEvents interface {
	Events(context.Context, time.Time, time.Time) ([]ViewEvent, error)
}
type ViewMeals interface {
	Meals(context.Context, string, string) ([]ViewMeal, error)
}
type ViewFailureSource string

const (
	ViewFailureLocation ViewFailureSource = "location"
	ViewFailureEvents   ViewFailureSource = "events"
	ViewFailureMeals    ViewFailureSource = "meals"
)

type ViewQuery struct {
	View, Date, Month string
	OnFailure         func(ViewFailureSource, error)
}

func (query ViewQuery) report(source ViewFailureSource, err error) {
	if query.OnFailure != nil {
		query.OnFailure(source, err)
	}
}

type ViewResult struct {
	Chores                                 []ViewChore
	Events                                 []ViewEvent
	Meals                                  []ViewMeal
	LocationError, EventsError, MealsError error
}
type ViewService struct {
	location ViewLocation
	chores   ViewChores
	events   ViewEvents
	meals    ViewMeals
	now      func() time.Time
}

func NewView(location ViewLocation, chores ViewChores, events ViewEvents, meals ViewMeals, now func() time.Time) *ViewService {
	return &ViewService{location, chores, events, meals, now}
}
func (service *ViewService) List(ctx context.Context, query ViewQuery) (ViewResult, error) {
	result := ViewResult{Chores: []ViewChore{}, Events: []ViewEvent{}, Meals: []ViewMeal{}}
	location := time.UTC
	if service.location != nil {
		got, err := service.location.Location(ctx)
		if err != nil {
			result.LocationError = err
			query.report(ViewFailureLocation, err)
		} else if got != nil {
			location = got
		}
	}
	now := service.now().In(location)
	var start, end time.Time
	switch query.View {
	case "week", "day":
		date := now
		if query.Date != "" {
			parsed, err := time.ParseInLocation("2006-01-02", query.Date, location)
			if err != nil {
				return result, ErrViewDate
			}
			date = parsed
		}
		offset := 0
		if query.View == "week" {
			offset = (int(date.Weekday()) + 6) % 7
		}
		start = time.Date(date.Year(), date.Month(), date.Day()-offset, 0, 0, 0, 0, location)
		if query.View == "week" {
			end = start.AddDate(0, 0, 7)
		} else {
			end = start.AddDate(0, 0, 1)
		}
	default:
		month := query.Month
		if month == "" {
			month = now.Format("2006-01")
		}
		parsed, err := time.ParseInLocation("2006-01", month, location)
		if err != nil {
			return result, ErrViewMonth
		}
		start = parsed
		end = start.AddDate(0, 1, 0)
	}
	chores, err := service.chores.Chores(ctx, start, end.Add(-time.Nanosecond))
	if err != nil {
		return result, fmt.Errorf("%w: %w", ErrViewChores, err)
	}
	forecasts, err := service.chores.Forecast(ctx, start, end)
	if err != nil {
		return result, fmt.Errorf("%w: %w", ErrViewForecast, err)
	}
	for _, chore := range chores {
		result.Chores = append(result.Chores, copyViewChore(chore))
	}
	for _, chore := range forecasts {
		result.Chores = append(result.Chores, copyViewChore(chore))
	}
	if service.events != nil {
		events, err := service.events.Events(ctx, start, end)
		if err != nil {
			result.EventsError = err
			query.report(ViewFailureEvents, err)
		} else {
			for _, event := range events {
				event.EndTime = copyViewPointer(event.EndTime)
				event.CategoryID = copyViewPointer(event.CategoryID)
				result.Events = append(result.Events, event)
			}
		}
	}
	if service.meals != nil {
		meals, err := service.meals.Meals(ctx, start.Format("2006-01-02"), end.AddDate(0, 0, -1).Format("2006-01-02"))
		if err != nil {
			result.MealsError = err
			query.report(ViewFailureMeals, err)
		} else {
			for _, meal := range meals {
				meal.RecipeID = copyViewPointer(meal.RecipeID)
				result.Meals = append(result.Meals, meal)
			}
		}
	}
	return result, nil
}

func copyViewPointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
func copyViewChore(chore ViewChore) ViewChore {
	chore.CategoryID = copyViewPointer(chore.CategoryID)
	chore.AssignedToUserID = copyViewPointer(chore.AssignedToUserID)
	if chore.EligibleAssignees != nil {
		chore.EligibleAssignees = append([]string{}, chore.EligibleAssignees...)
	}
	chore.DueDate = copyViewPointer(chore.DueDate)
	chore.DueTime = copyViewPointer(chore.DueTime)
	chore.SeriesID = copyViewPointer(chore.SeriesID)
	chore.RecurrenceUntil = copyViewPointer(chore.RecurrenceUntil)
	chore.RecurrenceCount = copyViewPointer(chore.RecurrenceCount)
	chore.ResolvedAt = copyViewPointer(chore.ResolvedAt)
	chore.ResolvedByUserID = copyViewPointer(chore.ResolvedByUserID)
	chore.CompletedAt = copyViewPointer(chore.CompletedAt)
	chore.CompletedByUserID = copyViewPointer(chore.CompletedByUserID)
	return chore
}
