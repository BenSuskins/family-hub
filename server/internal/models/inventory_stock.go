package models

// InventoryStockChange changes one stock value without replacing item details.
type InventoryStockChange struct {
	Quantity      *int64 `json:"quantity,omitempty"`
	QuantityDelta *int64 `json:"quantityDelta,omitempty"`
	Level         *int64 `json:"level,omitempty"`
}
