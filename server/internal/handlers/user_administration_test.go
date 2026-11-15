package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/users"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/go-chi/chi/v5"
)

type userHandlerFake struct {
	user              app.User
	writeErr, readErr error
	writes, reads     int
}

func (fake *userHandlerFake) List(context.Context) ([]app.User, error) {
	return []app.User{fake.user}, fake.readErr
}
func (fake *userHandlerFake) Find(context.Context, string) (app.User, error) {
	fake.reads++
	return fake.user, fake.readErr
}
func (fake *userHandlerFake) SetRole(context.Context, string, app.Role) error {
	fake.writes++
	return fake.writeErr
}
func (fake *userHandlerFake) SetAvatar(context.Context, string, string) error {
	fake.writes++
	return fake.writeErr
}
func (fake *userHandlerFake) ClearAvatar(context.Context, string) error {
	fake.writes++
	return fake.writeErr
}

func avatarRequest(t *testing.T) *http.Request {
	t.Helper()
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(imageData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/avatar", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request.WithContext(context.WithValue(request.Context(), middleware.UserContextKey, models.User{ID: "user"}))
}

func TestUserAdministrationWriteAndReloadErrors(t *testing.T) {
	failure := errors.New("storage unavailable")
	for _, operation := range []string{"promote", "demote", "avatar"} {
		for _, stage := range []string{"write", "reload"} {
			t.Run(operation+"/"+stage, func(t *testing.T) {
				fake := &userHandlerFake{}
				if stage == "write" {
					fake.writeErr = failure
				} else {
					fake.readErr = failure
				}
				handler := &APIHandler{users: app.New(fake)}
				recorder := httptest.NewRecorder()
				if operation == "avatar" {
					handler.UploadAvatar(recorder, avatarRequest(t))
				} else {
					router := chi.NewRouter()
					if operation == "promote" {
						router.Post("/users/{id}", handler.PromoteUser)
					} else {
						router.Post("/users/{id}", handler.DemoteUser)
					}
					router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/users/user", nil))
				}
				expected := "failed to reload user"
				if stage == "write" {
					expected = map[string]string{"promote": "failed to promote user", "demote": "failed to demote user", "avatar": "failed to save avatar"}[operation]
				}
				var response map[string]string
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatal(err)
				}
				if recorder.Code != http.StatusInternalServerError || response["error"] != expected {
					t.Fatalf("response: %d %v", recorder.Code, response)
				}
				wantReads := 1
				if stage == "write" {
					wantReads = 0
				}
				if fake.writes != 1 || fake.reads != wantReads {
					t.Fatalf("writes = %d, reads = %d", fake.writes, fake.reads)
				}
			})
		}
	}
}

func TestUserResponseJSON(t *testing.T) {
	when := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	fake := &userHandlerFake{user: app.User{ID: "user", OIDCSubject: "subject", Email: "user@example.com", Name: "User", AvatarURL: "/avatar/user", Role: app.RoleMember, CreatedAt: when, UpdatedAt: when}}
	handler := &APIHandler{users: app.New(fake)}
	recorder := httptest.NewRecorder()
	handler.GetUser(recorder, httptest.NewRequest(http.MethodGet, "/users/user", nil))
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	expected := map[string]any{"ID": "user", "OIDCSubject": "subject", "Email": "user@example.com", "Name": "User", "AvatarURL": "/avatar/user", "Role": "member", "CreatedAt": "2026-01-02T03:04:05Z", "UpdatedAt": "2026-01-02T03:04:05Z", "OnboardedAt": nil}
	if recorder.Code != http.StatusOK || !reflect.DeepEqual(response, expected) {
		t.Fatalf("response: %d %#v", recorder.Code, response)
	}
}
