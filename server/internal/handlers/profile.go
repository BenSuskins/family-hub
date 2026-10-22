package handlers

import (
	"net/http"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/go-chi/chi/v5"
)

const maxAvatarBytes = 2 * 1024 * 1024 // 2 MB

type ProfileHandler struct {
	userRepo repository.UserRepository
}

func NewProfileHandler(userRepo repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

func (handler *ProfileHandler) Serve(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userID")

	avatarData, err := handler.userRepo.FindAvatarData(ctx, userID)
	if err != nil || avatarData == "" {
		writeJSONError(w, http.StatusNotFound, "avatar not found")
		return
	}

	imageBytes, ok := decodeDataURI(avatarData)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "avatar not found")
		return
	}

	mimeType, ok := detectImageContentType(imageBytes)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "avatar not found")
		return
	}

	w.Header().Set("Content-Type", mimeType)
	w.Write(imageBytes)
}
