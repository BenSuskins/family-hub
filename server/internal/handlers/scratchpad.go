package handlers

import (
	"encoding/json"
	"math"
	"net/http"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

const maxScratchpadBytes = 1 << 20

var scratchpadColors = map[string]bool{"#3c654c": true, "#8b503c": true, "#b88642": true, "#343c31": true, "#5d7ea2": true, "#9d3c34": true}

type ScratchpadHandler struct {
	repository repository.ScratchpadRepository
}

func NewScratchpadHandler(repository repository.ScratchpadRepository) *ScratchpadHandler {
	return &ScratchpadHandler{repository: repository}
}

func (handler *ScratchpadHandler) Get(w http.ResponseWriter, r *http.Request) {
	scratchpad, err := handler.repository.Get(r.Context())
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
	if len(body.Strokes) > 1000 {
		writeJSONError(w, http.StatusBadRequest, "scratchpad has too many strokes")
		return
	}
	for _, stroke := range body.Strokes {
		if !scratchpadColors[stroke.Color] || len(stroke.Points) < 1 || len(stroke.Points) > 2000 {
			writeJSONError(w, http.StatusBadRequest, "scratchpad drawing contains an invalid stroke")
			return
		}
		for _, point := range stroke.Points {
			if math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) || point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 {
				writeJSONError(w, http.StatusBadRequest, "scratchpad point is outside the drawing area")
				return
			}
		}
	}
	user := middleware.GetUser(r.Context())
	scratchpad, err := handler.repository.Save(r.Context(), body.Strokes, user.ID)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "could not save scratchpad", err)
		return
	}
	writeJSON(w, http.StatusOK, scratchpad)
}
