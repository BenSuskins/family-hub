package handlers

import (
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"time"
)

type inventoryAreaResponse struct {
	ID              string
	Name            string
	Icon            string
	Tint            string
	Items           []inventoryItemResponse
	CreatedByUserID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
type inventoryItemResponse struct {
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

func inventoryItemResponseFor(item inventorydomain.Item) inventoryItemResponse {
	return inventoryItemResponse{ID: item.ID, AreaID: item.AreaID, Name: item.Name, TrackingMode: item.TrackingMode, Quantity: item.Quantity, Level: item.Level, Unit: item.Unit, LowAt: item.LowAt, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func inventoryAreaResponseFor(area inventorydomain.Area) inventoryAreaResponse {
	var items []inventoryItemResponse
	if area.Items != nil {
		items = make([]inventoryItemResponse, len(area.Items))
		for index, item := range area.Items {
			items[index] = inventoryItemResponseFor(item)
		}
	}
	return inventoryAreaResponse{ID: area.ID, Name: area.Name, Icon: area.Icon, Tint: area.Tint, Items: items, CreatedByUserID: area.CreatedByUserID, CreatedAt: area.CreatedAt, UpdatedAt: area.UpdatedAt}
}
func inventoryAreaResponses(areas []inventorydomain.Area) []inventoryAreaResponse {
	response := make([]inventoryAreaResponse, len(areas))
	for index, area := range areas {
		response[index] = inventoryAreaResponseFor(area)
	}
	return response
}
