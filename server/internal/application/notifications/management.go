package notifications

import (
	"context"
	"encoding/json"

	"github.com/bensuskins/family-hub/internal/models"
)

type ReminderSettingsWriter interface {
	Set(context.Context, string, string) error
}
type DeviceRegistry interface {
	Register(context.Context, models.NotificationDevice) error
	Remove(context.Context, string, string) error
}
type ReminderManagement struct {
	config   ChoreConfig
	settings ReminderSettingsWriter
	devices  DeviceRegistry
	hash     func(string) string
}

func NewReminderManagement(config ChoreConfig, settings ReminderSettingsWriter, devices DeviceRegistry, hash func(string) string) *ReminderManagement {
	return &ReminderManagement{config: config, settings: settings, devices: devices, hash: hash}
}
func (service *ReminderManagement) Settings(ctx context.Context) (models.ChoreReminders, error) {
	return service.config.ChoreSettings(ctx)
}
func (service *ReminderManagement) SaveSettings(ctx context.Context, enabled bool, times []string) error {
	body := struct {
		Enabled bool     `json:"enabled"`
		Times   []string `json:"times"`
	}{Enabled: enabled, Times: times}
	value, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return service.settings.Set(ctx, "chore_reminders", string(value))
}
func (service *ReminderManagement) Register(ctx context.Context, device models.NotificationDevice, bearer *string) error {
	if bearer != nil {
		hash := service.hash(*bearer)
		device.APITokenHash = &hash
	}
	return service.devices.Register(ctx, device)
}
func (service *ReminderManagement) Remove(ctx context.Context, id, user string) error {
	return service.devices.Remove(ctx, id, user)
}
