package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestAdminHandler_CreateToken_ReturnsHTML(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	tokenRepo := repository.NewAPITokenRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)

	admin := models.User{Name: "Admin", Email: "admin@test.com", Role: models.RoleAdmin}
	created, err := userRepo.Create(t.Context(), admin)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	admin = created

	handler := NewAdminHandler(userRepo, tokenRepo, settingsRepo, categoryRepo, assignmentRepo)

	form := url.Values{"name": {"mytoken"}, "scope": {"api"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/tokens", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = requestWithUser(req, admin)

	w := httptest.NewRecorder()
	handler.CreateToken(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, `{"`) {
		t.Error("response contains raw JSON, expected HTML")
	}
	if !strings.Contains(body, "mytoken") {
		t.Error("expected token name in response HTML")
	}
}
