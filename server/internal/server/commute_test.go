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
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCommuteRouteRequiresSessionOrBearer(t *testing.T) {
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
	configuration := config.Config{SessionSecret: strings.Repeat("s", 32), HuxleyBaseURL: provider.URL, HuxleyAccessToken: "test-provider-token"}
	auth, err := services.NewAuthService(context.Background(), configuration, users)
	if err != nil {
		t.Fatal(err)
	}
	server := New(database, configuration, auth)
	session := httptest.NewRecorder()
	if err := auth.SetSession(session, user.ID); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		bearer   string
		cookie   bool
		expected int
	}{
		{"no credentials", "", false, http.StatusFound},
		{"invalid bearer", "invalid-test-token", false, http.StatusUnauthorized},
		{"valid bearer", "valid-test-token", false, http.StatusOK},
		{"valid session", "", true, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/commute/trains", nil)
			if test.bearer != "" {
				request.Header.Set("Authorization", "Bearer "+test.bearer)
			}
			if test.cookie {
				request.AddCookie(session.Result().Cookies()[0])
			}
			recorder := httptest.NewRecorder()
			server.router.ServeHTTP(recorder, request)
			if recorder.Code != test.expected {
				t.Fatalf("expected %d, got %d: %s", test.expected, recorder.Code, recorder.Body.String())
			}
		})
	}
}
