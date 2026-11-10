package handlers

import (
	"errors"
	"net/http"

	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/go-chi/chi/v5"
)

// areaAPIBody is the JSON request body for creating/updating an area.
type areaAPIBody struct {
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
	Tint string `json:"tint,omitempty"`
}

// itemAPIBody is the JSON request body for creating/updating an item.
type itemAPIBody struct {
	Name         string `json:"name"`
	TrackingMode string `json:"trackingMode,omitempty"`
	Quantity     int    `json:"quantity"`
	Level        int    `json:"level"`
	Unit         string `json:"unit,omitempty"`
	LowAt        int    `json:"lowAt"`
}

// ListInventory returns every area with its items nested — a single call backs
// the iOS Inventory home screen (areas list + running-low rollup).
func (handler *APIHandler) ListInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	areas, err := handler.inventoryService.FindAllAreas(ctx)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load inventory", err)
		return
	}
	if areas == nil {
		areas = []models.InventoryArea{}
	}
	writeJSON(w, http.StatusOK, areas)
}

func (handler *APIHandler) CreateInventoryArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body areaAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := handler.inventoryService.CreateArea(ctx, models.InventoryArea{
		Name:            body.Name,
		Icon:            body.Icon,
		Tint:            body.Tint,
		CreatedByUserID: user.ID,
	})
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to create area", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateInventoryArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	areaID := chi.URLParam(r, "id")

	area, err := handler.inventoryService.FindAreaByID(ctx, areaID)
	if err != nil {
		if errors.Is(err, inventorydomain.ErrAreaNotFound) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load area", err)
		}
		return
	}

	var body areaAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	area.Name = body.Name
	if body.Icon != "" {
		area.Icon = body.Icon
	}
	if body.Tint != "" {
		area.Tint = body.Tint
	}

	updated, err := handler.inventoryService.SaveArea(ctx, area)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to update area", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteInventoryArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	areaID := chi.URLParam(r, "id")

	if _, err := handler.inventoryService.FindAreaByID(ctx, areaID); err != nil {
		if errors.Is(err, inventorydomain.ErrAreaNotFound) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load area", err)
		}
		return
	}

	if err := handler.inventoryService.DeleteArea(ctx, areaID); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to delete area", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) CreateInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	areaID := chi.URLParam(r, "id")

	if _, err := handler.inventoryService.FindAreaByID(ctx, areaID); err != nil {
		if errors.Is(err, inventorydomain.ErrAreaNotFound) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load area", err)
		}
		return
	}

	var body itemAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := handler.inventoryService.CreateItem(ctx, models.InventoryItem{
		AreaID:          areaID,
		Name:            body.Name,
		TrackingMode:    body.TrackingMode,
		Quantity:        body.Quantity,
		Level:           body.Level,
		Unit:            body.Unit,
		LowAt:           body.LowAt,
		CreatedByUserID: user.ID,
	})
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to create item", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID := chi.URLParam(r, "id")

	item, err := handler.inventoryService.FindItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, inventorydomain.ErrItemNotFound) {
			writeJSONError(w, http.StatusNotFound, "item not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load item", err)
		}
		return
	}

	var body itemAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	item.Name = body.Name
	item.TrackingMode = body.TrackingMode
	item.Quantity = body.Quantity
	item.Level = body.Level
	item.Unit = body.Unit
	item.LowAt = body.LowAt

	updated, err := handler.inventoryService.SaveItem(ctx, item)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to update item", err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID := chi.URLParam(r, "id")

	if _, err := handler.inventoryService.FindItemByID(ctx, itemID); err != nil {
		if errors.Is(err, inventorydomain.ErrItemNotFound) {
			writeJSONError(w, http.StatusNotFound, "item not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load item", err)
		}
		return
	}

	if err := handler.inventoryService.DeleteItem(ctx, itemID); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to delete item", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) UpdateInventoryStock(w http.ResponseWriter, r *http.Request) {
	var change models.InventoryStockChange
	if !decodeJSONBody(w, r, &change) {
		return
	}
	item, err := handler.stockService.Update(r.Context(), chi.URLParam(r, "id"), change)
	if err != nil {
		switch {
		case errors.Is(err, inventorydomain.ErrItemNotFound):
			writeJSONError(w, http.StatusNotFound, "item not found")
		case errors.Is(err, inventorydomain.ErrInvalidStockChange):
			writeJSONError(w, http.StatusBadRequest, "supply one valid stock value for the item's tracking mode")
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to update stock", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, item)
}
