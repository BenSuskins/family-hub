package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCalendarIncludesTheLastDayAndOverlappingEvents(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	subscriptions := repository.NewICalSubscriptionRepository(database)
	ctx := context.Background()
	if err := subscriptions.Create(ctx, models.ICalSubscription{ID: "calendar", Name: "Calendar", URL: "https://example.com/calendar.ics"}); err != nil {
		t.Fatal(err)
	}
	feed := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
		"BEGIN:VEVENT\r\nUID:overlap\r\nSUMMARY:Across months\r\nDTSTART:20260331T230000Z\r\nDTEND:20260401T020000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:last\r\nSUMMARY:Last day\r\nDTSTART:20260430T120000Z\r\nDTEND:20260430T130000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:next\r\nSUMMARY:Next month\r\nDTSTART:20260501T000000Z\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:ended\r\nSUMMARY:Ends at start\r\nDTSTART:20260331T220000Z\r\nDTEND:20260401T000000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	if err := subscriptions.UpdateCache(ctx, "calendar", feed, time.Now()); err != nil {
		t.Fatal(err)
	}
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{
		OIDCSubject: "calendar-test", Email: "calendar@example.com", Name: "Calendar User", Role: models.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	chores := repository.NewChoreRepository(database)
	meals := repository.NewMealPlanRepository(database)
	for _, value := range []string{"2026-04-01T00:00:00Z", "2026-04-30T23:59:59Z", "2026-05-01T00:00:00Z"} {
		due, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := chores.Create(ctx, models.Chore{Name: value, DueDate: &due, CreatedByUserID: user.ID}); err != nil {
			t.Fatal(err)
		}
		if err := meals.Upsert(ctx, models.MealPlan{Date: due.Format(DateFormat), MealType: models.MealType("dinner"), Name: value, CreatedByUserID: user.ID}); err != nil {
			t.Fatal(err)
		}
	}
	handler := &APIHandler{
		choreRepo:    chores,
		icalFetcher:  services.NewICalFetcher(subscriptions),
		mealPlanRepo: meals,
	}
	response := httptest.NewRecorder()
	handler.ListCalendar(response, httptest.NewRequest(http.MethodGet, "/api/calendar?view=month&month=2026-04", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var calendar struct {
		Events []models.Event    `json:"events"`
		Chores []models.Chore    `json:"chores"`
		Meals  []models.MealPlan `json:"meals"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &calendar); err != nil {
		t.Fatal(err)
	}
	if len(calendar.Events) != 2 || calendar.Events[0].Title != "Across months" || calendar.Events[1].Title != "Last day" {
		t.Fatalf("events = %+v; want Across months and Last day", calendar.Events)
	}
	if len(calendar.Chores) != 2 || len(calendar.Meals) != 2 {
		t.Fatalf("chores = %d, meals = %d; want 2 each", len(calendar.Chores), len(calendar.Meals))
	}
	dayResponse := httptest.NewRecorder()
	handler.ListCalendar(dayResponse, httptest.NewRequest(http.MethodGet, "/api/calendar?view=day&date=2026-04-30", nil))
	if err := json.Unmarshal(dayResponse.Body.Bytes(), &calendar); err != nil {
		t.Fatal(err)
	}
	if len(calendar.Events) != 1 || len(calendar.Chores) != 1 || len(calendar.Meals) != 1 {
		t.Fatalf("last day: events = %d, chores = %d, meals = %d; want 1 each", len(calendar.Events), len(calendar.Chores), len(calendar.Meals))
	}

}
