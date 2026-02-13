package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func setupDashboardHandler(t *testing.T) (*DashboardHandler, models.User, *repository.SQLiteChoreRepository) {
	t.Helper()
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	choreRepo := repository.NewChoreRepository(database)
	eventRepo := repository.NewEventRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)
	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)

	user, err := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	handler := NewDashboardHandler(choreRepo, eventRepo, userRepo, assignmentRepo, choreService, mealPlanRepo, categoryRepo)
	return handler, user, choreRepo
}

func requestWithUser(request *http.Request, user models.User) *http.Request {
	ctx := context.WithValue(request.Context(), middleware.UserContextKey, user)
	return request.WithContext(ctx)
}

func TestDashboard_FullPage(t *testing.T) {
	handler, user, _ := setupDashboardHandler(t)

	router := chi.NewRouter()
	router.Get("/", handler.Dashboard)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = requestWithUser(request, user)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}
}

func TestLeaderboard_WeekPeriod(t *testing.T) {
	handler, user, _ := setupDashboardHandler(t)

	router := chi.NewRouter()
	router.Get("/leaderboard", handler.Leaderboard)

	request := httptest.NewRequest(http.MethodGet, "/leaderboard?period=week", nil)
	request = requestWithUser(request, user)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}
}

func TestLeaderboard_MonthPeriod(t *testing.T) {
	handler, user, _ := setupDashboardHandler(t)

	router := chi.NewRouter()
	router.Get("/leaderboard", handler.Leaderboard)

	request := httptest.NewRequest(http.MethodGet, "/leaderboard?period=month", nil)
	request = requestWithUser(request, user)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", recorder.Code)
	}
}
