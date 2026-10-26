package handlers

import (
	"database/sql"
	"errors"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
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
	items, err := handler.shopping.FindAll(r.Context())
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
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
	writeJSON(w, http.StatusCreated, item)
}
func (handler *HouseholdHandler) UpdateShopping(w http.ResponseWriter, r *http.Request) {
	var body models.ShoppingChange
	if !decodeJSONBody(w, r, &body) {
		return
	}
	item, err := handler.shopping.Update(r.Context(), chi.URLParam(r, "id"), body)
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
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
	var body models.ShoppingRestock
	if !decodeJSONBody(w, r, &body) {
		return
	}
	item, err := handler.shopping.Restock(r.Context(), chi.URLParam(r, "id"), body)
	if err != nil {
		shoppingFailure(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func shoppingFailure(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeJSONError(w, http.StatusNotFound, "item not found")
	case errors.Is(err, repository.ErrShoppingConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, repository.ErrShoppingInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		writeAPIError(w, r, http.StatusInternalServerError, "shopping change failed", err)
	}
}
