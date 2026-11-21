package services

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	mealapp "github.com/bensuskins/family-hub/internal/application/meals"

	"github.com/bensuskins/family-hub/internal/models"
)

func TestDashboardAdapterPreservesChoreAndMealFields(t *testing.T) {
	value := "value"
	count := 7
	instant := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	chore := models.Chore{IsForecast: true, ID: "id", Name: "name", Description: "description", CreatedByUserID: "creator", CategoryID: &value, AssignedToUserID: &value, LastAssignedIndex: 2, EligibleAssignees: []string{"one", "two"}, DueDate: &instant, DueTime: &value, RecurrenceType: models.RecurrenceType("weekly"), RecurrenceValue: "test", RecurOnComplete: true, SeriesID: &value, RecurrenceUntil: &instant, RecurrenceCount: &count, Status: models.ChoreStatus("pending"), ResolvedAt: &instant, ResolvedByUserID: &value, CompletedAt: &instant, CompletedByUserID: &value, CreatedAt: instant, UpdatedAt: instant}
	meal := mealapp.Meal{Date: "2024-01-02", MealType: mealapp.Type("dinner"), RecipeID: &value, Name: "meal", Notes: "notes", CreatedByUserID: "creator", CreatedAt: instant, UpdatedAt: instant}
	compareJSON := func(name string, original, mapped any) {
		t.Helper()
		originalJSON, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		mappedJSON, err := json.Marshal(mapped)
		if err != nil {
			t.Fatal(err)
		}
		var first, second any
		if err := json.Unmarshal(originalJSON, &first); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(mappedJSON, &second); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Errorf("%s fields changed: %s vs %s", name, originalJSON, mappedJSON)
		}
	}
	compareJSON("chore", chore, dashboardChores([]models.Chore{chore})[0])
	compareJSON("meal", meal, dashboardMeals([]mealapp.Meal{meal})[0])
	if dashboardChores(nil) != nil || dashboardMeals(nil) != nil {
		t.Fatal("nil inputs changed")
	}
}
