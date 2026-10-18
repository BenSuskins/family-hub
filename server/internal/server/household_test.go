package server

import (
	"context"
	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHouseholdRoutesRequireAuthenticationAndAdmin(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	user, err := users.Create(context.Background(), models.User{OIDCSubject: "member", Email: "member@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Create(context.Background(), models.APIToken{Name: "test", TokenHash: repository.HashToken("member-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	server := New(database, config.Config{}, nil)
	for _, path := range []string{"/api/home-info", "/api/weather", "/api/departures", "/api/shopping"} {
		for _, token := range []string{"invalid", "member-token"} {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			server.router.ServeHTTP(response, request)
			expected := 200
			if token == "invalid" {
				expected = 401
			}
			if response.Code != expected {
				t.Fatalf("%s with %s: %d %s", path, token, response.Code, response.Body.String())
			}
		}
	}
	request := httptest.NewRequest(http.MethodPatch, "/api/home-info", strings.NewReader(`{"arrivalMarginMinutes":20}`))
	request.Header.Set("Authorization", "Bearer member-token")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != 403 {
		t.Fatalf("member changed home settings: %d", response.Code)
	}
}
