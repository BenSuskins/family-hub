package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

func HouseholdLocation(ctx context.Context, settings repository.SettingsRepository) (*time.Location, error) {
	name := "Europe/London"
	value, err := settings.Get(ctx, "home_info")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		var home models.HomeInfo
		if err := json.Unmarshal([]byte(value), &home); err != nil {
			return nil, err
		}
		if home.Timezone != "" {
			name = home.Timezone
		}
	}
	return time.LoadLocation(name)
}
