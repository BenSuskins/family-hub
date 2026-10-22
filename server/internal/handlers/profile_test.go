package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestProfileHandler_ServeReturnsAvatarBytes(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	userRepository := repository.NewUserRepository(database)
	user, err := userRepository.Create(context.Background(), models.User{
		OIDCSubject: "avatar-subject",
		Email:       "avatar@example.com",
		Name:        "Avatar User",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	if err := userRepository.UpdateAvatar(context.Background(), user.ID, pngDataURI(t, 1, 1)); err != nil {
		t.Fatalf("storing avatar: %v", err)
	}

	router := chi.NewRouter()
	router.Get("/avatar/{userID}", NewProfileHandler(userRepository).Serve)
	request := httptest.NewRequest(http.MethodGet, "/avatar/"+user.ID, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", response.Code)
	}
	if response.Header().Get("Content-Type") != "image/png" {
		t.Errorf("got Content-Type %q, want image/png", response.Header().Get("Content-Type"))
	}
	if response.Body.Len() == 0 {
		t.Error("expected avatar bytes")
	}
}

func TestProfileHandler_ServeMissingAvatarReturnsJSON404(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	userRepository := repository.NewUserRepository(database)
	router := chi.NewRouter()
	router.Get("/avatar/{userID}", NewProfileHandler(userRepository).Serve)
	request := httptest.NewRequest(http.MethodGet, "/avatar/missing", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("got status %d, want 404", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("got Content-Type %q, want application/json", got)
	}
}
