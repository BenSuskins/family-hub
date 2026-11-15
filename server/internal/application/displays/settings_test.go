package displays_test

import (
	"context"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/bensuskins/family-hub/internal/models"
)

type settingsStoreFake struct{ display app.Display }

func (s *settingsStoreFake) List(context.Context) ([]app.Display, error) {
	return []app.Display{s.display}, nil
}
func (s *settingsStoreFake) FindByID(context.Context, string) (app.Display, error) {
	return s.display, nil
}
func (s *settingsStoreFake) FindByTokenID(context.Context, string) (app.Display, error) {
	return s.display, nil
}
func (s *settingsStoreFake) UpdateSections(_ context.Context, _ string, sections []string) error {
	s.display.EnabledSections = sections
	return nil
}
func (s *settingsStoreFake) UpdateDetails(_ context.Context, _, name string) error {
	s.display.Name = name
	return nil
}
func (s *settingsStoreFake) UpdateCommutePreferences(_ context.Context, _ string, p models.DisplayCommutePreferences) error {
	s.display.CommutePreferences = p
	return nil
}
func (s *settingsStoreFake) Revoke(context.Context, string) error { return nil }
func TestDisplaySettingsDefensivelyCopySectionsAndSchedules(t *testing.T) {
	store := &settingsStoreFake{display: app.Display{EnabledSections: []string{"calendar"}, CommutePreferences: models.DisplayCommutePreferences{ToWork: models.DisplayCommuteSchedule{Weekdays: []int{2}}}}}
	service := app.NewSettings(store)
	item, err := service.FindByID(context.Background(), "display")
	if err != nil {
		t.Fatal(err)
	}
	item.EnabledSections[0] = "changed"
	item.CommutePreferences.ToWork.Weekdays[0] = 7
	if store.display.EnabledSections[0] != "calendar" || store.display.CommutePreferences.ToWork.Weekdays[0] != 2 {
		t.Fatal("caller changed stored settings")
	}
	sections := []string{"weather"}
	if err := service.UpdateSections(context.Background(), "display", sections); err != nil {
		t.Fatal(err)
	}
	sections[0] = "changed"
	if store.display.EnabledSections[0] != "weather" {
		t.Fatal("store retained caller's slice")
	}
}
