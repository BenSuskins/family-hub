package services

import (
	"encoding/json"
	"github.com/bensuskins/family-hub/internal/models"
	"testing"
	"time"
)

func TestCalendarViewAdapterPreservesAllFields(t *testing.T) {
	id := "id"
	n := 3
	at := time.Date(2026, 4, 2, 3, 4, 5, 0, time.UTC)
	chore := models.Chore{IsForecast: true, ID: "c", Name: "n", Description: "d", CreatedByUserID: "u", CategoryID: &id, AssignedToUserID: &id, LastAssignedIndex: 2, EligibleAssignees: []string{"u"}, DueDate: &at, DueTime: &id, RecurrenceType: "daily", RecurrenceValue: "v", RecurOnComplete: true, SeriesID: &id, RecurrenceUntil: &at, RecurrenceCount: &n, Status: "pending", ResolvedAt: &at, ResolvedByUserID: &id, CompletedAt: &at, CompletedByUserID: &id, CreatedAt: at, UpdatedAt: at}
	event := models.Event{ID: "e", Title: "t", Description: "d", Location: "l", StartTime: at, EndTime: &at, AllDay: true, Color: "c", CategoryID: &id, CreatedByUserID: "u", CreatedAt: at, UpdatedAt: at}
	meal := models.MealPlan{Date: "2026-04-02", MealType: "dinner", RecipeID: &id, Name: "m", Notes: "n", CreatedByUserID: "u", CreatedAt: at, UpdatedAt: at}
	for _, pair := range []struct {
		name             string
		original, mapped any
	}{{"chore", chore, calendarChoreValues([]models.Chore{chore})[0]}, {"event", event, calendarEventValues([]models.Event{event})[0]}, {"meal", meal, calendarMealValues([]models.MealPlan{meal})[0]}} {
		t.Run(pair.name, func(t *testing.T) {
			old, err := json.Marshal(pair.original)
			if err != nil {
				t.Fatal(err)
			}
			newValue, err := json.Marshal(pair.mapped)
			if err != nil {
				t.Fatal(err)
			}
			if string(old) != string(newValue) {
				t.Fatalf("original %s, mapped %s", old, newValue)
			}
		})
	}
}
