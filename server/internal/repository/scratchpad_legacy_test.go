package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestScratchpadReadsLegacyDrawingWithoutRepair(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	ctx := context.Background()
	legacy := `[{"color":"#legacy","points":[{"x":-1,"y":2}]}]`
	if _, err := db.ExecContext(ctx, "UPDATE household_scratchpad SET strokes_json=? WHERE id=1", legacy); err != nil {
		t.Fatal(err)
	}
	snapshot, err := repository.NewScratchpadRepository(db).Get(ctx)
	if err != nil || len(snapshot.Strokes) != 1 || snapshot.Strokes[0].Color != "#legacy" || snapshot.Strokes[0].Points[0].X != -1 {
		t.Fatalf("legacy %+v %v", snapshot, err)
	}
	var stored string
	if err := db.QueryRowContext(ctx, "SELECT strokes_json FROM household_scratchpad WHERE id=1").Scan(&stored); err != nil || stored != legacy {
		t.Fatalf("record changed %q %v", stored, err)
	}
}
