package inventory

import "errors"

var ErrInvalidStockState = errors.New("invalid inventory stock state")

type Stock struct{ item Item }

func RestoreStock(item Item) (Stock, error) {
	if item.ID == "" || (item.TrackingMode != TrackingModeCount && item.TrackingMode != TrackingModeLevel) || item.Quantity < 0 || item.Level < 0 || item.Level > 100 || item.LowAt < 0 || (item.TrackingMode == TrackingModeLevel && item.LowAt > 100) {
		return Stock{}, ErrInvalidStockState
	}
	return Stock{item: item}, nil
}

// RestoreLegacyStock retains stored values that do not meet current invariants.
func RestoreLegacyStock(item Item) Stock { return Stock{item: item} }
func (stock Stock) Snapshot() Item       { return stock.item }
func (stock Stock) Change(change StockChange) (Stock, error) {
	item, err := ApplyStock(stock.item, change)
	if err != nil {
		return Stock{}, err
	}
	return Stock{item: item}, nil
}
