package household_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/household"
	domain "github.com/bensuskins/family-hub/internal/domain/household"
)

type settingsFake struct {
	values  map[string]string
	failure error
	writes  int
}

func (fake *settingsFake) Load(_ context.Context, key string) (string, bool, error) {
	value, found := fake.values[key]
	return value, found, fake.failure
}
func (fake *settingsFake) Set(_ context.Context, key, value string) error {
	fake.writes++
	fake.values[key] = value
	return fake.failure
}

func (fake *settingsFake) LoadHome(ctx context.Context, defaults domain.Home) (domain.Home, error) {
	value, found, err := fake.Load(ctx, "home_info")
	if err != nil || !found {
		return defaults, err
	}
	var record struct {
		Latitude             *float64 `json:"latitude"`
		Longitude            *float64 `json:"longitude"`
		Timezone             string   `json:"timezone"`
		ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
	}
	record.Latitude = defaults.Latitude
	record.Longitude = defaults.Longitude
	record.Timezone = defaults.Timezone
	record.ArrivalMarginMinutes = defaults.ArrivalMarginMinutes
	err = json.Unmarshal([]byte(value), &record)
	return domain.Home{Latitude: record.Latitude, Longitude: record.Longitude, Timezone: record.Timezone, ArrivalMarginMinutes: record.ArrivalMarginMinutes}, err
}
func (fake *settingsFake) SaveHome(ctx context.Context, home domain.Home) error {
	value, err := json.Marshal(struct {
		Latitude             *float64 `json:"latitude"`
		Longitude            *float64 `json:"longitude"`
		Timezone             string   `json:"timezone"`
		ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
	}{home.Latitude, home.Longitude, home.Timezone, home.ArrivalMarginMinutes})
	if err != nil {
		return err
	}
	return fake.Set(ctx, "home_info", string(value))
}
func TestHomeDefaultsAndValidation(t *testing.T) {
	store := &settingsFake{values: map[string]string{}}
	service := app.New(store, store)
	home, err := service.Home(context.Background())
	if err != nil || home.Timezone != "Europe/London" || home.ArrivalMarginMinutes != 10 {
		t.Fatalf("default %+v: %v", home, err)
	}
	if service.FamilyName(context.Background()) != "Family" {
		t.Fatal("missing default family name")
	}
	if _, err := service.SaveHome(context.Background(), home); !errors.Is(err, domain.ErrInvalidLocation) {
		t.Fatalf("invalid home: %v", err)
	}
	if store.writes != 0 {
		t.Fatal("invalid home reached storage")
	}
	latitude, longitude := 51.5, -0.1
	home.Latitude = &latitude
	home.Longitude = &longitude
	if _, err := service.SaveHome(context.Background(), home); err != nil {
		t.Fatal(err)
	}
	latitude = 0
	stored, err := service.Home(context.Background())
	if err != nil || *stored.Latitude != 51.5 {
		t.Fatalf("home snapshot changed: %+v %v", stored, err)
	}
}

func TestHomeSnapshotsAreDefensive(t *testing.T) {
	latitude, longitude := 51.5, -0.1
	fake := &homeFake{home: domain.Home{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London", ArrivalMarginMinutes: 10}}
	service := app.New(&settingsFake{values: map[string]string{}}, fake)
	home, err := service.Home(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*home.Latitude = 0
	if *fake.home.Latitude != 51.5 {
		t.Fatal("home read exposed stored latitude")
	}
	home.Latitude = &latitude
	saved, err := service.SaveHome(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	*saved.Longitude = 1
	if *fake.home.Longitude != -0.1 {
		t.Fatal("home write exposed stored longitude")
	}
}

type homeFake struct{ home domain.Home }

func (fake *homeFake) LoadHome(context.Context, domain.Home) (domain.Home, error) {
	return fake.home, nil
}
func (fake *homeFake) SaveHome(_ context.Context, home domain.Home) error {
	fake.home = home
	return nil
}
