package handlers

import (
	"time"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
)

type recipeIngredientGroup struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}
type recipeRequest struct {
	Title       string                  `json:"title"`
	Steps       []string                `json:"steps"`
	Ingredients []recipeIngredientGroup `json:"ingredients"`
	MealType    string                  `json:"mealType,omitempty"`
	Servings    *int                    `json:"servings,omitempty"`
	PrepTime    *string                 `json:"prepTime,omitempty"`
	CookTime    *string                 `json:"cookTime,omitempty"`
	SourceURL   *string                 `json:"sourceURL,omitempty"`
	ImageData   *string                 `json:"imageData,omitempty"`
}

func (body recipeRequest) recipe() app.SavedRecipe {
	result := app.SavedRecipe{Title: body.Title, Steps: body.Steps, Servings: body.Servings, PrepTime: body.PrepTime, CookTime: body.CookTime, SourceURL: body.SourceURL}
	if body.MealType != "" {
		value := body.MealType
		result.MealType = &value
	}
	if body.Ingredients != nil {
		result.Ingredients = make([]app.IngredientGroup, len(body.Ingredients))
		for i, group := range body.Ingredients {
			result.Ingredients[i] = app.IngredientGroup{Name: group.Name, Items: group.Items}
		}
	}
	return result
}

type recipeResponse struct {
	ID              string
	Title           string
	Instructions    string
	Steps           []string
	Ingredients     []recipeIngredientGroup
	MealType        *string
	Servings        *int
	PrepTime        *string
	CookTime        *string
	SourceURL       *string
	CategoryID      *string
	HasImage        bool
	StepDurations   []*int
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func mapRecipeResponse(item app.SavedRecipe) recipeResponse {
	result := recipeResponse{ID: item.ID, Title: item.Title, Instructions: item.Instructions, Steps: item.Steps, MealType: item.MealType, Servings: item.Servings, PrepTime: item.PrepTime, CookTime: item.CookTime, SourceURL: item.SourceURL, CategoryID: item.CategoryID, HasImage: item.HasImage, StepDurations: item.StepDurations, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	if item.Ingredients != nil {
		result.Ingredients = make([]recipeIngredientGroup, len(item.Ingredients))
		for i, group := range item.Ingredients {
			result.Ingredients[i] = recipeIngredientGroup{Name: group.Name, Items: group.Items}
		}
	}
	return result
}

func normalizeLegacyRecipeFallback(response *recipeResponse) {
	for i := range response.Ingredients {
		if response.Ingredients[i].Items == nil {
			response.Ingredients[i].Items = []string{}
		}
	}
}
