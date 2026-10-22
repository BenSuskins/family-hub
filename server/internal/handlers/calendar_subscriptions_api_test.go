package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

type calendarRefreshFake struct {
	refreshedID string
	err         error
}

func (fake *calendarRefreshFake) ForceRefreshByID(_ context.Context, id string) error {
	fake.refreshedID = id
	return fake.err
}

func TestCalendarSubscriptionAPIRequiresAdmin(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	user := models.User{ID: "member", Role: models.RoleMember}
	handler := NewCalendarSubscriptionsAPIHandler(repository.NewICalSubscriptionRepository(database), &calendarRefreshFake{})
	request := httptest.NewRequest(http.MethodGet, "/api/calendars", nil)
	recorder := httptest.NewRecorder()
	handlerRouter(handler, user).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestCalendarSubscriptionAPIListsWithoutCachedData(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewICalSubscriptionRepository(database)
	cached := "BEGIN:VCALENDAR\r\nsecret cached calendar data"
	if err := repo.Create(context.Background(), models.ICalSubscription{ID: "school", Name: "School", URL: "https://example.com/calendar.ics"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateCache(context.Background(), "school", cached, time.Now()); err != nil {
		t.Fatal(err)
	}
	handler := NewCalendarSubscriptionsAPIHandler(repo, &calendarRefreshFake{})
	recorder := httptest.NewRecorder()
	handlerRouter(handler, models.User{ID: "admin", Role: models.RoleAdmin}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/calendars", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), cached) || strings.Contains(recorder.Body.String(), "CachedData") {
		t.Fatalf("calendar response exposed cached feed: %s", recorder.Body.String())
	}
	var feeds []CalendarSubscriptionResponse
	if err := json.NewDecoder(recorder.Body).Decode(&feeds); err != nil {
		t.Fatal(err)
	}
	if len(feeds) != 1 || feeds[0].ID != "school" || feeds[0].Name != "School" {
		t.Fatalf("unexpected feeds: %+v", feeds)
	}
}

func TestCalendarSubscriptionAPICreatesAndRejectsInvalidFeed(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewICalSubscriptionRepository(database)
	handler := NewCalendarSubscriptionsAPIHandler(repo, &calendarRefreshFake{})
	router := handlerRouter(handler, models.User{ID: "admin", Role: models.RoleAdmin})

	request := httptest.NewRequest(http.MethodPost, "/api/calendars", bytes.NewBufferString(`{"name":"School","url":"https://example.com/calendar.ics","color":"teal"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var created CalendarSubscriptionResponse
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Name != "School" || created.Color != "teal" || created.ID == "" {
		t.Fatalf("unexpected created feed: %+v", created)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/calendars", bytes.NewBufferString(`{"name":"Local","url":"http://127.0.0.1/calendar.ics","color":"teal"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid URL to return 400, got %d", recorder.Code)
	}
}

func TestCalendarSubscriptionAPIUpdatesColorRefreshesAndDeletes(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	repo := repository.NewICalSubscriptionRepository(database)
	if err := repo.Create(context.Background(), models.ICalSubscription{ID: "school", Name: "School", URL: "https://example.com/calendar.ics"}); err != nil {
		t.Fatal(err)
	}
	refresher := &calendarRefreshFake{}
	handler := NewCalendarSubscriptionsAPIHandler(repo, refresher)
	router := handlerRouter(handler, models.User{ID: "admin", Role: models.RoleAdmin})

	request := httptest.NewRequest(http.MethodPatch, "/api/calendars/school/color", strings.NewReader(`{"color":"rose"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected color update 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	feed, err := repo.FindByID(context.Background(), "school")
	if err != nil || feed.Color != "rose" {
		t.Fatalf("expected feed color rose, got %+v, %v", feed, err)
	}

	request = httptest.NewRequest(http.MethodPost, "/api/calendars/school/refresh", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || refresher.refreshedID != "school" {
		t.Fatalf("expected refresh 204 for school, got %d for %q", recorder.Code, refresher.refreshedID)
	}

	request = httptest.NewRequest(http.MethodDelete, "/api/calendars/school", nil)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", recorder.Code)
	}
}

func handlerRouter(handler *CalendarSubscriptionsAPIHandler, user models.User) *chi.Mux {
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Use(middleware.RequireAdmin)
	router.Get("/api/calendars", handler.List)
	router.Post("/api/calendars", handler.Create)
	router.Patch("/api/calendars/{id}/color", handler.UpdateColor)
	router.Post("/api/calendars/{id}/refresh", handler.Refresh)
	router.Delete("/api/calendars/{id}", handler.Delete)
	return router
}
