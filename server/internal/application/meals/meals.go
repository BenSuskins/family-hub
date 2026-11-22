package meals

import (
	"context"
	"errors"
	"fmt"

	"time"
)

type Type string

const (
	Breakfast Type = "breakfast"
	Lunch     Type = "lunch"
	Dinner    Type = "dinner"
)

type Meal struct {
	Date            string
	MealType        Type
	RecipeID        *string
	Name            string
	Notes           string
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

var ErrNotFound = errors.New("meal not found")
var ErrSave = errors.New("saving meal")
var ErrRead = errors.New("reading meal")

type Filter struct {
	DateFrom string
	DateTo   string
}
type Store interface {
	FindByDateAndType(context.Context, string, Type) (Meal, error)
	FindAll(context.Context, Filter) ([]Meal, error)
	FindByDate(context.Context, string) ([]Meal, error)
	Upsert(context.Context, Meal) error
	Delete(context.Context, string, Type) error
	ClearRecipeID(context.Context, string) error
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) FindAll(ctx context.Context, filter Filter) ([]Meal, error) {
	meals, err := service.store.FindAll(ctx, filter)
	return copyMeals(meals), err
}
func (service *Service) FindByDate(ctx context.Context, date string) ([]Meal, error) {
	meals, err := service.store.FindByDate(ctx, date)
	return copyMeals(meals), err
}
func (service *Service) Save(ctx context.Context, meal Meal) (Meal, error) {
	if err := service.store.Upsert(ctx, copyMeal(meal)); err != nil {
		return Meal{}, fmt.Errorf("%w: %w", ErrSave, err)
	}
	result, err := service.store.FindByDateAndType(ctx, meal.Date, meal.MealType)
	if err != nil {
		return Meal{}, fmt.Errorf("%w: %w", ErrRead, err)
	}
	return copyMeal(result), nil
}
func (service *Service) Delete(ctx context.Context, date string, mealType Type) error {
	return service.store.Delete(ctx, date, mealType)
}
func (service *Service) ClearRecipeID(ctx context.Context, id string) error {
	return service.store.ClearRecipeID(ctx, id)
}

func copyMeal(meal Meal) Meal {
	if meal.RecipeID != nil {
		id := *meal.RecipeID
		meal.RecipeID = &id
	}
	return meal
}
func copyMeals(meals []Meal) []Meal {
	if meals == nil {
		return nil
	}
	result := make([]Meal, len(meals))
	for index, meal := range meals {
		result[index] = copyMeal(meal)
	}
	return result
}
