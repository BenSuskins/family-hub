package contracttests

import (
	"context"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/household"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type householdSettingsFake struct{ values map[string]string }

func (fake *householdSettingsFake) Load(ctx context.Context, key string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	value, found := fake.values[key]
	return value, found, nil
}
func (fake *householdSettingsFake) Set(ctx context.Context, key, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fake.values[key] = value
	return nil
}

func TestHouseholdSettingsStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) app.Settings{
		"fake":   func(*testing.T) app.Settings { return &householdSettingsFake{values: map[string]string{}} },
		"sqlite": func(t *testing.T) app.Settings { return repository.NewSettingsRepository(testutil.NewTestDatabase(t)) },
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			ctx := context.Background()
			value, found, err := store.Load(ctx, "contract_missing")
			if err != nil || found || value != "" {
				t.Fatalf("missing: %q %v %v", value, found, err)
			}
			for _, expected := range []string{"first", "", "replacement"} {
				if err := store.Set(ctx, "contract_setting", expected); err != nil {
					t.Fatal(err)
				}
				value, found, err = store.Load(ctx, "contract_setting")
				if err != nil || !found || value != expected {
					t.Fatalf("load: %q %v %v; want %q", value, found, err, expected)
				}
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, _, err := store.Load(cancelled, "contract_setting"); err == nil {
				t.Fatal("cancelled load succeeded")
			}
			if err := store.Set(cancelled, "contract_setting", "changed"); err == nil {
				t.Fatal("cancelled write succeeded")
			}
			value, found, err = store.Load(ctx, "contract_setting")
			if err != nil || !found || value != "replacement" {
				t.Fatalf("cancelled write changed value: %q %v %v", value, found, err)
			}
		})
	}
}
