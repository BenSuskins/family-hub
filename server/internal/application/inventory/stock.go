package inventory

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
)

type StockStore interface {
	ModifyStock(context.Context, string, func(models.InventoryItem) (models.InventoryItem, error)) (models.InventoryItem, error)
}

type StockService struct{ store StockStore }

func NewStockService(store StockStore) *StockService { return &StockService{store: store} }
func (service *StockService) Update(ctx context.Context, id string, change models.InventoryStockChange) (models.InventoryItem, error) {
	if err := domain.ValidateStockChange(change); err != nil {
		return models.InventoryItem{}, err
	}
	return service.store.ModifyStock(ctx, id, func(item models.InventoryItem) (models.InventoryItem, error) { return domain.ApplyStock(item, change) })
}
