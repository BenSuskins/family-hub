package meals_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/meals"
	"github.com/bensuskins/family-hub/internal/models"
)

type mealFake struct {
	saved                   models.MealPlan
	writeError, errorOnRead error
}

func (fake *mealFake) Upsert(_ context.Context, meal models.MealPlan) error {
	fake.saved = meal
	return fake.writeError
}
func (fake *mealFake) FindByDateAndType(context.Context, string, models.MealType) (models.MealPlan, error) {
	return fake.saved, fake.errorOnRead
}
func (fake *mealFake) FindAll(context.Context, app.Filter) ([]models.MealPlan, error) {
	return nil, nil
}
func (fake *mealFake) FindByDate(context.Context, string) ([]models.MealPlan, error) { return nil, nil }
func (fake *mealFake) Delete(context.Context, string, models.MealType) error         { return nil }
func (fake *mealFake) ClearRecipeID(context.Context, string) error                   { return nil }
func TestSaveDistinguishesWriteAndReadFailures(t *testing.T) {
	store := &mealFake{}
	service := app.New(store)
	meal := models.MealPlan{Date: "2026-11-07", MealType: models.MealTypeDinner, Name: "Soup"}
	got, err := service.Save(context.Background(), meal)
	if err != nil || got.Name != "Soup" {
		t.Fatalf("saved %+v: %v", got, err)
	}
	store.writeError = errors.New("write failed")
	if _, err = service.Save(context.Background(), meal); !errors.Is(err, app.ErrSave) {
		t.Fatalf("write error: %v", err)
	}
	store.writeError = nil
	store.errorOnRead = errors.New("read failed")
	if _, err = service.Save(context.Background(), meal); !errors.Is(err, app.ErrRead) {
		t.Fatalf("read error: %v", err)
	}
}
