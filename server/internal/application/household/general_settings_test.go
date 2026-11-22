package household_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/household"
)

func TestGeneralSettingsDefaultsAndLegacyValues(t *testing.T) {
	store := &settingsFake{values: map[string]string{}}
	service := app.New(store, nil)
	settings, err := service.GeneralSettings(context.Background())
	if err != nil || settings.FamilyName != "Family" || settings.MealWeekStart != "friday" {
		t.Fatalf("defaults: %+v %v", settings, err)
	}
	store.values["family_name"] = ""
	store.values["meal_week_start"] = "Friday"
	settings, err = service.GeneralSettings(context.Background())
	if err != nil || settings.FamilyName != "" || settings.MealWeekStart != "friday" {
		t.Fatalf("legacy: %+v %v", settings, err)
	}
}

func TestUpdateGeneralSettingsValidatesBeforeWriting(t *testing.T) {
	store := &settingsFake{values: map[string]string{"family_name": "Original"}}
	service := app.New(store, nil)
	err := service.UpdateGeneralSettings(context.Background(), map[string]string{"family_name": "Changed", "meal_week_start": "Friday"})
	if !errors.Is(err, app.ErrInvalidGeneralSetting) || store.writes != 0 {
		t.Fatalf("invalid update: %v writes %d", err, store.writes)
	}
	err = service.UpdateGeneralSettings(context.Background(), map[string]string{"family_name": "Changed", "meal_week_start": "monday"})
	if err != nil || store.values["family_name"] != "Changed" || store.values["meal_week_start"] != "monday" {
		t.Fatalf("update: %v %+v", err, store.values)
	}
}

func TestGeneralSettingsReportsWhichLoadFailed(t *testing.T) {
	store := &settingsFake{values: map[string]string{}, failure: errors.New("read failed")}
	_, err := app.New(store, nil).GeneralSettings(context.Background())
	if !errors.Is(err, app.ErrFamilyNameSetting) {
		t.Fatalf("load: %v", err)
	}
}

type failingGeneralSettings struct {
	failKey string
	failure error
}

func (fake failingGeneralSettings) Load(_ context.Context, key string) (string, bool, error) {
	if key == fake.failKey {
		return "", false, fake.failure
	}
	return "name", true, nil
}
func (fake failingGeneralSettings) Set(context.Context, string, string) error { return fake.failure }

func TestGeneralSettingsMealReadError(t *testing.T) {
	cause := errors.New("read failed")
	_, err := app.New(failingGeneralSettings{failKey: app.MealWeekStartKey, failure: cause}, nil).GeneralSettings(context.Background())
	if !errors.Is(err, app.ErrMealWeekSetting) || !errors.Is(err, cause) {
		t.Fatalf("meal load error: %v", err)
	}
}

func TestUpdateGeneralSettingsReturnsWriteError(t *testing.T) {
	cause := errors.New("write failed")
	err := app.New(failingGeneralSettings{failure: cause}, nil).UpdateGeneralSettings(context.Background(), map[string]string{app.FamilyNameKey: "Name"})
	if !errors.Is(err, cause) {
		t.Fatalf("write error: %v", err)
	}
}
