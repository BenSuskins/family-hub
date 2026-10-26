package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestDisplayTokenCannotUseMemberRoutes(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	ctx := context.Background()
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "display-auth", Email: "display-auth@example.com", Name: "Display", Role: models.RoleMember})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	tokens := repository.NewAPITokenRepository(database)
	if _, err := tokens.Create(ctx, models.APIToken{Name: "display", TokenHash: repository.HashToken("display-token"), Scope: models.TokenScopeDisplay, CreatedByUserID: user.ID}); err != nil {
		t.Fatalf("creating display token: %v", err)
	}
	if _, err := database.ExecContext(ctx, `DELETE FROM users WHERE id=?`, user.ID); err != nil {
		t.Fatalf("deleting pairing admin: %v", err)
	}
	displayHandler := middleware.RequireDisplay(tokens)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	memberHandler := middleware.RequireUser(tokens, users)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, test := range []struct {
		name    string
		handler http.Handler
		want    int
	}{{"display route", displayHandler, http.StatusOK}, {"member route", memberHandler, http.StatusUnauthorized}} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			request.Header.Set("Authorization", "Bearer display-token")
			response := httptest.NewRecorder()
			test.handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("want %d, got %d", test.want, response.Code)
			}
		})
	}
}
