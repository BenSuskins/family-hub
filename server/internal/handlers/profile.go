package handlers

import (
	"log/slog"
	"net/http"

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

	imageBytes, contentType, ok := readUploadedImage(w, r, "avatar", maxAvatarBytes)
	if !ok {
		return
	}

	dataURI := encodeDataURI(contentType, imageBytes)
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

	imageBytes, ok := decodeDataURI(avatarData)
	if !ok {
		http.NotFound(w, r)
		return
	}

	mimeType, ok := detectImageContentType(imageBytes)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Write(imageBytes)
}
