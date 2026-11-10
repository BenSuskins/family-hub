package handlers

import (
	"encoding/json"
	"net/http"

	"errors"
	scratchapp "github.com/bensuskins/family-hub/internal/application/scratchpad"
	scratchdomain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
)

const maxScratchpadBytes = 1 << 20

type ScratchpadHandler struct{ service *scratchapp.Service }

func NewScratchpadHandler(service *scratchapp.Service) *ScratchpadHandler {
	return &ScratchpadHandler{service: service}
}

func (handler *ScratchpadHandler) Get(w http.ResponseWriter, r *http.Request) {
	scratchpad, err := handler.service.Get(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "could not load scratchpad", err)
		return
	}
	writeJSON(w, http.StatusOK, scratchpad)
}

func (handler *ScratchpadHandler) Save(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Strokes []models.ScratchpadStroke `json:"strokes"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxScratchpadBytes)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "scratchpad drawing is invalid or too large")
		return
	}
	user := middleware.GetUser(r.Context())
	scratchpad, err := handler.service.Save(r.Context(), body.Strokes, user.ID)
	if err != nil {
		if errors.Is(err, scratchdomain.ErrTooManyStrokes) || errors.Is(err, scratchdomain.ErrInvalidStroke) || errors.Is(err, scratchdomain.ErrInvalidPoint) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPIError(w, r, http.StatusInternalServerError, "could not save scratchpad", err)
		return
	}
	writeJSON(w, http.StatusOK, scratchpad)
}
