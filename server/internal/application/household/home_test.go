package household_test

import (
	"context"
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
func TestHomeDefaultsAndValidation(t *testing.T) {
	store := &settingsFake{values: map[string]string{}}
	service := app.New(store)
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
