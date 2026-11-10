package scratchpad

import (
	"errors"
	"math"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrTooManyStrokes = errors.New("scratchpad has too many strokes")
	ErrInvalidStroke  = errors.New("scratchpad drawing contains an invalid stroke")
	ErrInvalidPoint   = errors.New("scratchpad point is outside the drawing area")
)

func Validate(strokes []models.ScratchpadStroke) error {
	if len(strokes) > 1000 {
		return ErrTooManyStrokes
	}
	for _, stroke := range strokes {
		switch stroke.Color {
		case "#3c654c", "#8b503c", "#b88642", "#343c31", "#5d7ea2", "#9d3c34":
		default:
			return ErrInvalidStroke
		}
		if len(stroke.Points) < 1 || len(stroke.Points) > 2000 {
			return ErrInvalidStroke
		}
		for _, point := range stroke.Points {
			if math.IsNaN(point.X) || math.IsNaN(point.Y) || math.IsInf(point.X, 0) || math.IsInf(point.Y, 0) || point.X < 0 || point.X > 1 || point.Y < 0 || point.Y > 1 {
				return ErrInvalidPoint
			}
		}
	}
	return nil
}
