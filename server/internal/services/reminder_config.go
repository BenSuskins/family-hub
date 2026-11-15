package services

import (
	"context"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type ReminderConfig struct{ settings repository.SettingsRepository }

func NewReminderConfig(settings repository.SettingsRepository) *ReminderConfig {
	return &ReminderConfig{settings: settings}
}
func (config *ReminderConfig) ChoreSettings(ctx context.Context) (models.ChoreReminders, error) {
	return LoadChoreReminders(ctx, config.settings)
}
func (config *ReminderConfig) BinSchedule(ctx context.Context) (models.BinSchedule, error) {
	return LoadBinSchedule(ctx, config.settings)
}
func (config *ReminderConfig) HouseholdLocation(ctx context.Context) (*time.Location, error) {
	return HouseholdLocation(ctx, config.settings)
}
