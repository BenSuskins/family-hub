package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestLoginPage_DevAutoLogin_WhenOIDCNotConfigured(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)

	authService, err := services.NewAuthService(
		context.Background(),
		config.Config{SessionSecret: "test-secret"},
		userRepo,
	)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}

	handler := NewAuthHandler(authService)

	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	recorder := httptest.NewRecorder()
	handler.LoginPage(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Errorf("expected 302, got %d\nbody: %s", recorder.Code, recorder.Body.String())
	}

	location := recorder.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %q", location)
	}

	var sessionCookie string
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session" {
			sessionCookie = cookie.Value
			break
		}
	}
	if sessionCookie == "" {
		t.Error("expected session cookie to be set")
	}

	// Verify the session cookie is a valid, decodable session for the dev user
	sessionRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	sessionRequest.AddCookie(&http.Cookie{Name: "session", Value: sessionCookie})
	session, err := authService.GetSession(sessionRequest)
	if err != nil {
		t.Fatalf("session cookie not decodable: %v", err)
	}
	if session.UserID == "" {
		t.Error("expected non-empty UserID in session")
	}
}
