package inventory

import (
	"errors"
	"math"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrInvalidStockChange = errors.New("invalid stock change")
	ErrItemNotFound       = errors.New("inventory item not found")
	ErrAreaNotFound       = errors.New("inventory area not found")
)

func ValidateStockChange(change models.InventoryStockChange) error {
	count := 0
	for _, value := range []*int64{change.Quantity, change.QuantityDelta, change.Level} {
		if value != nil {
			count++
		}
	}
	if count != 1 || change.Quantity != nil && *change.Quantity < 0 || change.Level != nil && (*change.Level < 0 || *change.Level > 100) {
		return ErrInvalidStockChange
	}
	return nil
}

func ApplyStock(item models.InventoryItem, change models.InventoryStockChange) (models.InventoryItem, error) {
	if err := ValidateStockChange(change); err != nil {
		return models.InventoryItem{}, err
	}
	switch {
	case change.Quantity != nil:
		if item.TrackingMode != models.TrackingModeCount {
			return models.InventoryItem{}, ErrInvalidStockChange
		}
		item.Quantity = int(*change.Quantity)
	case change.QuantityDelta != nil:
		if item.TrackingMode != models.TrackingModeCount {
			return models.InventoryItem{}, ErrInvalidStockChange
		}
		delta := *change.QuantityDelta
		quantity := int64(item.Quantity)
		if delta > 0 && quantity > math.MaxInt64-delta {
			return models.InventoryItem{}, ErrInvalidStockChange
		}
		if delta < 0 && (quantity <= 0 || delta < -quantity) {
			item.Quantity = 0
		} else {
			result := quantity + delta
			if result < 0 {
				result = 0
			}
			item.Quantity = int(result)
		}
	case change.Level != nil:
		if item.TrackingMode != models.TrackingModeLevel {
			return models.InventoryItem{}, ErrInvalidStockChange
		}
		item.Level = int(*change.Level)
	}
	return item, nil
}

// NormalizeItem clamps an item's fields to valid ranges before persisting:
// quantity and low_at are never negative, level is bounded to 0–100, and in
// level mode low_at is read as a percentage so it is bounded to 0–100 too. An
// unrecognized tracking mode falls back to count.
func NormalizeItem(item models.InventoryItem) models.InventoryItem {
	if item.TrackingMode != models.TrackingModeLevel {
		item.TrackingMode = models.TrackingModeCount
	}
	item.Quantity = atLeast(item.Quantity, 0)
	item.Level = clamp(item.Level, 0, 100)
	if item.TrackingMode == models.TrackingModeLevel {
		item.LowAt = clamp(item.LowAt, 0, 100)
	} else {
		item.LowAt = atLeast(item.LowAt, 0)
	}
	return item
}

func atLeast(value, low int) int {
	if value < low {
		return low
	}
	return value
}

func clamp(value, low, high int) int {
	return atLeast(min(value, high), low)
}
