package handlers

import (
	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"
	"time"
)

type calendarChoreResponse struct {
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
type calendarEventResponse struct {
	ID, Title, Description, Location string
	StartTime                        time.Time
	EndTime                          *time.Time
	AllDay                           bool
	Color                            string
	CategoryID                       *string
	CreatedByUserID                  string
	CreatedAt, UpdatedAt             time.Time
}
type calendarMealResponse struct {
	Date, MealType               string
	RecipeID                     *string
	Name, Notes, CreatedByUserID string
	CreatedAt, UpdatedAt         time.Time
}
type calendarViewResponse struct {
	Chores       []calendarChoreResponse `json:"chores"`
	Events       []calendarEventResponse `json:"events"`
	Meals        []calendarMealResponse  `json:"meals"`
	Availability map[string]string       `json:"availability"`
}

func mapCalendarView(result calendarapp.ViewResult) calendarViewResponse {
	response := calendarViewResponse{Chores: make([]calendarChoreResponse, len(result.Chores)), Events: make([]calendarEventResponse, len(result.Events)), Meals: make([]calendarMealResponse, len(result.Meals)), Availability: map[string]string{"events": availabilityStatus(result.EventsError == nil), "meals": availabilityStatus(result.MealsError == nil)}}
	for i, item := range result.Chores {
		response.Chores[i] = calendarChoreResponse{IsForecast: item.IsForecast, ID: item.ID, Name: item.Name, Description: item.Description, CreatedByUserID: item.CreatedByUserID, CategoryID: item.CategoryID, AssignedToUserID: item.AssignedToUserID, LastAssignedIndex: item.LastAssignedIndex, EligibleAssignees: item.EligibleAssignees, DueDate: item.DueDate, DueTime: item.DueTime, RecurrenceType: item.RecurrenceType, RecurrenceValue: item.RecurrenceValue, RecurOnComplete: item.RecurOnComplete, SeriesID: item.SeriesID, RecurrenceUntil: item.RecurrenceUntil, RecurrenceCount: item.RecurrenceCount, Status: item.Status, ResolvedAt: item.ResolvedAt, ResolvedByUserID: item.ResolvedByUserID, CompletedAt: item.CompletedAt, CompletedByUserID: item.CompletedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	}
	for i, item := range result.Events {
		response.Events[i] = calendarEventResponse{ID: item.ID, Title: item.Title, Description: item.Description, Location: item.Location, StartTime: item.StartTime, EndTime: item.EndTime, AllDay: item.AllDay, Color: item.Color, CategoryID: item.CategoryID, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	}
	for i, item := range result.Meals {
		response.Meals[i] = calendarMealResponse{Date: item.Date, MealType: item.MealType, RecipeID: item.RecipeID, Name: item.Name, Notes: item.Notes, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	}
	return response
}
