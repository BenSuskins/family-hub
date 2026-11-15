package server

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

func TestOpenAPIDashboardRoutesContract(t *testing.T) {
	fixture := newContractFixture(t)
	routes := []struct{ path, token, wrong string }{{"/api/dashboard", "member", "display"}, {"/api/display/dashboard", "display", "member"}}
	for _, route := range routes {
		t.Run(route.path+" auth and empty", func(t *testing.T) {
			for _, token := range []string{"", route.wrong} {
				response := fixture.request(t, http.MethodGet, route.path, token, "", false)
				if response.Code != http.StatusUnauthorized {
					t.Fatalf("token %q: %d %s", token, response.Code, response.Body.String())
				}
			}
			response := fixture.request(t, http.MethodGet, route.path, route.token, "", true)
			if response.Code != http.StatusOK {
				t.Fatalf("empty: %d %s", response.Code, response.Body.String())
			}
			var data map[string]json.RawMessage
			if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"chores_due_today_list", "chores_overdue_list", "today_meals", "bin_collections_tomorrow"} {
				if string(data[field]) != "[]" {
					t.Fatalf("%s = %s, want []", field, data[field])
				}
			}
		})
	}
	ctx := context.Background()
	user, err := repository.NewUserRepository(fixture.database).FindByOIDCSubject(ctx, "member")
	if err != nil {
		t.Fatal(err)
	}
	chore, err := repository.NewChoreRepository(fixture.database).Create(ctx, models.Chore{Name: "Wash dishes", Description: "Kitchen", CreatedByUserID: user.ID, Status: models.ChoreStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	anchor := time.Now()
	date := anchor.Format("2006-01-02")
	if err := repository.NewMealPlanRepository(fixture.database).Upsert(ctx, models.MealPlan{Date: date, MealType: models.MealTypeDinner, Name: "Soup", Notes: "Warm", CreatedByUserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	tomorrow := anchor.In(location).AddDate(0, 0, 1).Format("2006-01-02")
	schedule, err := json.Marshal(models.BinSchedule{Enabled: true, Collections: []models.BinCollection{{Date: tomorrow, Type: "food_caddy", Name: "Food caddy"}, {Date: "1900-01-01", Type: "grey", Name: "Waste"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.NewSettingsRepository(fixture.database).Set(ctx, "bin_schedule", string(schedule)); err != nil {
		t.Fatal(err)
	}
	for _, route := range routes {
		t.Run(route.path+" populated", func(t *testing.T) {
			response := fixture.request(t, http.MethodGet, route.path, route.token, "", true)
			now := time.Now()
			if now.Format("2006-01-02") != date || now.In(location).AddDate(0, 0, 1).Format("2006-01-02") != tomorrow {
				t.Skip("date changed during dashboard request")
			}
			if response.Code != http.StatusOK {
				t.Fatalf("%d %s", response.Code, response.Body.String())
			}
			var body struct {
				ChoresDueToday int                          `json:"chores_due_today"`
				ChoresOverdue  int                          `json:"chores_overdue"`
				Chores         []map[string]json.RawMessage `json:"chores_due_today_list"`
				Meals          []map[string]json.RawMessage `json:"today_meals"`
				Bins           []map[string]string          `json:"bin_collections_tomorrow"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.ChoresDueToday != 1 || body.ChoresOverdue != 0 || len(body.Chores) != 1 {
				t.Fatalf("chores: %s", response.Body.String())
			}
			choreFields := []string{"IsForecast", "ID", "Name", "Description", "CreatedByUserID", "CategoryID", "AssignedToUserID", "LastAssignedIndex", "EligibleAssignees", "DueDate", "DueTime", "RecurrenceType", "RecurrenceValue", "RecurOnComplete", "SeriesID", "RecurrenceUntil", "RecurrenceCount", "Status", "ResolvedAt", "ResolvedByUserID", "CompletedAt", "CompletedByUserID", "CreatedAt", "UpdatedAt"}
			if len(body.Chores[0]) != len(choreFields) {
				t.Fatalf("chore fields: %s", response.Body.String())
			}
			for _, field := range choreFields {
				if _, ok := body.Chores[0][field]; !ok {
					t.Fatalf("missing chore field %s", field)
				}
			}
			var actualID string
			_ = json.Unmarshal(body.Chores[0]["ID"], &actualID)
			if actualID != chore.ID {
				t.Fatalf("chore ID %q", actualID)
			}
			if len(body.Meals) != 1 {
				t.Fatalf("meals: %s", response.Body.String())
			}
			mealFields := []string{"Date", "MealType", "RecipeID", "Name", "Notes", "CreatedByUserID", "CreatedAt", "UpdatedAt"}
			if len(body.Meals[0]) != len(mealFields) {
				t.Fatalf("meal fields: %s", response.Body.String())
			}
			for _, field := range mealFields {
				if _, ok := body.Meals[0][field]; !ok {
					t.Fatalf("missing meal field %s", field)
				}
			}
			if string(body.Meals[0]["Name"]) != `"Soup"` || string(body.Meals[0]["Notes"]) != `"Warm"` {
				t.Fatalf("meal: %s", response.Body.String())
			}
			if !reflect.DeepEqual(body.Bins, []map[string]string{{"date": tomorrow, "type": "food_caddy", "name": "Food caddy"}}) {
				t.Fatalf("bins: %+v", body.Bins)
			}
		})
	}
}
