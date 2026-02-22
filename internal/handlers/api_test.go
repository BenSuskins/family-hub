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

	apiHandler := NewAPIHandler(nil, nil, nil, nil, tokenRepo)

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

	handler := NewAPIHandler(nil, nil, nil, nil, tokenRepo)

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
