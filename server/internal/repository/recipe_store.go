package repository

import (
	"context"
	"database/sql"
	"errors"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
	"github.com/bensuskins/family-hub/internal/models"
)

func (store *SQLiteRecipeRepository) FindByID(ctx context.Context, id string) (app.SavedRecipe, error) {
	item, err := store.findByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return app.SavedRecipe{}, app.ErrNotFound
	}
	return recipeFromModel(item), err
}
func (store *SQLiteRecipeRepository) FindAll(ctx context.Context) ([]app.SavedRecipe, error) {
	items, err := store.findAll(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, nil
	}
	result := make([]app.SavedRecipe, len(items))
	for i, item := range items {
		result[i] = recipeFromModel(item)
	}
	return result, nil
}
func (store *SQLiteRecipeRepository) Create(ctx context.Context, item app.SavedRecipe) (app.SavedRecipe, error) {
	created, err := store.create(ctx, recipeToModel(item))
	return recipeFromModel(created), err
}
func (store *SQLiteRecipeRepository) Update(ctx context.Context, item app.SavedRecipe) error {
	return store.update(ctx, recipeToModel(item))
}
func (store *SQLiteRecipeRepository) Delete(ctx context.Context, id string) error {
	return store.delete(ctx, id)
}
func recipeFromModel(item models.Recipe) app.SavedRecipe {
	result := app.SavedRecipe{ID: item.ID, Title: item.Title, Instructions: item.Instructions, Servings: copyRecipePointer(item.Servings), PrepTime: copyRecipePointer(item.PrepTime), CookTime: copyRecipePointer(item.CookTime), SourceURL: copyRecipePointer(item.SourceURL), CategoryID: copyRecipePointer(item.CategoryID), HasImage: item.HasImage, StepDurations: copyRecipeDurations(item.StepDurations), CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.MealType != nil {
		value := string(*item.MealType)
		result.MealType = &value
	}
	if item.Steps != nil {
		result.Steps = append([]string{}, item.Steps...)
	}
	if item.Ingredients != nil {
		result.Ingredients = make([]app.IngredientGroup, len(item.Ingredients))
		for i, group := range item.Ingredients {
			result.Ingredients[i].Name = group.Name
			if group.Items != nil {
				result.Ingredients[i].Items = append([]string{}, group.Items...)
			}
		}
	}
	return result
}
func recipeToModel(item app.SavedRecipe) models.Recipe {
	result := models.Recipe{ID: item.ID, Title: item.Title, Instructions: item.Instructions, Servings: copyRecipePointer(item.Servings), PrepTime: copyRecipePointer(item.PrepTime), CookTime: copyRecipePointer(item.CookTime), SourceURL: copyRecipePointer(item.SourceURL), CategoryID: copyRecipePointer(item.CategoryID), HasImage: item.HasImage, StepDurations: copyRecipeDurations(item.StepDurations), CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.MealType != nil {
		value := models.RecipeMealType(*item.MealType)
		result.MealType = &value
	}
	if item.Steps != nil {
		result.Steps = append([]string{}, item.Steps...)
	}
	if item.Ingredients != nil {
		result.Ingredients = make([]models.IngredientGroup, len(item.Ingredients))
		for i, group := range item.Ingredients {
			result.Ingredients[i].Name = group.Name
			if group.Items != nil {
				result.Ingredients[i].Items = append([]string{}, group.Items...)
			}
		}
	}
	return result
}

func copyRecipePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func copyRecipeDurations(values []*int) []*int {
	if values == nil {
		return nil
	}
	result := make([]*int, len(values))
	for i, value := range values {
		result[i] = copyRecipePointer(value)
	}
	return result
}
