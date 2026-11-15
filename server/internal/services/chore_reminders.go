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

func LoadChoreReminders(ctx context.Context, settings repository.SettingsRepository) (models.ChoreReminders, error) {
	result := models.ChoreReminders{Times: [2]string{"10:00", "16:30"}, Timezone: "Europe/London"}
	value, err := settings.Get(ctx, "chore_reminders")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil {
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return result, err
		}
	}
	home, err := settings.Get(ctx, "home_info")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil {
		var info models.HomeInfo
		if err := json.Unmarshal([]byte(home), &info); err != nil {
			return result, err
		}
		if info.Timezone != "" {
			result.Timezone = info.Timezone
		}
	}
	if _, err := time.LoadLocation(result.Timezone); err != nil {
		return result, err
	}
	return result, nil
}

func ChoreTimezone(settings repository.SettingsRepository) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		reminders, err := LoadChoreReminders(ctx, settings)
		return reminders.Timezone, err
	}
}
