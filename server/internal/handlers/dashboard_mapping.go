package handlers

import (
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"
	"time"
)

type dashboardChoreResponse struct {
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
type dashboardMealResponse struct {
	Date, MealType               string
	RecipeID                     *string
	Name, Notes, CreatedByUserID string
	CreatedAt, UpdatedAt         time.Time
}
type dashboardBinResponse struct {
	Date string `json:"date"`
	Type string `json:"type"`
	Name string `json:"name"`
}
type dashboardResponse struct {
	ChoresDueToday         int                      `json:"chores_due_today"`
	ChoresOverdue          int                      `json:"chores_overdue"`
	ChoresDueTodayList     []dashboardChoreResponse `json:"chores_due_today_list"`
	ChoresOverdueList      []dashboardChoreResponse `json:"chores_overdue_list"`
	MealsThisWeek          int                      `json:"meals_this_week"`
	TodayMeals             []dashboardMealResponse  `json:"today_meals"`
	BinCollectionsTomorrow []dashboardBinResponse   `json:"bin_collections_tomorrow"`
}

func mapDashboard(stats dashboardapp.Stats) dashboardResponse {
	response := dashboardResponse{ChoresDueToday: len(stats.ChoresToday), ChoresOverdue: len(stats.ChoresOverdue), MealsThisWeek: stats.MealsThisWeek, ChoresDueTodayList: make([]dashboardChoreResponse, 0, len(stats.ChoresToday)), ChoresOverdueList: make([]dashboardChoreResponse, 0, len(stats.ChoresOverdue)), TodayMeals: make([]dashboardMealResponse, 0, len(stats.TodayMeals)), BinCollectionsTomorrow: make([]dashboardBinResponse, 0, len(stats.BinCollectionsTomorrow))}
	for _, item := range stats.ChoresToday {
		response.ChoresDueTodayList = append(response.ChoresDueTodayList, mapDashboardChore(item))
	}
	for _, item := range stats.ChoresOverdue {
		response.ChoresOverdueList = append(response.ChoresOverdueList, mapDashboardChore(item))
	}
	for _, item := range stats.TodayMeals {
		response.TodayMeals = append(response.TodayMeals, dashboardMealResponse{item.Date, item.MealType, item.RecipeID, item.Name, item.Notes, item.CreatedByUserID, item.CreatedAt, item.UpdatedAt})
	}
	for _, item := range stats.BinCollectionsTomorrow {
		response.BinCollectionsTomorrow = append(response.BinCollectionsTomorrow, dashboardBinResponse{item.Date, item.Type, item.Name})
	}
	return response
}
func mapDashboardChore(item dashboardapp.Chore) dashboardChoreResponse {
	return dashboardChoreResponse{IsForecast: item.IsForecast, ID: item.ID, Name: item.Name, Description: item.Description, CreatedByUserID: item.CreatedByUserID, CategoryID: item.CategoryID, AssignedToUserID: item.AssignedToUserID, LastAssignedIndex: item.LastAssignedIndex, EligibleAssignees: item.EligibleAssignees, DueDate: item.DueDate, DueTime: item.DueTime, RecurrenceType: item.RecurrenceType, RecurrenceValue: item.RecurrenceValue, RecurOnComplete: item.RecurOnComplete, SeriesID: item.SeriesID, RecurrenceUntil: item.RecurrenceUntil, RecurrenceCount: item.RecurrenceCount, Status: item.Status, ResolvedAt: item.ResolvedAt, ResolvedByUserID: item.ResolvedByUserID, CompletedAt: item.CompletedAt, CompletedByUserID: item.CompletedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
