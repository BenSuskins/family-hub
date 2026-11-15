package notifications_test

import (
	"context"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/models"
)

type reminderConfigFake struct{}

func (reminderConfigFake) ChoreSettings(context.Context) (models.ChoreReminders, error) {
	return models.ChoreReminders{Enabled: true, Times: [2]string{"10:00", "16:30"}, Timezone: "UTC"}, nil
}

type reminderChoresFake struct{}

func (reminderChoresFake) ReminderChores(context.Context) ([]models.Chore, error) {
	return []models.Chore{{ID: "chore", Name: "Bins", Status: models.ChoreStatusPending}}, nil
}

type reminderDevicesFake struct{ events []string }

func (f *reminderDevicesFake) Devices(context.Context) ([]models.NotificationDevice, error) {
	return []models.NotificationDevice{{ID: "device"}}, nil
}
func (f *reminderDevicesFake) Claim(context.Context, string, string, int, time.Time) (string, error) {
	f.events = append(f.events, "claim")
	return "delivery", nil
}
func (f *reminderDevicesFake) Finish(context.Context, string, bool) error {
	f.events = append(f.events, "finish")
	return nil
}
func (f *reminderDevicesFake) Invalidate(context.Context, models.NotificationDevice) error {
	return nil
}
func (f *reminderDevicesFake) Prune(context.Context, string) error { return nil }

type reminderSenderFake struct{ devices *reminderDevicesFake }

func (f reminderSenderFake) Send(_ context.Context, _ models.NotificationDevice, _ app.ChoreReminderSummary, _ string, _ time.Time) error {
	f.devices.events = append(f.devices.events, "send")
	return nil
}
func TestChoreReminderClaimsBeforeSending(t *testing.T) {
	devices := &reminderDevicesFake{}
	service := app.NewChoreReminderService(reminderConfigFake{}, reminderChoresFake{}, devices, reminderSenderFake{devices}, nil)
	err := service.Check(context.Background(), time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	if err != nil || len(devices.events) != 3 || devices.events[0] != "claim" || devices.events[1] != "send" || devices.events[2] != "finish" {
		t.Fatalf("events=%v err=%v", devices.events, err)
	}
}
