package categories

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrReload = errors.New("category reload failed")
var ErrNotFound = errors.New("category not found")

type Category struct {
	ID              string
	Name            string
	CreatedByUserID string
	CreatedAt       time.Time
}

type Store interface {
	FindByID(context.Context, string) (Category, error)
	FindAll(context.Context) ([]Category, error)
	Create(context.Context, Category) (Category, error)
	Update(context.Context, string, string) error
	Delete(context.Context, string) error
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) List(ctx context.Context) ([]Category, error) {
	items, err := service.store.FindAll(ctx)
	if err != nil || items == nil {
		return items, err
	}
	result := make([]Category, len(items))
	copy(result, items)
	return result, nil
}
func (service *Service) Create(ctx context.Context, name, creatorID string) (Category, error) {
	return service.store.Create(ctx, Category{Name: name, CreatedByUserID: creatorID})
}
func (service *Service) Update(ctx context.Context, id, name string) (Category, error) {
	if err := service.store.Update(ctx, id, name); err != nil {
		return Category{}, err
	}
	category, err := service.store.FindByID(ctx, id)
	if err != nil {
		return Category{}, fmt.Errorf("%w: %w", ErrReload, err)
	}
	return category, nil
}
func (service *Service) Delete(ctx context.Context, id string) error {
	return service.store.Delete(ctx, id)
}
