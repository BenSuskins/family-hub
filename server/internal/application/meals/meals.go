package meals

import (
	"context"
	"errors"
	"fmt"

	"github.com/bensuskins/family-hub/internal/models"
)

var ErrSave = errors.New("saving meal")
var ErrRead = errors.New("reading meal")

type Filter struct {
	DateFrom string
	DateTo   string
}
type Store interface {
	FindByDateAndType(context.Context, string, models.MealType) (models.MealPlan, error)
	FindAll(context.Context, Filter) ([]models.MealPlan, error)
	FindByDate(context.Context, string) ([]models.MealPlan, error)
	Upsert(context.Context, models.MealPlan) error
	Delete(context.Context, string, models.MealType) error
	ClearRecipeID(context.Context, string) error
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) FindAll(ctx context.Context, filter Filter) ([]models.MealPlan, error) {
	return service.store.FindAll(ctx, filter)
}
func (service *Service) FindByDate(ctx context.Context, date string) ([]models.MealPlan, error) {
	return service.store.FindByDate(ctx, date)
}
func (service *Service) Save(ctx context.Context, meal models.MealPlan) (models.MealPlan, error) {
	if err := service.store.Upsert(ctx, meal); err != nil {
		return models.MealPlan{}, fmt.Errorf("%w: %w", ErrSave, err)
	}
	result, err := service.store.FindByDateAndType(ctx, meal.Date, meal.MealType)
	if err != nil {
		return models.MealPlan{}, fmt.Errorf("%w: %w", ErrRead, err)
	}
	return result, nil
}
func (service *Service) Delete(ctx context.Context, date string, mealType models.MealType) error {
	return service.store.Delete(ctx, date, mealType)
}
func (service *Service) ClearRecipeID(ctx context.Context, id string) error {
	return service.store.ClearRecipeID(ctx, id)
}
