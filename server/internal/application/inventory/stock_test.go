package inventory_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

type stockFake struct {
	item   domain.Item
	calls  int
	before func()
}

func (fake *stockFake) ModifyStock(_ context.Context, _ string, change func(domain.Stock) (domain.Stock, error)) (domain.Item, error) {
	fake.calls++
	if fake.before != nil {
		fake.before()
	}
	current := domain.RestoreLegacyStock(fake.item)
	updated, err := change(current)
	if err == nil {
		fake.item = updated.Snapshot()
	}
	return updated.Snapshot(), err
}
func TestUpdateStockValidatesBeforeStorage(t *testing.T) {
	store := &stockFake{item: domain.Item{TrackingMode: domain.TrackingModeCount, Quantity: 4}}
	service := app.NewStockService(store)
	if _, err := service.Update(context.Background(), "item", domain.StockChange{}); !errors.Is(err, domain.ErrInvalidStockChange) {
		t.Fatalf("got %v", err)
	}
	if store.calls != 0 {
		t.Fatal("invalid request reached storage")
	}
	delta := int64(2)
	saved, err := service.Update(context.Background(), "item", domain.StockChange{QuantityDelta: &delta})
	if err != nil || saved.Quantity != 6 {
		t.Fatalf("saved %+v: %v", saved, err)
	}
}

func TestUpdateStockCopiesInputBeforeStorage(t *testing.T) {
	delta := int64(2)
	store := &stockFake{item: domain.Item{ID: "stock", TrackingMode: domain.TrackingModeCount, Quantity: 4}, before: func() { delta = 100 }}
	saved, err := app.NewStockService(store).Update(context.Background(), "stock", domain.StockChange{QuantityDelta: &delta})
	if err != nil || saved.Quantity != 6 {
		t.Fatalf("saved %+v: %v", saved, err)
	}
}
