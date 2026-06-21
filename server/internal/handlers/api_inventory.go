package handlers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

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
	areas, err := handler.inventoryRepo.FindAllAreas(ctx)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load inventory")
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

	created, err := handler.inventoryRepo.CreateArea(ctx, models.InventoryArea{
		Name:            body.Name,
		Icon:            body.Icon,
		Tint:            body.Tint,
		CreatedByUserID: user.ID,
	})
	if err != nil {
		slog.Error("creating inventory area via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create area")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateInventoryArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	areaID := chi.URLParam(r, "id")

	area, err := handler.inventoryRepo.FindAreaByID(ctx, areaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load area")
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

	if err := handler.inventoryRepo.UpdateArea(ctx, area); err != nil {
		slog.Error("updating inventory area via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update area")
		return
	}

	updated, err := handler.inventoryRepo.FindAreaByID(ctx, areaID)
	if err != nil {
		writeJSON(w, http.StatusOK, area)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteInventoryArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	areaID := chi.URLParam(r, "id")

	if _, err := handler.inventoryRepo.FindAreaByID(ctx, areaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load area")
		}
		return
	}

	if err := handler.inventoryRepo.DeleteArea(ctx, areaID); err != nil {
		slog.Error("deleting inventory area via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete area")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) CreateInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	areaID := chi.URLParam(r, "id")

	if _, err := handler.inventoryRepo.FindAreaByID(ctx, areaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "area not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load area")
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

	created, err := handler.inventoryRepo.CreateItem(ctx, models.InventoryItem{
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
		slog.Error("creating inventory item via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create item")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID := chi.URLParam(r, "id")

	item, err := handler.inventoryRepo.FindItemByID(ctx, itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "item not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load item")
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

	if err := handler.inventoryRepo.UpdateItem(ctx, item); err != nil {
		slog.Error("updating inventory item via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update item")
		return
	}

	updated, err := handler.inventoryRepo.FindItemByID(ctx, itemID)
	if err != nil {
		writeJSON(w, http.StatusOK, item)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemID := chi.URLParam(r, "id")

	if _, err := handler.inventoryRepo.FindItemByID(ctx, itemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "item not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "failed to load item")
		}
		return
	}

	if err := handler.inventoryRepo.DeleteItem(ctx, itemID); err != nil {
		slog.Error("deleting inventory item via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete item")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
