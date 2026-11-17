package inventory_test

import (
	"errors"
	"testing"

	inventory "github.com/bensuskins/family-hub/internal/domain/inventory"
)

func TestStockAggregateRestoresAndTransitionsImmutably(t *testing.T) {
	original := inventory.Item{ID: "stock", TrackingMode: inventory.TrackingModeCount, Quantity: 4, Level: 0, LowAt: 1}
	stock, err := inventory.RestoreStock(original)
	if err != nil {
		t.Fatal(err)
	}
	delta := int64(2)
	next, err := stock.Change(inventory.StockChange{QuantityDelta: &delta})
	if err != nil {
		t.Fatal(err)
	}
	if got := stock.Snapshot(); got.Quantity != 4 {
		t.Fatalf("original quantity = %d", got.Quantity)
	}
	if got := next.Snapshot(); got.Quantity != 6 {
		t.Fatalf("updated quantity = %d", got.Quantity)
	}
	changed := next.Snapshot()
	changed.Quantity = 200
	if next.Snapshot().Quantity != 6 {
		t.Fatal("snapshot changed aggregate")
	}
}

func TestStockAggregatePreservesLegacyState(t *testing.T) {
	original := inventory.Item{ID: "old", TrackingMode: "legacy", Quantity: -3, Level: 150, LowAt: -1}
	if _, err := inventory.RestoreStock(original); !errors.Is(err, inventory.ErrInvalidStockState) {
		t.Fatalf("restore = %v", err)
	}
	legacy := inventory.RestoreLegacyStock(original)
	if legacy.Snapshot() != original {
		t.Fatalf("legacy state changed: %+v", legacy.Snapshot())
	}
	level := int64(50)
	if _, err := legacy.Change(inventory.StockChange{Level: &level}); !errors.Is(err, inventory.ErrInvalidStockChange) {
		t.Fatalf("legacy tracking mode = %v", err)
	}
}

func TestLegacyStockTransitionKeepsExistingClamping(t *testing.T) {
	item := inventory.Item{ID: "old", TrackingMode: inventory.TrackingModeCount, Quantity: -5}
	if _, err := inventory.RestoreStock(item); !errors.Is(err, inventory.ErrInvalidStockState) {
		t.Fatalf("restore = %v", err)
	}
	stock := inventory.RestoreLegacyStock(item)
	delta := int64(1)
	updated, err := stock.Change(inventory.StockChange{QuantityDelta: &delta})
	if err != nil || updated.Snapshot().Quantity != 0 {
		t.Fatalf("updated %+v: %v", updated.Snapshot(), err)
	}
	if stock.Snapshot().Quantity != -5 {
		t.Fatal("legacy source was rewritten")
	}
}
