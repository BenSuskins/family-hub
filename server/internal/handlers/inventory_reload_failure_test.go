package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	inventoryapp "github.com/bensuskins/family-hub/internal/application/inventory"
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/go-chi/chi/v5"
)

type inventoryReloadStore struct {
	item  inventorydomain.Item
	saved inventorydomain.Item
	reads int
}

func (store *inventoryReloadStore) FindAllAreas(context.Context) ([]inventorydomain.Area, error) {
	return nil, nil
}
func (store *inventoryReloadStore) FindAreaByID(context.Context, string) (inventorydomain.Area, error) {
	return inventorydomain.Area{}, nil
}
func (store *inventoryReloadStore) CreateArea(context.Context, inventorydomain.Area) (inventorydomain.Area, error) {
	return inventorydomain.Area{}, nil
}
func (store *inventoryReloadStore) UpdateArea(context.Context, inventorydomain.Area) error {
	return nil
}
func (store *inventoryReloadStore) DeleteArea(context.Context, string) error { return nil }
func (store *inventoryReloadStore) FindItemByID(context.Context, string) (inventorydomain.Item, error) {
	store.reads++
	if store.reads > 1 {
		return inventorydomain.Item{}, errors.New("injected reload failure")
	}
	return store.item, nil
}
func (store *inventoryReloadStore) CreateItem(context.Context, inventorydomain.Item) (inventorydomain.Item, error) {
	return inventorydomain.Item{}, nil
}
func (store *inventoryReloadStore) UpdateItem(_ context.Context, item inventorydomain.Item) error {
	store.saved = item
	return nil
}
func (store *inventoryReloadStore) DeleteItem(context.Context, string) error { return nil }

func TestUpdateInventoryItemReloadFailureReturnsUnnormalizedInput(t *testing.T) {
	store := &inventoryReloadStore{item: inventorydomain.Item{ID: "item", AreaID: "area", Name: "Old", TrackingMode: "count"}}
	handler := &APIHandler{inventoryService: inventoryapp.New(store)}
	router := chi.NewRouter()
	router.Put("/api/inventory/items/{id}", handler.UpdateInventoryItem)
	request := httptest.NewRequest(http.MethodPut, "/api/inventory/items/item", strings.NewReader(`{"name":"Updated","trackingMode":"legacy","quantity":-4,"level":101,"lowAt":-3}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		ID           string
		Name         string
		TrackingMode string
		Quantity     int
		Level        int
		LowAt        int
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ID != "item" || body.Name != "Updated" || body.TrackingMode != "legacy" || body.Quantity != -4 || body.Level != 101 || body.LowAt != -3 {
		t.Fatalf("fallback response = %s", response.Body.String())
	}
	if store.reads != 2 || store.saved.TrackingMode != "count" || store.saved.Quantity != 0 || store.saved.Level != 100 || store.saved.LowAt != 0 {
		t.Fatalf("reads = %d, written item = %+v", store.reads, store.saved)
	}
}
