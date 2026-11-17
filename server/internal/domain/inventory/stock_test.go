package inventory_test

import (
	"errors"
	"math"
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/inventory"
)

func TestStockChange(t *testing.T) {
	value := func(v int64) *int64 { return &v }
	cases := []struct {
		name            string
		item            inventory.Item
		change          inventory.StockChange
		quantity, level int
		invalid         bool
	}{
		{name: "replace count", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: 3}, change: inventory.StockChange{Quantity: value(8)}, quantity: 8},
		{name: "decrement clamps at zero", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: 3}, change: inventory.StockChange{QuantityDelta: value(-9)}},
		{name: "legacy negative quantity clamps", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: -5}, change: inventory.StockChange{QuantityDelta: value(1)}},
		{name: "minimum quantity avoids overflow", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: math.MinInt64}, change: inventory.StockChange{QuantityDelta: value(-1)}},
		{name: "increment", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: 3}, change: inventory.StockChange{QuantityDelta: value(2)}, quantity: 5},
		{name: "overflow", item: inventory.Item{TrackingMode: inventory.TrackingModeCount, Quantity: math.MaxInt64}, change: inventory.StockChange{QuantityDelta: value(1)}, invalid: true},
		{name: "level", item: inventory.Item{TrackingMode: inventory.TrackingModeLevel, Level: 40}, change: inventory.StockChange{Level: value(90)}, level: 90},
		{name: "wrong mode", item: inventory.Item{TrackingMode: inventory.TrackingModeLevel}, change: inventory.StockChange{Quantity: value(2)}, invalid: true},
		{name: "too high", item: inventory.Item{TrackingMode: inventory.TrackingModeLevel}, change: inventory.StockChange{Level: value(101)}, invalid: true},
		{name: "multiple changes", item: inventory.Item{TrackingMode: inventory.TrackingModeCount}, change: inventory.StockChange{Quantity: value(1), QuantityDelta: value(1)}, invalid: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			original := test.item
			updated, err := inventory.ApplyStock(test.item, test.change)
			if test.invalid {
				if !errors.Is(err, inventory.ErrInvalidStockChange) {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != nil || updated.Quantity != test.quantity || updated.Level != test.level {
				t.Fatalf("updated %+v: %v", updated, err)
			}
			if test.item != original {
				t.Fatal("input changed")
			}
		})
	}
}

func TestNormalizeItem(t *testing.T) {
	item := inventory.Item{TrackingMode: "legacy", Quantity: -2, Level: 101, LowAt: -3}
	got := inventory.NormalizeItem(item)
	if got.TrackingMode != inventory.TrackingModeCount || got.Quantity != 0 || got.Level != 100 || got.LowAt != 0 {
		t.Fatalf("normalized item = %+v", got)
	}
	level := inventory.NormalizeItem(inventory.Item{TrackingMode: inventory.TrackingModeLevel, LowAt: 101})
	if level.LowAt != 100 {
		t.Fatalf("level threshold = %d, want 100", level.LowAt)
	}
	if item.Quantity != -2 {
		t.Fatal("normalization changed input")
	}
}
