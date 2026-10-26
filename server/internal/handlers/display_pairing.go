package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const pairingChallengeLifetime = 5 * time.Minute

type DisplayPairingHandler struct {
	repository repository.DisplayPairingRepository
	tokens     repository.APITokenRepository
}

func NewDisplayPairingHandler(pairings repository.DisplayPairingRepository, tokens repository.APITokenRepository) *DisplayPairingHandler {
	return &DisplayPairingHandler{repository: pairings, tokens: tokens}
}

func (handler *DisplayPairingHandler) CreateChallenge(w http.ResponseWriter, r *http.Request) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create pairing code")
		return
	}
	secret := hex.EncodeToString(secretBytes)
	id := uuid.NewString()
	expiresAt := time.Now().UTC().Add(pairingChallengeLifetime)
	if err := handler.repository.CreateChallenge(r.Context(), id, repository.HashToken(secret), expiresAt); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create pairing request")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"challengeId": id, "pairingCode": secret, "expiresAt": expiresAt})
}

func (handler *DisplayPairingHandler) Approve(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PairingCode string `json:"pairingCode"`
		Name        string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid pairing request")
		return
	}
	name := strings.TrimSpace(request.Name)
	if len(name) < 1 || len(name) > 60 || len(request.PairingCode) != 64 {
		writeJSONError(w, http.StatusBadRequest, "display name or pairing code is invalid")
		return
	}
	user := middleware.GetUser(r.Context())
	err := handler.repository.ApproveChallenge(r.Context(), chi.URLParam(r, "id"), repository.HashToken(request.PairingCode), name, user.ID)
	if errors.Is(err, repository.ErrPairingUnavailable) {
		writeJSONError(w, http.StatusConflict, "pairing request expired or already used")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not approve pairing request")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

func (handler *DisplayPairingHandler) Claim(w http.ResponseWriter, r *http.Request) {
	var request struct {
		PairingCode string `json:"pairingCode"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil || len(request.PairingCode) != 64 {
		writeJSONError(w, http.StatusBadRequest, "invalid pairing request")
		return
	}
	rawTokenBytes := make([]byte, 32)
	if _, err := rand.Read(rawTokenBytes); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create display credential")
		return
	}
	plainToken := hex.EncodeToString(rawTokenBytes)
	token := models.APIToken{Name: "Wall display", TokenHash: repository.HashToken(plainToken), Scope: models.TokenScopeDisplay}
	err := handler.repository.ClaimChallenge(r.Context(), chi.URLParam(r, "id"), repository.HashToken(request.PairingCode), token)
	if errors.Is(err, repository.ErrPairingUnavailable) {
		writeJSONError(w, http.StatusConflict, "pairing request is not approved or has already been claimed")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not complete display pairing")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": plainToken})
}

func (handler *DisplayPairingHandler) List(w http.ResponseWriter, r *http.Request) {
	displays, err := handler.repository.List(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load paired displays")
		return
	}
	writeJSON(w, http.StatusOK, displays)
}

func (handler *DisplayPairingHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := handler.repository.Revoke(r.Context(), chi.URLParam(r, "id")); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "display not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not revoke display")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
