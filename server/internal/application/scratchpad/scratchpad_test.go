package scratchpad_test

import (
	"context"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/scratchpad"
	"github.com/bensuskins/family-hub/internal/models"
)

type scratchpadFake struct{ value models.Scratchpad }

func (fake *scratchpadFake) Get(context.Context) (models.Scratchpad, error) { return fake.value, nil }
func (fake *scratchpadFake) Save(_ context.Context, strokes []models.ScratchpadStroke, _ string) (models.Scratchpad, error) {
	fake.value.Strokes = strokes
	return fake.value, nil
}
func TestScratchpadSnapshotsDoNotShareMutablePoints(t *testing.T) {
	store := &scratchpadFake{}
	service := app.New(store)
	input := []models.ScratchpadStroke{{Color: "#3c654c", Points: []models.ScratchpadPoint{{X: 0.2, Y: 0.3}}}}
	saved, err := service.Save(context.Background(), input, "member")
	if err != nil {
		t.Fatal(err)
	}
	input[0].Points[0].X = 0.9
	saved.Strokes[0].Points[0].Y = 0.9
	loaded, err := service.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Strokes[0].Points[0] != (models.ScratchpadPoint{X: 0.2, Y: 0.3}) {
		t.Fatalf("stored value changed: %+v", loaded)
	}
	loaded.Strokes[0].Points[0].X = 0.9
	again, _ := service.Get(context.Background())
	if again.Strokes[0].Points[0].X != 0.2 {
		t.Fatal("read snapshot changed stored value")
	}
}
