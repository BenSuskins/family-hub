package scratchpad_test

import (
	"context"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/scratchpad"
	domain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
)

type scratchpadFake struct{ value domain.Snapshot }

func (fake *scratchpadFake) Get(context.Context) (domain.Snapshot, error) { return fake.value, nil }
func (fake *scratchpadFake) Save(_ context.Context, strokes []domain.Stroke, _ string) (domain.Snapshot, error) {
	fake.value.Strokes = strokes
	return fake.value, nil
}
func TestScratchpadSnapshotsDoNotShareMutablePoints(t *testing.T) {
	store := &scratchpadFake{}
	service := app.New(store)
	input := []domain.Stroke{{Color: "#3c654c", Points: []domain.Point{{X: 0.2, Y: 0.3}}}}
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
	if loaded.Strokes[0].Points[0] != (domain.Point{X: 0.2, Y: 0.3}) {
		t.Fatalf("stored value changed: %+v", loaded)
	}
	loaded.Strokes[0].Points[0].X = 0.9
	again, _ := service.Get(context.Background())
	if again.Strokes[0].Points[0].X != 0.2 {
		t.Fatal("read snapshot changed stored value")
	}
}
