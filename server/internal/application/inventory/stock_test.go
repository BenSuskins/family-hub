package inventory_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
)

type stockFake struct {
	item  models.InventoryItem
	calls int
}

func (fake *stockFake) ModifyStock(_ context.Context, _ string, change func(models.InventoryItem) (models.InventoryItem, error)) (models.InventoryItem, error) {
	fake.calls++
	updated, err := change(fake.item)
	if err == nil {
		fake.item = updated
	}
	return updated, err
}
func TestUpdateStockValidatesBeforeStorage(t *testing.T) {
	store := &stockFake{item: models.InventoryItem{TrackingMode: models.TrackingModeCount, Quantity: 4}}
	service := app.NewStockService(store)
	if _, err := service.Update(context.Background(), "item", models.InventoryStockChange{}); !errors.Is(err, domain.ErrInvalidStockChange) {
		t.Fatalf("got %v", err)
	}
	if store.calls != 0 {
		t.Fatal("invalid request reached storage")
	}
	delta := int64(2)
	saved, err := service.Update(context.Background(), "item", models.InventoryStockChange{QuantityDelta: &delta})
	if err != nil || saved.Quantity != 6 {
		t.Fatalf("saved %+v: %v", saved, err)
	}
}
