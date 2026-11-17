package inventory

import "time"

const (
	TrackingModeCount = "count"
	TrackingModeLevel = "level"
)

type Area struct {
	ID              string
	Name            string
	Icon            string
	Tint            string
	Items           []Item
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Item struct {
	ID              string
	AreaID          string
	Name            string
	TrackingMode    string
	Quantity        int
	Level           int
	Unit            string
	LowAt           int
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type StockChange struct {
	Quantity      *int64
	QuantityDelta *int64
	Level         *int64
}
