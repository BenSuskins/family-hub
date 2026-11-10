package shopping

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
)

type Store interface {
	FindAll(context.Context) ([]models.ShoppingItem, error)
	Create(context.Context, string, *string) (models.ShoppingItem, error)
	Update(context.Context, string, models.ShoppingChange) (models.ShoppingItem, error)
	Delete(context.Context, string) error
	ClearPurchased(context.Context) error
	Restock(context.Context, string, models.ShoppingRestock) (models.ShoppingItem, error)
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) List(ctx context.Context) ([]models.ShoppingItem, error) {
	return service.store.FindAll(ctx)
}
func (service *Service) Create(ctx context.Context, name string, inventoryID *string) (models.ShoppingItem, error) {
	normalized, err := domain.NormalizeName(name)
	if err != nil {
		return models.ShoppingItem{}, err
	}
	return service.store.Create(ctx, normalized, inventoryID)
}
func (service *Service) Update(ctx context.Context, id string, change models.ShoppingChange) (models.ShoppingItem, error) {
	return service.store.Update(ctx, id, change)
}
func (service *Service) Delete(ctx context.Context, id string) error {
	return service.store.Delete(ctx, id)
}
func (service *Service) ClearPurchased(ctx context.Context) error {
	return service.store.ClearPurchased(ctx)
}
func (service *Service) Restock(ctx context.Context, id string, request models.ShoppingRestock) (models.ShoppingItem, error) {
	return service.store.Restock(ctx, id, request)
}
