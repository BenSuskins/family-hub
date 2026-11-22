package scratchpad_test

import (
	"errors"
	"math"
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/scratchpad"
)

func TestDrawingValidation(t *testing.T) {
	valid := scratchpad.Stroke{Color: "#3c654c", Points: []scratchpad.Point{{X: 0, Y: 1}}}
	cases := []struct {
		name    string
		strokes []scratchpad.Stroke
		want    error
	}{
		{"valid", []scratchpad.Stroke{valid}, nil},
		{"too many", make([]scratchpad.Stroke, 1001), scratchpad.ErrTooManyStrokes},
		{"invalid color", []scratchpad.Stroke{{Color: "#ffffff", Points: valid.Points}}, scratchpad.ErrInvalidStroke},
		{"no points", []scratchpad.Stroke{{Color: valid.Color}}, scratchpad.ErrInvalidStroke},
		{"outside canvas", []scratchpad.Stroke{{Color: valid.Color, Points: []scratchpad.Point{{X: -0.1}}}}, scratchpad.ErrInvalidPoint},
		{"not finite", []scratchpad.Stroke{{Color: valid.Color, Points: []scratchpad.Point{{X: math.NaN()}}}}, scratchpad.ErrInvalidPoint},
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
