package meals_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/meals"
)

type mealFake struct {
	saved                   app.Meal
	writeError, errorOnRead error
}

func (fake *mealFake) Upsert(_ context.Context, meal app.Meal) error {
	fake.saved = meal
	return fake.writeError
}
func (fake *mealFake) FindByDateAndType(context.Context, string, app.Type) (app.Meal, error) {
	return fake.saved, fake.errorOnRead
}
func (fake *mealFake) FindAll(context.Context, app.Filter) ([]app.Meal, error) {
	return nil, nil
}
func (fake *mealFake) FindByDate(context.Context, string) ([]app.Meal, error) { return nil, nil }
func (fake *mealFake) Delete(context.Context, string, app.Type) error         { return nil }
func (fake *mealFake) ClearRecipeID(context.Context, string) error            { return nil }
func TestSaveDistinguishesWriteAndReadFailures(t *testing.T) {
	store := &mealFake{}
	service := app.New(store)
	meal := app.Meal{Date: "2026-11-07", MealType: app.Dinner, Name: "Soup"}
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

func TestSaveCopiesRecipeID(t *testing.T) {
	store := &mealFake{}
	service := app.New(store)
	recipeID := "recipe"
	meal := app.Meal{Date: "2026-11-07", MealType: app.Dinner, RecipeID: &recipeID}

	saved, err := service.Save(context.Background(), meal)
	if err != nil {
		t.Fatal(err)
	}
	recipeID = "input changed"
	if *store.saved.RecipeID != "recipe" || *saved.RecipeID != "recipe" {
		t.Fatal("changing the input changed the stored or returned meal")
	}
	*saved.RecipeID = "result changed"
	if *store.saved.RecipeID != "recipe" {
		t.Fatal("changing the result changed the stored meal")
	}
}
