package handlers

import (
	"time"

	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
)

type mealResponse struct {
	Date            string
	MealType        mealapp.Type
	RecipeID        *string
	Name            string
	Notes           string
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func mealResponseFor(meal mealapp.Meal) mealResponse {
	return mealResponse{Date: meal.Date, MealType: meal.MealType, RecipeID: meal.RecipeID, Name: meal.Name, Notes: meal.Notes, CreatedByUserID: meal.CreatedByUserID, CreatedAt: meal.CreatedAt, UpdatedAt: meal.UpdatedAt}
}
func mealResponses(meals []mealapp.Meal) []mealResponse {
	response := make([]mealResponse, len(meals))
	for index, meal := range meals {
		response[index] = mealResponseFor(meal)
	}
	return response
}
