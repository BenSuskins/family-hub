package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestRequireOnboarding_RedirectsToSetupWhenNotComplete(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	settingsRepo := repository.NewSettingsRepository(database)
	// onboarding_complete not set → should redirect to /setup

	user := models.User{ID: "u1", Role: models.RoleAdmin}
	handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/setup" {
		t.Errorf("expected redirect to /setup, got %q", loc)
	}
}

func TestRequireOnboarding_RedirectsToWelcomeWhenUserNotOnboarded(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	settingsRepo := repository.NewSettingsRepository(database)
	_ = settingsRepo.Set(context.Background(), "onboarding_complete", "true")

	user := models.User{ID: "u2", OnboardedAt: nil}
	handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/welcome" {
		t.Errorf("expected redirect to /welcome, got %q", loc)
	}
}

func TestRequireOnboarding_PassesThroughWhenFullyOnboarded(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	settingsRepo := repository.NewSettingsRepository(database)
	_ = settingsRepo.Set(context.Background(), "onboarding_complete", "true")

	now := time.Now()
	user := models.User{ID: "u3", OnboardedAt: &now}
	handler := RequireOnboarding(settingsRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
