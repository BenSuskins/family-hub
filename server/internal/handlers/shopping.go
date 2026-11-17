package handlers

import (
	"errors"
	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (handler *HouseholdHandler) RegisterShopping(router chi.Router) {
	router.Get("/api/shopping", handler.ListShopping)
	router.Post("/api/shopping/items", handler.CreateShopping)
	router.Patch("/api/shopping/items/{id}", handler.UpdateShopping)
	router.Delete("/api/shopping/items/{id}", handler.DeleteShopping)
	router.Delete("/api/shopping/purchased", handler.ClearShopping)
	router.Post("/api/shopping/items/{id}/restock", handler.RestockShopping)
}
func (handler *HouseholdHandler) ListShopping(w http.ResponseWriter, r *http.Request) {
	items, err := handler.shopping.List(r.Context())
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, shoppingResponses(items))
}
func (handler *HouseholdHandler) CreateShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name            string  `json:"name"`
		InventoryItemID *string `json:"inventoryItemID"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	item, err := handler.shopping.Create(r.Context(), body.Name, body.InventoryItemID)
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, shoppingResponseFor(item))
}
func (handler *HouseholdHandler) UpdateShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      *string `json:"name"`
		Purchased *bool   `json:"purchased"`
		Version   int     `json:"version"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	item, err := handler.shopping.Update(r.Context(), chi.URLParam(r, "id"), shoppingdomain.Change{Name: body.Name, Purchased: body.Purchased, Version: body.Version})
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, shoppingResponseFor(item))
}
func (handler *HouseholdHandler) DeleteShopping(w http.ResponseWriter, r *http.Request) {
	if err := handler.shopping.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		shoppingFailure(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (handler *HouseholdHandler) ClearShopping(w http.ResponseWriter, r *http.Request) {
	if err := handler.shopping.ClearPurchased(r.Context()); err != nil {
		shoppingFailure(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (handler *HouseholdHandler) RestockShopping(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PurchaseID string `json:"purchaseID"`
		Quantity   *int   `json:"quantity"`
		Level      *int   `json:"level"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	item, err := handler.shopping.Restock(r.Context(), chi.URLParam(r, "id"), shoppingdomain.Restock{PurchaseID: body.PurchaseID, Quantity: body.Quantity, Level: body.Level})
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, shoppingResponseFor(item))
}
func shoppingFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, shoppingdomain.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, "item not found")
	case errors.Is(err, shoppingdomain.ErrConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, shoppingdomain.ErrInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		writeAPIError(w, r, http.StatusInternalServerError, "shopping change failed", err)
	}
}

type shoppingResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	InventoryItemID *string `json:"inventoryItemID"`
	Purchased       bool    `json:"purchased"`
	PurchaseID      *string `json:"purchaseID"`
	Restocked       bool    `json:"restocked"`
	Version         int     `json:"version"`
}

func shoppingResponseFor(item shoppingdomain.Item) shoppingResponse {
	return shoppingResponse{ID: item.ID, Name: item.Name, InventoryItemID: item.InventoryItemID, Purchased: item.Purchased, PurchaseID: item.PurchaseID, Restocked: item.Restocked, Version: item.Version}
}
func shoppingResponses(items []shoppingdomain.Item) []shoppingResponse {
	if items == nil {
		return nil
	}
	result := make([]shoppingResponse, len(items))
	for index, item := range items {
		result[index] = shoppingResponseFor(item)
	}
	return result
}
