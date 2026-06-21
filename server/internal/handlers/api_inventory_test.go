package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func newInventoryTestUser(t *testing.T, userRepo *repository.SQLiteUserRepository) models.User {
	t.Helper()
	user, err := userRepo.Create(context.Background(), models.User{
		OIDCSubject: "sub-inventory",
		Email:       "inventory@example.com",
		Name:        "Inventory User",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return user
}

func TestListInventory_EmptyIsNotNull(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	invRepo := repository.NewInventoryRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, invRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/inventory", handler.ListInventory)

	request := httptest.NewRequest(http.MethodGet, "/api/inventory", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if strings.TrimSpace(recorder.Body.String()) != "[]" {
		t.Errorf("expected [] not null, got %s", recorder.Body.String())
	}
}

func TestCreateInventoryArea_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	invRepo := repository.NewInventoryRepository(database)
	userRepo := repository.NewUserRepository(database)
	user := newInventoryTestUser(t, userRepo)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, invRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/inventory/areas", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateInventoryArea(w, r.WithContext(ctx))
	})

	body := `{"name":"Laundry cupboard","icon":"drop","tint":"blue"}`
	request := httptest.NewRequest(http.MethodPost, "/api/inventory/areas", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var area models.InventoryArea
	json.NewDecoder(recorder.Body).Decode(&area)
	if area.Name != "Laundry cupboard" || area.Icon != "drop" || area.Tint != "blue" {
		t.Errorf("unexpected area: %+v", area)
	}
	if area.ID == "" {
		t.Error("expected non-empty area ID")
	}
}

func TestCreateInventoryArea_API_MissingName(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	invRepo := repository.NewInventoryRepository(database)
	userRepo := repository.NewUserRepository(database)
	user := newInventoryTestUser(t, userRepo)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, invRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/inventory/areas", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateInventoryArea(w, r.WithContext(ctx))
	})

	request := httptest.NewRequest(http.MethodPost, "/api/inventory/areas", strings.NewReader(`{"name":""}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

// TestInventory_FullFlow exercises create area → add item → list (nested) →
// update item quantity → delete area cascades.
func TestInventory_FullFlow(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	invRepo := repository.NewInventoryRepository(database)
	userRepo := repository.NewUserRepository(database)
	user := newInventoryTestUser(t, userRepo)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, invRepo, nil, nil, "", "", "")

	withUser := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next(w, r.WithContext(ctx))
		}
	}

	router := chi.NewRouter()
	router.Get("/api/inventory", handler.ListInventory)
	router.Post("/api/inventory/areas", withUser(handler.CreateInventoryArea))
	router.Post("/api/inventory/areas/{id}/items", withUser(handler.CreateInventoryItem))
	router.Put("/api/inventory/items/{id}", handler.UpdateInventoryItem)
	router.Delete("/api/inventory/areas/{id}", handler.DeleteInventoryArea)

	// Create area
	rec := doRequest(t, router, http.MethodPost, "/api/inventory/areas", `{"name":"Bathroom","icon":"pills","tint":"teal"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create area: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var area models.InventoryArea
	json.NewDecoder(rec.Body).Decode(&area)

	// Add item
	rec = doRequest(t, router, http.MethodPost, "/api/inventory/areas/"+area.ID+"/items",
		`{"name":"Toilet roll","quantity":12,"unit":"rolls","lowAt":8}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create item: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var item models.InventoryItem
	json.NewDecoder(rec.Body).Decode(&item)
	if item.AreaID != area.ID || item.TrackingMode != models.TrackingModeCount || item.Quantity != 12 || item.LowAt != 8 {
		t.Fatalf("unexpected item: %+v", item)
	}

	// List nests the item under its area
	rec = doRequest(t, router, http.MethodGet, "/api/inventory", "")
	var areas []models.InventoryArea
	json.NewDecoder(rec.Body).Decode(&areas)
	if len(areas) != 1 || len(areas[0].Items) != 1 {
		t.Fatalf("expected 1 area with 1 item, got %+v", areas)
	}

	// Update item quantity (stepper)
	rec = doRequest(t, router, http.MethodPut, "/api/inventory/items/"+item.ID,
		`{"name":"Toilet roll","quantity":3,"unit":"rolls","lowAt":8}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update item: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated models.InventoryItem
	json.NewDecoder(rec.Body).Decode(&updated)
	if updated.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", updated.Quantity)
	}

	// A level-tracked item round-trips its percentage and threshold
	rec = doRequest(t, router, http.MethodPost, "/api/inventory/areas/"+area.ID+"/items",
		`{"name":"Fabric softener","trackingMode":"level","level":40,"lowAt":20}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create level item: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var levelItem models.InventoryItem
	json.NewDecoder(rec.Body).Decode(&levelItem)
	if levelItem.TrackingMode != models.TrackingModeLevel || levelItem.Level != 40 || levelItem.LowAt != 20 {
		t.Errorf("unexpected level item: %+v", levelItem)
	}

	// Delete area cascades the item
	rec = doRequest(t, router, http.MethodDelete, "/api/inventory/areas/"+area.ID, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete area: expected 204, got %d", rec.Code)
	}
	rec = doRequest(t, router, http.MethodPut, "/api/inventory/items/"+item.ID,
		`{"name":"Toilet roll","quantity":1,"unit":"rolls","lowAt":8}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected item to be gone (404) after area delete, got %d", rec.Code)
	}
}

func doRequest(t *testing.T, router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}
