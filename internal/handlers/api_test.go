package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestAPITokenAuth_RejectsICalScopedToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tokenRepo := repository.NewAPITokenRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "ical-test@example.com",
		Name:        "iCal Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	rawToken := "ical-scoped-test-token"
	_, err = tokenRepo.Create(ctx, models.APIToken{
		Name:            "iCal Token",
		TokenHash:       repository.HashToken(rawToken),
		Scope:           "ical",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating ical token: %v", err)
	}

	apiHandler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil)

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(middleware.APITokenAuth(tokenRepo, userRepo))
		r.Get("/api/chores", apiHandler.ListChores)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/chores", nil)
	request.Header.Set("Authorization", "Bearer "+rawToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for ical-scoped token on API route, got %d", recorder.Code)
	}
}

func TestDeleteToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tokenRepo := repository.NewAPITokenRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	created, err := tokenRepo.Create(ctx, models.APIToken{
		Name:            "To Revoke",
		TokenHash:       "hash-revoke",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating token: %v", err)
	}

	handler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil)

	router := chi.NewRouter()
	router.Delete("/api/tokens/{id}", handler.DeleteToken)

	request := httptest.NewRequest(http.MethodDelete, "/api/tokens/"+created.ID, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	tokens, err := tokenRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("listing tokens after delete: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after revoke, got %d", len(tokens))
	}
}

func TestDashboardStats_IncludesChores(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-dashboard",
		Email:       "dashboard@example.com",
		Name:        "Dashboard User",
		Role:        models.RoleMember,
	})

	today := time.Now().Truncate(24 * time.Hour)
	_, _ = choreRepo.Create(ctx, models.Chore{
		Name:            "Due today chore",
		CreatedByUserID: user.ID,
		DueDate:         &today,
		Status:          models.ChoreStatusPending,
	})

	yesterday := today.AddDate(0, 0, -1)
	_, _ = choreRepo.Create(ctx, models.Chore{
		Name:            "Overdue chore",
		CreatedByUserID: user.ID,
		DueDate:         &yesterday,
		Status:          models.ChoreStatusOverdue,
	})

	handler := NewAPIHandler(choreRepo, userRepo, nil, nil, nil, nil, nil, nil)

	router := chi.NewRouter()
	router.Get("/api/dashboard", handler.DashboardStats)

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&body)

	dueTodayList, ok := body["chores_due_today_list"]
	if !ok {
		t.Fatal("expected chores_due_today_list in response")
	}
	if len(dueTodayList.([]interface{})) != 1 {
		t.Errorf("expected 1 chore due today, got %d", len(dueTodayList.([]interface{})))
	}

	overdueList, ok := body["chores_overdue_list"]
	if !ok {
		t.Fatal("expected chores_overdue_list in response")
	}
	if len(overdueList.([]interface{})) != 1 {
		t.Errorf("expected 1 overdue chore, got %d", len(overdueList.([]interface{})))
	}
}
