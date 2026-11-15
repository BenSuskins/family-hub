package contracttests

import (
	"context"
	"reflect"
	"testing"

	"github.com/bensuskins/family-hub/internal/application/bins"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type binScheduleFake struct{ schedule bins.Schedule }

func (fake *binScheduleFake) Load(ctx context.Context) (bins.Schedule, error) {
	if err := ctx.Err(); err != nil {
		return bins.Schedule{}, err
	}
	result := fake.schedule
	result.Collections = append([]bins.Collection{}, fake.schedule.Collections...)
	return result, nil
}
func (fake *binScheduleFake) Save(ctx context.Context, schedule bins.Schedule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fake.schedule = schedule
	fake.schedule.Collections = append([]bins.Collection(nil), schedule.Collections...)
	return nil
}
func TestBinScheduleStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) bins.Store{"fake": func(*testing.T) bins.Store { return &binScheduleFake{} }, "sqlite": func(t *testing.T) bins.Store {
		return services.NewBinScheduleAdapter(repository.NewSettingsRepository(testutil.NewTestDatabase(t)))
	}}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := factory(t)
			loaded, err := store.Load(ctx)
			if err != nil || loaded.Enabled || loaded.Collections == nil || len(loaded.Collections) != 0 {
				t.Fatalf("initial: %+v %v", loaded, err)
			}
			original := bins.Schedule{Collections: []bins.Collection{{Date: "2099-02-03", Type: "food_caddy", Name: "Food caddy"}}, Enabled: true, UploadedAt: "2099-01-01T00:00:00Z"}
			if err := store.Save(ctx, original); err != nil {
				t.Fatal(err)
			}
			original.Collections[0].Name = "changed"
			loaded, err = store.Load(ctx)
			if err != nil || loaded.Collections[0].Name != "Food caddy" || !loaded.Enabled {
				t.Fatalf("load: %+v %v", loaded, err)
			}
			loaded.Collections[0].Name = "mutated"
			again, err := store.Load(ctx)
			if err != nil || again.Collections[0].Name != "Food caddy" {
				t.Fatalf("copy: %+v %v", again, err)
			}
			expected := bins.Schedule{Collections: []bins.Collection{}, UploadedAt: "legacy"}
			if err := store.Save(ctx, bins.Schedule{UploadedAt: "legacy"}); err != nil {
				t.Fatal(err)
			}
			actual, err := store.Load(ctx)
			if err != nil || !reflect.DeepEqual(actual, expected) {
				t.Fatalf("empty: %+v %v", actual, err)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.Load(cancelled); err == nil {
				t.Fatal("cancelled load succeeded")
			}
			if err := store.Save(cancelled, original); err == nil {
				t.Fatal("cancelled write succeeded")
			}
			actual, err = store.Load(ctx)
			if err != nil || !reflect.DeepEqual(actual, expected) {
				t.Fatalf("cancelled write changed value: %+v %v", actual, err)
			}
		})
	}
}
