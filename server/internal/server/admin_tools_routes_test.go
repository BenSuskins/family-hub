package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestAdminToolsRoutesRequireAdminAndServeMobileBackupAPI(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	createToken := func(subject string, role models.Role, value string) {
		t.Helper()
		user, err := users.Create(context.Background(), models.User{
			OIDCSubject: subject,
			Email:       subject + "@example.com",
			Name:        subject,
			Role:        role,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tokens.Create(context.Background(), models.APIToken{
			Name: "test", TokenHash: repository.HashToken(value), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID,
		}); err != nil {
			t.Fatal(err)
		}
	}
	createToken("admin", models.RoleAdmin, "admin-token")
	createToken("member", models.RoleMember, "member-token")
	application := New(database, config.Config{}, nil)

	request := func(method, path, token string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(""))
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		application.router.ServeHTTP(response, req)
		return response
	}

	if response := request(http.MethodGet, "/api/calendars", "member-token"); response.Code != http.StatusForbidden {
		t.Fatalf("member listed calendars: %d", response.Code)
	}
	if response := request(http.MethodGet, "/api/calendars", "admin-token"); response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "[]") {
		t.Fatalf("admin calendar list response: %d %s", response.Code, response.Body.String())
	}
	if response := request(http.MethodGet, "/api/admin/backup", "member-token"); response.Code != http.StatusForbidden {
		t.Fatalf("member downloaded database backup: %d", response.Code)
	}
	if response := request(http.MethodGet, "/api/admin/backup", "admin-token"); response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/gzip" {
		t.Fatalf("admin backup response: %d %s", response.Code, response.Header().Get("Content-Type"))
	}
	if response := request(http.MethodPost, "/api/admin/restore", "member-token"); response.Code != http.StatusForbidden {
		t.Fatalf("member restored database: %d", response.Code)
	}
	if response := request(http.MethodPost, "/api/admin/restore", "admin-token"); response.Code != http.StatusBadRequest {
		t.Fatalf("admin invalid restore response: %d", response.Code)
	}
}
