package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCommuteRouteRequiresBearerToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(context.Background(), models.User{
		OIDCSubject: "commuter", Email: "commuter@example.com", Name: "Commuter", Role: models.RoleMember,
	})
	if err != nil {
		t.Fatal(err)
	}
	tokens := repository.NewAPITokenRepository(database)
	_, err = tokens.Create(context.Background(), models.APIToken{
		Name: "commute-test", TokenHash: repository.HashToken("valid-test-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":[]}`))
	}))
	t.Cleanup(provider.Close)
	configuration := config.Config{HuxleyBaseURL: provider.URL, HuxleyAccessToken: "test-provider-token"}
	server := New(database, configuration)
	for _, test := range []struct {
		name     string
		bearer   string
		expected int
	}{
		{"no credentials", "", http.StatusUnauthorized},
		{"invalid bearer", "invalid-test-token", http.StatusUnauthorized},
		{"valid bearer", "valid-test-token", http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/commute/trains/from/KEL/to/LST", nil)
			if test.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+test.bearer)
			}
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)
			if recorder.Code != test.expected {
				t.Fatalf("expected %d, got %d: %s", test.expected, recorder.Code, recorder.Body.String())
			}
		})
	}
}
