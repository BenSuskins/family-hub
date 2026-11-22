package contracttests

import (
	"context"
	"encoding/json"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/household"
	domain "github.com/bensuskins/family-hub/internal/domain/household"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type homeContractRecord struct {
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	Timezone             string   `json:"timezone"`
	ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
}
type homeContractFake struct {
	value string
	found bool
}

func (fake *homeContractFake) LoadHome(ctx context.Context, defaults domain.Home) (domain.Home, error) {
	if err := ctx.Err(); err != nil {
		return defaults, err
	}
	if !fake.found {
		return defaults, nil
	}
	record := homeContractRecord{defaults.Latitude, defaults.Longitude, defaults.Timezone, defaults.ArrivalMarginMinutes}
	err := json.Unmarshal([]byte(fake.value), &record)
	return domain.Home{Latitude: record.Latitude, Longitude: record.Longitude, Timezone: record.Timezone, ArrivalMarginMinutes: record.ArrivalMarginMinutes}, err
}
func (fake *homeContractFake) SaveHome(ctx context.Context, home domain.Home) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := json.Marshal(homeContractRecord{home.Latitude, home.Longitude, home.Timezone, home.ArrivalMarginMinutes})
	if err != nil {
		return err
	}
	fake.value = string(data)
	fake.found = true
	return nil
}

type homeFixture struct {
	store app.HomeStore
	put   func(string) error
	raw   func() (string, error)
}

func TestHomeStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) homeFixture{
		"fake": func(t *testing.T) homeFixture {
			fake := &homeContractFake{}
			return homeFixture{fake, func(value string) error { fake.value = value; fake.found = true; return nil }, func() (string, error) { return fake.value, nil }}
		},
		"sqlite": func(t *testing.T) homeFixture {
			settings := repository.NewSettingsRepository(testutil.NewTestDatabase(t))
			return homeFixture{settings, func(value string) error { return settings.Set(context.Background(), "home_info", value) }, func() (string, error) { return settings.Get(context.Background(), "home_info") }}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			fixture := factory(t)
			ctx := context.Background()
			defaults := domain.Home{Timezone: "Europe/London", ArrivalMarginMinutes: 10}
			home, err := fixture.store.LoadHome(ctx, defaults)
			if err != nil || home.Timezone != "Europe/London" || home.ArrivalMarginMinutes != 10 {
				t.Fatalf("defaults=%+v %v", home, err)
			}
			if err := fixture.put(`{"latitude":51.5}`); err != nil {
				t.Fatal(err)
			}
			home, err = fixture.store.LoadHome(ctx, defaults)
			if err != nil || home.Latitude == nil || *home.Latitude != 51.5 || home.Timezone != "Europe/London" {
				t.Fatalf("partial=%+v %v", home, err)
			}
			if err := fixture.put(`{"latitude":51.5,"longitude":"invalid","timezone":"UTC"}`); err != nil {
				t.Fatal(err)
			}
			home, err = fixture.store.LoadHome(ctx, defaults)
			if err == nil || home.Latitude == nil || *home.Latitude != 51.5 || home.Timezone != "UTC" {
				t.Fatalf("legacy invalid=%+v %v", home, err)
			}
			latitude, longitude := 50.0, -1.0
			home = domain.Home{Latitude: &latitude, Longitude: &longitude, Timezone: "UTC", ArrivalMarginMinutes: 9}
			if err := fixture.store.SaveHome(ctx, home); err != nil {
				t.Fatal(err)
			}
			raw, err := fixture.raw()
			if err != nil || raw != `{"latitude":50,"longitude":-1,"timezone":"UTC","arrivalMarginMinutes":9}` {
				t.Fatalf("record=%s %v", raw, err)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if err := fixture.store.SaveHome(cancelled, home); err == nil {
				t.Fatal("cancelled save succeeded")
			}
		})
	}
}
