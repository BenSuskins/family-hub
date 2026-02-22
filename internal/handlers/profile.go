package handlers

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

const maxAvatarBytes = 1 * 1024 * 1024 // 1 MB

type ProfileHandler struct {
	userRepo repository.UserRepository
}

func NewProfileHandler(userRepo repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

func (handler *ProfileHandler) Page(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	avatarData, err := handler.userRepo.FindAvatarData(ctx, user.ID)
	if err != nil {
		slog.Error("finding avatar data", "error", err)
	}

	component := pages.Profile(pages.ProfileProps{
		User:            user,
		HasCustomAvatar: avatarData != "",
	})
	component.Render(ctx, w)
}

func (handler *ProfileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseMultipartForm(maxAvatarBytes + 1024); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Missing avatar file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxAvatarBytes+1))
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	if len(imageBytes) > maxAvatarBytes {
		http.Error(w, "Image exceeds 1 MB limit", http.StatusBadRequest)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	encoded := base64.StdEncoding.EncodeToString(imageBytes)
	dataURI := "data:" + contentType + ";base64," + encoded

	if err := handler.userRepo.UpdateAvatar(ctx, user.ID, dataURI); err != nil {
		slog.Error("updating avatar", "error", err)
		http.Error(w, "Failed to save avatar", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (handler *ProfileHandler) Remove(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := handler.userRepo.ClearAvatar(ctx, user.ID); err != nil {
		slog.Error("clearing avatar", "error", err)
		http.Error(w, "Failed to remove avatar", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (handler *ProfileHandler) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	avatarData, err := handler.userRepo.FindAvatarData(ctx, userID)
	if err != nil || avatarData == "" {
		http.NotFound(w, r)
		return
	}

	// Parse "data:<mime>;base64,<payload>"
	withoutPrefix, ok := strings.CutPrefix(avatarData, "data:")
	if !ok {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(withoutPrefix, ";base64,", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	mimeType := parts[0]
	payload := parts[1]

	imageBytes, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		slog.Error("decoding avatar base64", "error", err)
		http.Error(w, "Corrupted avatar data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	w.Write(imageBytes)
}
