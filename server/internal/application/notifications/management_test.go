package notifications_test

import (
	"context"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/models"
)

type managementStoreFake struct{ device models.NotificationDevice }

func (s *managementStoreFake) Register(_ context.Context, device models.NotificationDevice) error {
	s.device = device
	return nil
}
func (s *managementStoreFake) Remove(context.Context, string, string) error { return nil }

type settingsWriterFake struct{ value string }

func (s *settingsWriterFake) Set(_ context.Context, _, value string) error {
	s.value = value
	return nil
}
func TestRegisterDeviceStoresHashedCredential(t *testing.T) {
	store := &managementStoreFake{}
	service := app.NewReminderManagement(reminderConfigFake{}, &settingsWriterFake{}, store, func(string) string { return "hashed" })
	token := "token"
	if err := service.Register(context.Background(), models.NotificationDevice{ID: "device", UserID: "user"}, &token); err != nil {
		t.Fatal(err)
	}
	if store.device.APITokenHash == nil || *store.device.APITokenHash != "hashed" {
		t.Fatalf("device=%+v", store.device)
	}
}
