package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bensuskins/family-hub/internal/application/bins"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type BinScheduleAdapter struct{ settings repository.SettingsRepository }

func NewBinScheduleAdapter(settings repository.SettingsRepository) *BinScheduleAdapter {
	return &BinScheduleAdapter{settings: settings}
}
func (adapter *BinScheduleAdapter) Load(ctx context.Context) (bins.Schedule, error) {
	schedule, err := LoadBinSchedule(ctx, adapter.settings)
	return applicationBinSchedule(schedule), err
}
func (adapter *BinScheduleAdapter) Save(ctx context.Context, schedule bins.Schedule) error {
	record := models.BinSchedule{Collections: modelBinCollections(schedule.Collections), Enabled: schedule.Enabled, UploadedAt: schedule.UploadedAt}
	value, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return adapter.settings.Set(ctx, binScheduleSetting, string(value))
}

type BinCalendarParser struct{}

func (BinCalendarParser) Parse(data string, now time.Time) ([]bins.Collection, error) {
	collections, err := ParseBinSchedule(data, now)
	if err != nil {
		return nil, err
	}
	return applicationBinCollections(collections), nil
}
func applicationBinSchedule(schedule models.BinSchedule) bins.Schedule {
	return bins.Schedule{Collections: applicationBinCollections(schedule.Collections), Enabled: schedule.Enabled, UploadedAt: schedule.UploadedAt}
}
func applicationBinCollections(collections []models.BinCollection) []bins.Collection {
	if collections == nil {
		return nil
	}
	result := make([]bins.Collection, len(collections))
	for index, item := range collections {
		result[index] = bins.Collection{Date: item.Date, Type: item.Type, Name: item.Name}
	}
	return result
}
func modelBinCollections(collections []bins.Collection) []models.BinCollection {
	if collections == nil {
		return nil
	}
	result := make([]models.BinCollection, len(collections))
	for index, item := range collections {
		result[index] = models.BinCollection{Date: item.Date, Type: item.Type, Name: item.Name}
	}
	return result
}
