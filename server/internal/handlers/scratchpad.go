package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	scratchapp "github.com/bensuskins/family-hub/internal/application/scratchpad"
	scratchdomain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
	"github.com/bensuskins/family-hub/internal/middleware"
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
	writeJSON(w, http.StatusOK, scratchpadResponseFor(scratchpad))
}

func (handler *ScratchpadHandler) Save(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Strokes []scratchpadStrokeBody `json:"strokes"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxScratchpadBytes)).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "scratchpad drawing is invalid or too large")
		return
	}
	user := middleware.GetUser(r.Context())
	scratchpad, err := handler.service.Save(r.Context(), scratchpadDomainStrokes(body.Strokes), user.ID)
	if err != nil {
		if errors.Is(err, scratchdomain.ErrTooManyStrokes) || errors.Is(err, scratchdomain.ErrInvalidStroke) || errors.Is(err, scratchdomain.ErrInvalidPoint) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPIError(w, r, http.StatusInternalServerError, "could not save scratchpad", err)
		return
	}
	writeJSON(w, http.StatusOK, scratchpadResponseFor(scratchpad))
}

type scratchpadPointBody struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type scratchpadStrokeBody struct {
	Color  string                `json:"color"`
	Points []scratchpadPointBody `json:"points"`
}
type scratchpadResponse struct {
	Strokes   []scratchpadStrokeBody `json:"strokes"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

func scratchpadDomainStrokes(values []scratchpadStrokeBody) []scratchdomain.Stroke {
	if values == nil {
		return nil
	}
	strokes := make([]scratchdomain.Stroke, len(values))
	for i, value := range values {
		strokes[i].Color = value.Color
		if value.Points != nil {
			strokes[i].Points = make([]scratchdomain.Point, len(value.Points))
			for j, point := range value.Points {
				strokes[i].Points[j] = scratchdomain.Point{X: point.X, Y: point.Y}
			}
		}
	}
	return strokes
}
func scratchpadResponseFor(snapshot scratchdomain.Snapshot) scratchpadResponse {
	var strokes []scratchpadStrokeBody
	if snapshot.Strokes != nil {
		strokes = make([]scratchpadStrokeBody, len(snapshot.Strokes))
		for i, stroke := range snapshot.Strokes {
			strokes[i].Color = stroke.Color
			if stroke.Points != nil {
				strokes[i].Points = make([]scratchpadPointBody, len(stroke.Points))
				for j, point := range stroke.Points {
					strokes[i].Points[j] = scratchpadPointBody{X: point.X, Y: point.Y}
				}
			}
		}
	}
	return scratchpadResponse{Strokes: strokes, UpdatedAt: snapshot.UpdatedAt}
}
