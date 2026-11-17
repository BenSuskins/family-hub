package inventory

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

type StockStore interface {
	ModifyStock(context.Context, string, func(domain.Stock) (domain.Stock, error)) (domain.Item, error)
}

type StockService struct{ store StockStore }

func NewStockService(store StockStore) *StockService { return &StockService{store: store} }
func (service *StockService) Update(ctx context.Context, id string, change domain.StockChange) (domain.Item, error) {
	change.Quantity = copyStockValue(change.Quantity)
	change.QuantityDelta = copyStockValue(change.QuantityDelta)
	change.Level = copyStockValue(change.Level)
	if err := domain.ValidateStockChange(change); err != nil {
		return domain.Item{}, err
	}
	return service.store.ModifyStock(ctx, id, func(stock domain.Stock) (domain.Stock, error) { return stock.Change(change) })
}

func copyStockValue(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
