package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestRetiredWebRoutesReturnJSONNotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	user, err := users.Create(context.Background(), models.User{
		OIDCSubject: "api-user",
		Email:       "api-user@example.com",
		Name:        "API user",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Create(context.Background(), models.APIToken{
		Name: "test", TokenHash: repository.HashToken("valid-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	}

	application := New(database, config.Config{}, nil)
	for _, path := range []string{
		"/", "/leaderboard", "/login", "/auth/callback", "/logout", "/profile",
		"/chores", "/calendars", "/meals", "/recipes", "/calendar", "/admin/users", "/static/app.css",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer valid-token")
			response := httptest.NewRecorder()
			application.router.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d with body %q", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json, got %q", response.Header().Get("Content-Type"))
			}
		})
	}
}

func TestAPIAuthenticationAndRoutingErrorsAreJSON(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	member, err := users.Create(context.Background(), models.User{
		OIDCSubject: "member",
		Email:       "member@example.com",
		Name:        "Member",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Create(context.Background(), models.APIToken{
		Name: "test", TokenHash: repository.HashToken("member-token"), Scope: models.TokenScopeAPI, CreatedByUserID: member.ID,
	}); err != nil {
		t.Fatal(err)
	}

	application := New(database, config.Config{}, nil)
	requests := []struct {
		name   string
		method string
		path   string
		token  string
		status int
	}{
		{name: "missing bearer token", method: http.MethodGet, path: "/api/me", status: http.StatusUnauthorized},
		{name: "invalid bearer token", method: http.MethodGet, path: "/api/me", token: "invalid", status: http.StatusUnauthorized},
		{name: "member denied admin route", method: http.MethodGet, path: "/api/tokens", token: "member-token", status: http.StatusForbidden},
		{name: "unknown route", method: http.MethodGet, path: "/api/not-a-route", token: "member-token", status: http.StatusNotFound},
		{name: "unsupported method", method: http.MethodPost, path: "/api/me", token: "member-token", status: http.StatusMethodNotAllowed},
	}

	for _, test := range requests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(""))
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()
			application.router.ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("expected status %d, got %d with body %q", test.status, response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json, got %q", response.Header().Get("Content-Type"))
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body["error"] == "" {
				t.Fatalf("expected JSON error body, got %q (decode error: %v)", response.Body.String(), err)
			}
		})
	}
}

func TestAuthenticatedAPIEndpointReturnsJSON(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	user, err := users.Create(context.Background(), models.User{
		OIDCSubject: "valid-api-user",
		Email:       "valid-api-user@example.com",
		Name:        "Valid API user",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Create(context.Background(), models.APIToken{
		Name: "test", TokenHash: repository.HashToken("valid-api-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID,
	}); err != nil {
		t.Fatal(err)
	}

	application := New(database, config.Config{})
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	request.Header.Set("Authorization", "Bearer valid-api-token")
	response := httptest.NewRecorder()
	application.router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200: %s", response.Code, response.Body.String())
	}
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("got Content-Type %q, want application/json", response.Header().Get("Content-Type"))
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if body["ID"] != user.ID {
		t.Errorf("got user ID %v, want %q", body["ID"], user.ID)
	}
}
