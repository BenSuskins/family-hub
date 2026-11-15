package recipes_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
)

type catalogueFake struct {
	recipe    app.SavedRecipe
	deleted   bool
	updateErr error
	findErr   error
}

func (f *catalogueFake) FindByID(context.Context, string) (app.SavedRecipe, error) {
	return f.recipe, f.findErr
}
func (f *catalogueFake) FindAll(context.Context) ([]app.SavedRecipe, error) {
	return []app.SavedRecipe{f.recipe}, nil
}
func (f *catalogueFake) Create(_ context.Context, recipe app.SavedRecipe) (app.SavedRecipe, error) {
	recipe.ID = "recipe"
	f.recipe = recipe
	return recipe, nil
}
func (f *catalogueFake) Update(_ context.Context, recipe app.SavedRecipe) error {
	f.recipe = recipe
	return f.updateErr
}
func (f *catalogueFake) Delete(context.Context, string) error { f.deleted = true; return nil }

type imageFake struct{ err error }

func (f imageFake) UpdateImage(context.Context, string, string) error { return f.err }
func (f imageFake) ClearImage(context.Context, string) error          { return f.err }

type mealLinksFake struct{ err error }

func (f mealLinksFake) ClearRecipeID(context.Context, string) error { return f.err }
func TestCreateRecipeKeepsSuccessWhenImageFails(t *testing.T) {
	store := &catalogueFake{}
	service := app.NewCatalogue(store, imageFake{err: errors.New("image failed")}, mealLinksFake{})
	result, err := service.Create(context.Background(), app.SavedRecipe{Title: "Soup"}, value("image"))
	if err != nil || result.Recipe.ID != "recipe" || result.Recipe.HasImage || result.ImageError == nil {
		t.Fatalf("result=%+v warning=%v err=%v", result, result.ImageError, err)
	}
}
func TestDeleteRecipeAttemptsDeleteWhenMealUnlinkFails(t *testing.T) {
	store := &catalogueFake{}
	service := app.NewCatalogue(store, imageFake{}, mealLinksFake{err: errors.New("unlink failed")})
	result, err := service.Delete(context.Background(), "recipe")
	if err != nil || result.MealUnlinkError == nil || !store.deleted {
		t.Fatalf("deleted=%v warning=%v err=%v", store.deleted, result.MealUnlinkError, err)
	}
}
func value(s string) *string { return &s }

func TestCatalogueCopiesNestedValues(t *testing.T) {
	store := &catalogueFake{}
	service := app.NewCatalogue(store, imageFake{}, mealLinksFake{})
	item := app.SavedRecipe{Title: "Soup", Steps: []string{"Cook"}, Ingredients: []app.IngredientGroup{{Name: "Main", Items: []string{"Water"}}}, PrepTime: value("10 minutes")}
	result, err := service.Create(context.Background(), item, nil)
	if err != nil {
		t.Fatal(err)
	}
	item.Steps[0] = "changed"
	item.Ingredients[0].Items[0] = "changed"
	*item.PrepTime = "changed"
	result.Recipe.Ingredients[0].Items[0] = "also changed"
	if store.recipe.Steps[0] != "Cook" || store.recipe.Ingredients[0].Items[0] != "Water" || *store.recipe.PrepTime != "10 minutes" {
		t.Fatalf("store=%+v", store.recipe)
	}
}
func TestRecipeDetailRestoresLegacyStepsAndTimers(t *testing.T) {
	store := &catalogueFake{recipe: app.SavedRecipe{Instructions: "Mix\nBake 10 minutes"}}
	service := app.NewCatalogue(store, imageFake{}, mealLinksFake{})
	item, err := service.Detail(context.Background(), "recipe")
	if err != nil || len(item.Steps) != 2 || len(item.StepDurations) != 2 || item.StepDurations[1] == nil || *item.StepDurations[1] != 600 {
		t.Fatalf("detail=%+v err=%v", item, err)
	}
}

func TestDeleteRecipeReportsLookupFailureSeparately(t *testing.T) {
	store := &catalogueFake{findErr: errors.New("read failed")}
	service := app.NewCatalogue(store, imageFake{}, mealLinksFake{})
	_, err := service.Delete(context.Background(), "recipe")
	if !errors.Is(err, app.ErrLookup) || store.deleted {
		t.Fatalf("error=%v deleted=%v", err, store.deleted)
	}
}

func TestUpdateReturnsPriorViewWhenReloadFails(t *testing.T) {
	store := &catalogueFake{recipe: app.SavedRecipe{ID: "recipe", Title: "Before", Instructions: "legacy", HasImage: true}}
	service := app.NewCatalogue(store, imageFake{err: errors.New("image failed")}, mealLinksFake{})
	prepared, err := service.PrepareUpdate(context.Background(), "recipe")
	if err != nil {
		t.Fatal(err)
	}
	store.findErr = errors.New("reload failed")
	change := app.SavedRecipe{Title: "After", Steps: []string{"Cook"}, Ingredients: []app.IngredientGroup{{Name: "Main"}}}
	result, err := service.Update(context.Background(), prepared, change, value("new image"))
	if err != nil || result.ImageError == nil || result.Recipe.Title != "After" || result.Recipe.Instructions != "legacy" || !result.Recipe.HasImage || !result.ReloadFallback {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestUpdateRejectsUnpreparedSnapshot(t *testing.T) {
	store := &catalogueFake{}
	service := app.NewCatalogue(store, imageFake{}, mealLinksFake{})
	_, err := service.Update(context.Background(), app.PreparedUpdate{}, app.SavedRecipe{Title: "Changed"}, nil)
	if !errors.Is(err, app.ErrNotFound) || store.recipe.Title != "" {
		t.Fatalf("stored=%+v err=%v", store.recipe, err)
	}
}
