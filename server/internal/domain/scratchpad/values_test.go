package scratchpad_test

import (
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/scratchpad"
)

func TestDrawingCopiesNestedPoints(t *testing.T) {
	original := []scratchpad.Stroke{{Color: "#3c654c", Points: []scratchpad.Point{{X: .2, Y: .3}}}}
	drawing, err := scratchpad.NewDrawing(original)
	if err != nil {
		t.Fatal(err)
	}
	original[0].Points[0].X = .9
	first := drawing.Strokes()
	if first[0].Points[0].X != .2 {
		t.Fatal("constructor shared input")
	}
	first[0].Points[0].X = .8
	if drawing.Strokes()[0].Points[0].X != .2 {
		t.Fatal("snapshot shared state")
	}
}
