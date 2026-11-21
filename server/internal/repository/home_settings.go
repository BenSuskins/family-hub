package repository

import (
	"context"
	"encoding/json"

	domain "github.com/bensuskins/family-hub/internal/domain/household"
)

type homeRecord struct {
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	Timezone             string   `json:"timezone"`
	ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
}

func homeRecordFor(home domain.Home) homeRecord {
	return homeRecord{home.Latitude, home.Longitude, home.Timezone, home.ArrivalMarginMinutes}
}
func (record homeRecord) home() domain.Home {
	return domain.Home{Latitude: record.Latitude, Longitude: record.Longitude, Timezone: record.Timezone, ArrivalMarginMinutes: record.ArrivalMarginMinutes}
}
func (repository *SQLiteSettingsRepository) LoadHome(ctx context.Context, defaults domain.Home) (domain.Home, error) {
	value, found, err := repository.Load(ctx, "home_info")
	if err != nil || !found {
		return defaults, err
	}
	record := homeRecordFor(defaults)
	err = json.Unmarshal([]byte(value), &record)
	return record.home(), err
}
func (repository *SQLiteSettingsRepository) SaveHome(ctx context.Context, home domain.Home) error {
	value, err := json.Marshal(homeRecordFor(home))
	if err != nil {
		return err
	}
	return repository.Set(ctx, "home_info", string(value))
}
