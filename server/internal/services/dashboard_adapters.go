package services

import (
	"context"
	"time"

	binsapp "github.com/bensuskins/family-hub/internal/application/bins"
	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"
	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type DashboardLocation struct{ settings repository.SettingsRepository }

func NewDashboardLocation(settings repository.SettingsRepository) *DashboardLocation {
	return &DashboardLocation{settings: settings}
}
func (adapter *DashboardLocation) Location(ctx context.Context) (*time.Location, error) {
	return HouseholdLocation(ctx, adapter.settings)
}

type DashboardBins struct{ service *binsapp.Service }

func NewDashboardBins(service *binsapp.Service) *DashboardBins {
	return &DashboardBins{service: service}
}
func (adapter *DashboardBins) Collections(ctx context.Context) ([]dashboardapp.BinCollection, error) {
	schedule, err := adapter.service.Get(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]dashboardapp.BinCollection, len(schedule.Collections))
	for i, item := range schedule.Collections {
		result[i] = dashboardapp.BinCollection{Date: item.Date, Type: item.Type, Name: item.Name}
	}
	return result, nil
}

type DashboardChores struct{ service *choreapp.ChoreService }

func NewDashboardChores(service *choreapp.ChoreService) *DashboardChores {
	return &DashboardChores{service: service}
}
func (adapter *DashboardChores) Due(ctx context.Context) ([]dashboardapp.Chore, []dashboardapp.Chore, error) {
	today, overdue, err := adapter.service.DueTasks(ctx)
	if err != nil {
		return nil, nil, err
	}
	return dashboardChores(today), dashboardChores(overdue), nil
}
func dashboardChores(chores []models.Chore) []dashboardapp.Chore {
	if chores == nil {
		return nil
	}
	result := make([]dashboardapp.Chore, len(chores))
	for index, chore := range chores {
		result[index] = dashboardapp.Chore{IsForecast: chore.IsForecast, ID: chore.ID, Name: chore.Name, Description: chore.Description, CreatedByUserID: chore.CreatedByUserID, CategoryID: chore.CategoryID, AssignedToUserID: chore.AssignedToUserID, LastAssignedIndex: chore.LastAssignedIndex, EligibleAssignees: chore.EligibleAssignees, DueDate: chore.DueDate, DueTime: chore.DueTime, RecurrenceType: string(chore.RecurrenceType), RecurrenceValue: chore.RecurrenceValue, RecurOnComplete: chore.RecurOnComplete, SeriesID: chore.SeriesID, RecurrenceUntil: chore.RecurrenceUntil, RecurrenceCount: chore.RecurrenceCount, Status: string(chore.Status), ResolvedAt: chore.ResolvedAt, ResolvedByUserID: chore.ResolvedByUserID, CompletedAt: chore.CompletedAt, CompletedByUserID: chore.CompletedByUserID, CreatedAt: chore.CreatedAt, UpdatedAt: chore.UpdatedAt}
	}
	return result
}

type DashboardMeals struct{ service *mealapp.Service }

func NewDashboardMeals(service *mealapp.Service) *DashboardMeals {
	return &DashboardMeals{service: service}
}
func (adapter *DashboardMeals) Between(ctx context.Context, from, to string) ([]dashboardapp.Meal, error) {
	meals, err := adapter.service.FindAll(ctx, mealapp.Filter{DateFrom: from, DateTo: to})
	return dashboardMeals(meals), err
}
func (adapter *DashboardMeals) On(ctx context.Context, date string) ([]dashboardapp.Meal, error) {
	meals, err := adapter.service.FindByDate(ctx, date)
	return dashboardMeals(meals), err
}
func dashboardMeals(meals []mealapp.Meal) []dashboardapp.Meal {
	if meals == nil {
		return nil
	}
	result := make([]dashboardapp.Meal, len(meals))
	for index, meal := range meals {
		result[index] = dashboardapp.Meal{Date: meal.Date, MealType: string(meal.MealType), RecipeID: meal.RecipeID, Name: meal.Name, Notes: meal.Notes, CreatedByUserID: meal.CreatedByUserID, CreatedAt: meal.CreatedAt, UpdatedAt: meal.UpdatedAt}
	}
	return result
}
