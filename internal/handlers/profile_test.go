package handlers

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func setupProfileHandler(t *testing.T) (*ProfileHandler, models.User, repository.UserRepository) {
	t.Helper()
	database := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(database)
	user, err := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-1",
		Email:       "alice@test.com",
		Name:        "Alice",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	return NewProfileHandler(userRepo), user, userRepo
}

func multipartUpload(t *testing.T, fieldName, fileName string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("creating form file: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		t.Fatalf("copying file content: %v", err)
	}
	writer.Close()
	return body, writer.FormDataContentType()
}

func TestProfileHandler_Upload_StoresDataURI(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	// 1x1 pixel PNG (minimal valid image)
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc,
		0x33, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}

	body, contentType := multipartUpload(t, "avatar", "test.png", pngBytes)

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Upload(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}

	avatarData, err := userRepo.FindAvatarData(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("FindAvatarData: %v", err)
	}
	if !strings.HasPrefix(avatarData, "data:") {
		t.Errorf("expected data URI, got %q", avatarData[:min(len(avatarData), 30)])
	}
}

func TestProfileHandler_Upload_RejectsTooLarge(t *testing.T) {
	handler, user, _ := setupProfileHandler(t)

	// Build a 2MB payload (exceeds 1MB limit)
	largeContent := make([]byte, 2*1024*1024)
	body, contentType := multipartUpload(t, "avatar", "big.jpg", largeContent)

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar", body)
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Upload(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProfileHandler_Remove_ClearsAvatar(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	userRepo.UpdateAvatar(context.Background(), user.ID, "data:image/png;base64,abc=")

	req := httptest.NewRequest(http.MethodPost, "/profile/avatar/delete", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))

	w := httptest.NewRecorder()
	handler.Remove(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}

	avatarData, _ := userRepo.FindAvatarData(context.Background(), user.ID)
	if avatarData != "" {
		t.Errorf("expected empty avatar_data after remove, got %q", avatarData)
	}
}

func TestProfileHandler_Serve_ReturnsImageBytes(t *testing.T) {
	handler, user, userRepo := setupProfileHandler(t)

	userRepo.UpdateAvatar(context.Background(), user.ID, "data:image/png;base64,iVBORw0KGgo=")

	router := chi.NewRouter()
	router.Get("/avatar/{userID}", handler.Serve)

	req := httptest.NewRequest(http.MethodGet, "/avatar/"+user.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "image/png") {
		t.Errorf("expected image/png content-type, got %q", w.Header().Get("Content-Type"))
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body")
	}
}

func TestProfileHandler_Serve_Returns404WhenNoAvatar(t *testing.T) {
	handler, user, _ := setupProfileHandler(t)

	router := chi.NewRouter()
	router.Get("/avatar/{userID}", handler.Serve)

	req := httptest.NewRequest(http.MethodGet, "/avatar/"+user.ID, nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
