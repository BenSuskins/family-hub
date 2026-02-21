package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func setupICalHandler(t *testing.T) (*ICalHandler, repository.APITokenRepository, repository.UserRepository) {
	t.Helper()
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	eventRepo := repository.NewEventRepository(database)
	userRepo := repository.NewUserRepository(database)
	tokenRepo := repository.NewAPITokenRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewICalHandler(choreRepo, eventRepo, userRepo, tokenRepo, settingsRepo, mealPlanRepo, "")
	return handler, tokenRepo, userRepo
}

func TestICalHandler_RejectsApiScopedToken(t *testing.T) {
	handler, tokenRepo, userRepo := setupICalHandler(t)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "api-scoped@example.com",
		Name:        "API Scoped User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	rawToken := "api-scoped-test-token"
	_, err = tokenRepo.Create(ctx, models.APIToken{
		Name:            "API Token",
		TokenHash:       repository.HashToken(rawToken),
		Scope:           "api",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating api token: %v", err)
	}

	router := chi.NewRouter()
	router.Get("/ical", handler.Feed)

	request := httptest.NewRequest(http.MethodGet, "/ical?token="+rawToken, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for api-scoped token on iCal route, got %d", recorder.Code)
	}
}

func TestICalHandler_AcceptsICalScopedToken(t *testing.T) {
	handler, tokenRepo, userRepo := setupICalHandler(t)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "ical-scoped@example.com",
		Name:        "iCal Scoped User",
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

	router := chi.NewRouter()
	router.Get("/ical", handler.Feed)

	request := httptest.NewRequest(http.MethodGet, "/ical?token="+rawToken, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusUnauthorized {
		t.Errorf("expected non-401 for ical-scoped token on iCal route, got %d", recorder.Code)
	}
}
