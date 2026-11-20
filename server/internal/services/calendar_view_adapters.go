package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"
	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type calendarLocation struct{ settings repository.SettingsRepository }

func (adapter calendarLocation) Location(ctx context.Context) (*time.Location, error) {
	value, err := adapter.settings.Get(ctx, "home_info")
	if errors.Is(err, sql.ErrNoRows) {
		return time.UTC, nil
	}
	if err != nil {
		return nil, err
	}
	var home models.HomeInfo
	if err := json.Unmarshal([]byte(value), &home); err != nil {
		return nil, fmt.Errorf("%w: %w", calendarapp.ErrViewLocationDecode, err)
	}
	if home.Timezone == "" {
		return time.UTC, nil
	}
	return time.LoadLocation(home.Timezone)
}

type calendarChores struct{ service *choreapp.ChoreService }

func (adapter calendarChores) Chores(ctx context.Context, start, end time.Time) ([]calendarapp.ViewChore, error) {
	items, err := adapter.service.FindChores(ctx, choreapp.ChoreFilter{DueAfter: &start, DueBefore: &end})
	return calendarChoreValues(items), err
}
func (adapter calendarChores) Forecast(ctx context.Context, start, end time.Time) ([]calendarapp.ViewChore, error) {
	items, err := adapter.service.Forecast(ctx, start, end)
	return calendarChoreValues(items), err
}
func calendarChoreValues(items []models.Chore) []calendarapp.ViewChore {
	mapped := dashboardChores(items)
	if mapped == nil {
		return nil
	}
	result := make([]calendarapp.ViewChore, len(mapped))
	for i, item := range mapped {
		result[i] = calendarapp.ViewChore(item)
	}
	return result
}

type calendarEvents struct{ service *calendarapp.Service }

func (adapter calendarEvents) Events(ctx context.Context, start, end time.Time) ([]calendarapp.ViewEvent, error) {
	items, err := adapter.service.FetchForRange(ctx, start, end)
	return calendarEventValues(items), err
}

func calendarEventValues(items []models.Event) []calendarapp.ViewEvent {
	if items == nil {
		return nil
	}
	result := make([]calendarapp.ViewEvent, len(items))
	for i, item := range items {
		result[i] = calendarapp.ViewEvent{ID: item.ID, Title: item.Title, Description: item.Description, Location: item.Location, StartTime: item.StartTime, EndTime: item.EndTime, AllDay: item.AllDay, Color: item.Color, CategoryID: item.CategoryID, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	}
	return result
}

type calendarMeals struct{ service *mealapp.Service }

func (adapter calendarMeals) Meals(ctx context.Context, from, to string) ([]calendarapp.ViewMeal, error) {
	items, err := adapter.service.FindAll(ctx, mealapp.Filter{DateFrom: from, DateTo: to})
	return calendarMealValues(items), err
}
func calendarMealValues(items []models.MealPlan) []calendarapp.ViewMeal {
	mapped := dashboardMeals(items)
	if mapped == nil {
		return nil
	}
	result := make([]calendarapp.ViewMeal, len(mapped))
	for i, item := range mapped {
		result[i] = calendarapp.ViewMeal(item)
	}
	return result
}

func NewCalendarView(settings repository.SettingsRepository, chores *choreapp.ChoreService, events *calendarapp.Service, meals *mealapp.Service, now func() time.Time) *calendarapp.ViewService {
	var location calendarapp.ViewLocation
	if settings != nil {
		location = calendarLocation{settings}
	}
	var eventSource calendarapp.ViewEvents
	if events != nil {
		eventSource = calendarEvents{events}
	}
	var mealSource calendarapp.ViewMeals
	if meals != nil {
		mealSource = calendarMeals{meals}
	}
	return calendarapp.NewView(location, calendarChores{chores}, eventSource, mealSource, now)
}
