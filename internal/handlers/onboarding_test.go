package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func setupOnboardingHandler(t *testing.T) (*OnboardingHandler, models.User, *repository.SQLiteSettingsRepository, *repository.SQLiteCategoryRepository) {
	t.Helper()
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)

	user, err := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-setup-test",
		Email:       "admin@example.com",
		Name:        "Admin User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	handler := NewOnboardingHandler(settingsRepo, userRepo, categoryRepo)
	return handler, user, settingsRepo, categoryRepo
}

func TestOnboarding_SetupPage(t *testing.T) {
	handler, user, _, _ := setupOnboardingHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/setup", nil)
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.SetupPage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOnboarding_SaveFamilyName(t *testing.T) {
	handler, user, settingsRepo, _ := setupOnboardingHandler(t)

	form := url.Values{"family_name": {"The Smiths"}}
	req := httptest.NewRequest(http.MethodPost, "/setup/family-name", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.SaveFamilyName(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	saved, err := settingsRepo.Get(context.Background(), "family_name")
	if err != nil || saved != "The Smiths" {
		t.Errorf("expected family_name to be 'The Smiths', got %q (err: %v)", saved, err)
	}
}

func TestOnboarding_AcknowledgeUsers(t *testing.T) {
	handler, user, _, _ := setupOnboardingHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/setup/acknowledge-users", nil)
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.AcknowledgeUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOnboarding_CompleteSetup_WithCategory(t *testing.T) {
	handler, user, settingsRepo, categoryRepo := setupOnboardingHandler(t)

	form := url.Values{"category_name": {"Cleaning"}}
	req := httptest.NewRequest(http.MethodPost, "/setup/first-category", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.CompleteSetup(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("expected redirect to /, got %q", loc)
	}

	complete, _ := settingsRepo.Get(context.Background(), "onboarding_complete")
	if complete != "true" {
		t.Error("expected onboarding_complete to be true")
	}

	categories, _ := categoryRepo.FindAll(context.Background())
	if len(categories) != 1 || categories[0].Name != "Cleaning" {
		t.Errorf("expected one category named 'Cleaning', got %v", categories)
	}
}

func TestOnboarding_CompleteSetup_WithoutCategory(t *testing.T) {
	handler, user, settingsRepo, categoryRepo := setupOnboardingHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/setup/first-category", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.CompleteSetup(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}

	complete, _ := settingsRepo.Get(context.Background(), "onboarding_complete")
	if complete != "true" {
		t.Error("expected onboarding_complete to be true")
	}

	categories, _ := categoryRepo.FindAll(context.Background())
	if len(categories) != 0 {
		t.Errorf("expected no categories, got %v", categories)
	}
}

func TestOnboarding_WelcomePage(t *testing.T) {
	handler, user, _, _ := setupOnboardingHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/welcome", nil)
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.WelcomePage(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOnboarding_WelcomeStart(t *testing.T) {
	handler, user, _, _ := setupOnboardingHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/welcome/start", nil)
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.WelcomeStart(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestOnboarding_CompleteWelcome(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)

	user, _ := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-welcome-test",
		Email:       "member@example.com",
		Name:        "Old Name",
		Role:        models.RoleMember,
	})

	handler := NewOnboardingHandler(settingsRepo, userRepo, categoryRepo)

	form := url.Values{"name": {"New Name"}}
	req := httptest.NewRequest(http.MethodPost, "/welcome/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = requestWithUser(req, user)
	rec := httptest.NewRecorder()
	handler.CompleteWelcome(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Errorf("expected redirect to /, got %q", loc)
	}

	updated, _ := userRepo.FindByID(context.Background(), user.ID)
	if updated.Name != "New Name" {
		t.Errorf("expected name to be 'New Name', got %q", updated.Name)
	}
	if updated.OnboardedAt == nil {
		t.Error("expected OnboardedAt to be set")
	}
}
