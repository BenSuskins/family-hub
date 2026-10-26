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

func validateDisplayCommutePreferences(preferences models.DisplayCommutePreferences) string {
	if preferences.TrainCount < 1 || preferences.TrainCount > 10 {
		return "train count must be from 1 to 10"
	}
	for _, schedule := range []models.DisplayCommuteSchedule{preferences.ToWork, preferences.ToHome} {
		if len(schedule.Weekdays) == 0 || len(schedule.Weekdays) > 7 {
			return "choose at least one commute day"
		}
		seen := map[int]bool{}
		for _, weekday := range schedule.Weekdays {
			if weekday < 1 || weekday > 7 || seen[weekday] {
				return "commute days are invalid"
			}
			seen[weekday] = true
		}
		_, startErr := time.Parse("15:04", schedule.Start)
		_, endErr := time.Parse("15:04", schedule.End)
		if startErr != nil || endErr != nil || schedule.Start == schedule.End {
			return "commute start and end times are invalid"
		}
	}
	for _, station := range []string{preferences.HomeStationCode, preferences.WorkStationCode} {
		if station != "" && !models.IsDisplayStationCode(station) {
			return "station codes must use three letters"
		}
	}
	if (preferences.ToWork.Enabled || preferences.ToHome.Enabled) && (!models.IsDisplayStationCode(preferences.HomeStationCode) || !models.IsDisplayStationCode(preferences.WorkStationCode)) {
		return "valid home and work station codes are required to enable commute trains"
	}
	return ""
}

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

var displaySections = map[string]bool{"calendar": true, "meals": true, "chores": true, "weather": true, "departures": true, "trains": true, "bins": true, "shopping": true}

func (handler *DisplayPairingHandler) UpdateSections(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EnabledSections []string `json:"enabledSections"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	seen := make(map[string]bool, len(body.EnabledSections))
	for _, section := range body.EnabledSections {
		if !displaySections[section] || seen[section] {
			writeJSONError(w, http.StatusBadRequest, "display sections are invalid")
			return
		}
		seen[section] = true
	}
	if err := handler.repository.UpdateSections(r.Context(), chi.URLParam(r, "id"), body.EnabledSections); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "display not found")
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update display settings")
	} else {
		writeJSON(w, http.StatusOK, map[string][]string{"enabledSections": body.EnabledSections})
	}
}

func (handler *DisplayPairingHandler) UpdateName(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if len(name) == 0 || len(name) > 60 {
		writeJSONError(w, http.StatusBadRequest, "display name must be between 1 and 60 bytes")
		return
	}
	if err := handler.repository.UpdateDetails(r.Context(), chi.URLParam(r, "id"), name); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "display not found")
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update display name")
	} else {
		writeJSON(w, http.StatusOK, map[string]string{"name": name})
	}
}

func (handler *DisplayPairingHandler) UpdateCommutePreferences(w http.ResponseWriter, r *http.Request) {
	var preferences models.DisplayCommutePreferences
	if !decodeJSONBody(w, r, &preferences) {
		return
	}
	if message := validateDisplayCommutePreferences(preferences); message != "" {
		writeJSONError(w, http.StatusBadRequest, message)
		return
	}
	preferences.HomeStationCode = models.NormalizeDisplayStationCode(preferences.HomeStationCode)
	preferences.WorkStationCode = models.NormalizeDisplayStationCode(preferences.WorkStationCode)
	if err := handler.repository.UpdateCommutePreferences(r.Context(), chi.URLParam(r, "id"), preferences); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "display not found")
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update display commute settings")
	} else {
		writeJSON(w, http.StatusOK, preferences)
	}
}

func (handler *DisplayPairingHandler) CommutePreferences(w http.ResponseWriter, r *http.Request) {
	display, err := handler.repository.FindByTokenID(r.Context(), middleware.GetDisplayTokenID(r.Context()))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load display commute settings")
		return
	}
	writeJSON(w, http.StatusOK, display.CommutePreferences)
}

func (handler *DisplayPairingHandler) AdminCommutePreferences(w http.ResponseWriter, r *http.Request) {
	display, err := handler.repository.FindByID(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "display not found")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load display commute settings")
		return
	}
	writeJSON(w, http.StatusOK, display.CommutePreferences)
}

func (handler *DisplayPairingHandler) Settings(w http.ResponseWriter, r *http.Request) {
	display, err := handler.repository.FindByTokenID(r.Context(), middleware.GetDisplayTokenID(r.Context()))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load display settings")
		return
	}
	writeJSON(w, http.StatusOK, display)
}
