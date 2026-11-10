package scratchpad_test

import (
	"errors"
	"math"
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/scratchpad"
	"github.com/bensuskins/family-hub/internal/models"
)

func TestDrawingValidation(t *testing.T) {
	valid := models.ScratchpadStroke{Color: "#3c654c", Points: []models.ScratchpadPoint{{X: 0, Y: 1}}}
	cases := []struct {
		name    string
		strokes []models.ScratchpadStroke
		want    error
	}{
		{"valid", []models.ScratchpadStroke{valid}, nil},
		{"too many", make([]models.ScratchpadStroke, 1001), scratchpad.ErrTooManyStrokes},
		{"invalid color", []models.ScratchpadStroke{{Color: "#ffffff", Points: valid.Points}}, scratchpad.ErrInvalidStroke},
		{"no points", []models.ScratchpadStroke{{Color: valid.Color}}, scratchpad.ErrInvalidStroke},
		{"outside canvas", []models.ScratchpadStroke{{Color: valid.Color, Points: []models.ScratchpadPoint{{X: -0.1}}}}, scratchpad.ErrInvalidPoint},
		{"not finite", []models.ScratchpadStroke{{Color: valid.Color, Points: []models.ScratchpadPoint{{X: math.NaN()}}}}, scratchpad.ErrInvalidPoint},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := scratchpad.Validate(test.strokes)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
		})
	}
}
