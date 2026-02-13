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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, tokenRepo)

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
