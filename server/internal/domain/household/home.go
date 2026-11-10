package household

import (
	"errors"
	"math"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrInvalidLocation = errors.New("valid coordinates and an arrival margin from 0 to 120 are required")
	ErrInvalidTimezone = errors.New("valid timezone required")
)

func ValidateHome(home models.HomeInfo) error {
	if home.Latitude == nil || home.Longitude == nil || math.IsNaN(*home.Latitude) || math.IsNaN(*home.Longitude) || math.Abs(*home.Latitude) > 90 || math.Abs(*home.Longitude) > 180 || home.ArrivalMarginMinutes < 0 || home.ArrivalMarginMinutes > 120 {
		return ErrInvalidLocation
	}
	if _, err := time.LoadLocation(home.Timezone); err != nil || home.Timezone == "" || home.Timezone == "Local" {
		return ErrInvalidTimezone
	}
	return nil
}
