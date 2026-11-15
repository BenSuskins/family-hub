package household

import (
	"context"
	"errors"
	"fmt"
)

const (
	FamilyNameKey    = "family_name"
	MealWeekStartKey = "meal_week_start"
)

var (
	ErrFamilyNameSetting     = errors.New("load family name setting")
	ErrMealWeekSetting       = errors.New("load meal week setting")
	ErrInvalidGeneralSetting = errors.New("invalid general setting")
)

type GeneralSettings struct {
	FamilyName    string
	MealWeekStart string
}

func (service *Service) GeneralSettings(ctx context.Context) (GeneralSettings, error) {
	settings := GeneralSettings{FamilyName: "Family", MealWeekStart: "friday"}
	familyName, found, err := service.settings.Load(ctx, FamilyNameKey)
	if err != nil {
		return GeneralSettings{}, fmt.Errorf("%w: %w", ErrFamilyNameSetting, err)
	}
	if found {
		settings.FamilyName = familyName
	}
	mealWeekStart, found, err := service.settings.Load(ctx, MealWeekStartKey)
	if err != nil {
		return GeneralSettings{}, fmt.Errorf("%w: %w", ErrMealWeekSetting, err)
	}
	if found && ValidMealWeekStart(mealWeekStart) {
		settings.MealWeekStart = mealWeekStart
	}
	return settings, nil
}

func ValidMealWeekStart(value string) bool {
	switch value {
	case "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday":
		return true
	default:
		return false
	}
}

func (service *Service) UpdateGeneralSettings(ctx context.Context, updates map[string]string) error {
	if len(updates) == 0 {
		return ErrInvalidGeneralSetting
	}
	for key, value := range updates {
		if value == "" || (key != FamilyNameKey && key != MealWeekStartKey) || (key == MealWeekStartKey && !ValidMealWeekStart(value)) {
			return fmt.Errorf("%w: %s", ErrInvalidGeneralSetting, key)
		}
	}
	for key, value := range updates {
		if err := service.settings.Set(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}
