package household_test

import (
	"errors"
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/household"
	"github.com/bensuskins/family-hub/internal/models"
)

func TestHomeValidation(t *testing.T) {
	latitude, longitude := 51.5, -0.1
	valid := models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London", ArrivalMarginMinutes: 10}
	if err := household.ValidateHome(valid); err != nil {
		t.Fatal(err)
	}
	invalid := valid
	invalid.Timezone = "Local"
	if err := household.ValidateHome(invalid); !errors.Is(err, household.ErrInvalidTimezone) {
		t.Fatalf("timezone error: %v", err)
	}
	invalid = valid
	invalid.Latitude = nil
	if err := household.ValidateHome(invalid); !errors.Is(err, household.ErrInvalidLocation) {
		t.Fatalf("location error: %v", err)
	}
}
