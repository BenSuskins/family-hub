package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestHomeInfoAdminAndValidation(t *testing.T) {
	handler := NewHouseholdHandler(repository.NewSettingsRepository(testutil.NewTestDatabase(t)), nil, nil, nil)
	for _, example := range []struct {
		role   models.Role
		body   string
		status int
	}{
		{models.RoleMember, `{"latitude":51.8,"longitude":0.7,"timezone":"Europe/London","arrivalMarginMinutes":10}`, 403},
		{models.RoleAdmin, `{"latitude":91,"longitude":0.7,"timezone":"Europe/London","arrivalMarginMinutes":10}`, 400},
		{models.RoleAdmin, `{"latitude":51.8,"longitude":0.7,"timezone":"invalid","arrivalMarginMinutes":10}`, 400},
		{models.RoleAdmin, `{"latitude":51.8,"longitude":0.7,"timezone":"Europe/London","arrivalMarginMinutes":121}`, 400},
		{models.RoleAdmin, `{"latitude":51.8,"longitude":0.7,"timezone":"Europe/London","arrivalMarginMinutes":10}`, 200},
	} {
		request := httptest.NewRequest(http.MethodPatch, "/api/home-info", strings.NewReader(example.body))
		request = request.WithContext(context.WithValue(request.Context(), middleware.UserContextKey, models.User{Role: example.role}))
		response := httptest.NewRecorder()
		handler.UpdateHomeInfo(response, request)
		if response.Code != example.status {
			t.Fatalf("%s: got %d: %s", example.body, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.HomeInfo(response, httptest.NewRequest(http.MethodGet, "/api/home-info", nil))
	var home models.HomeInfo
	if err := json.Unmarshal(response.Body.Bytes(), &home); err != nil {
		t.Fatal(err)
	}
	if home.Latitude == nil || *home.Latitude != 51.8 || home.ArrivalMarginMinutes != 10 {
		t.Fatalf("unexpected home: %+v", home)
	}
}

func TestShoppingPurchaseRestockAndDeletedInventory(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	database.SetMaxOpenConns(1)
	users := repository.NewUserRepository(database)
	user := newInventoryTestUser(t, users)
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(context.Background(), models.InventoryArea{Name: "Kitchen", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := inventory.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Soap", TrackingMode: "count", Quantity: 1, LowAt: 1, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHouseholdHandler(repository.NewSettingsRepository(database), nil, nil, repository.NewShoppingRepository(database))
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), middleware.UserContextKey, user)))
		})
	})
	handler.RegisterShopping(router)
	call := func(method, path, body string, status int) models.ShoppingItem {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		if response.Code != status {
			t.Fatalf("%s %s: got %d: %s", method, path, response.Code, response.Body.String())
		}
		var item models.ShoppingItem
		if status == 200 || status == 201 {
			if err := json.Unmarshal(response.Body.Bytes(), &item); err != nil {
				t.Fatal(err)
			}
		}
		return item
	}
	body := fmt.Sprintf(`{"name":"Soap","inventoryItemID":%q}`, stock.ID)
	item := call("POST", "/api/shopping/items", body, 201)
	duplicate := call("POST", "/api/shopping/items", body, 201)
	if duplicate.ID != item.ID {
		t.Fatal("duplicate linked entry")
	}
	purchased := call("PATCH", "/api/shopping/items/"+item.ID, fmt.Sprintf(`{"purchased":true,"version":%d}`, item.Version), 200)
	call("PATCH", "/api/shopping/items/"+item.ID, `{"name":"stale edit","version":0}`, 409)
	restock := fmt.Sprintf(`{"purchaseID":%q,"quantity":3}`, *purchased.PurchaseID)
	call("POST", "/api/shopping/items/"+item.ID+"/restock", restock, 200)
	call("POST", "/api/shopping/items/"+item.ID+"/restock", restock, 200)
	updated, err := inventory.FindItemByID(context.Background(), stock.ID)
	if err != nil || updated.Quantity != 4 {
		t.Fatalf("stock must increase exactly once: %+v %v", updated, err)
	}
	if err := inventory.DeleteArea(context.Background(), area.ID); err != nil {
		t.Fatal(err)
	}
	list, err := handler.shopping.FindAll(context.Background())
	if err != nil || len(list) != 1 || list[0].InventoryItemID != nil {
		t.Fatalf("shopping item must survive inventory deletion: %+v %v", list, err)
	}
}

func TestConcurrentRestockChangesStockOnce(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user := newInventoryTestUser(t, users)
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(context.Background(), models.InventoryArea{Name: "Bathroom", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := inventory.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Soap", TrackingMode: "count", Quantity: 2, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	shopping := repository.NewShoppingRepository(database)
	item, err := shopping.Create(context.Background(), "Soap", &stock.ID)
	if err != nil {
		t.Fatal(err)
	}
	purchased := true
	item, err = shopping.Update(context.Background(), item.ID, models.ShoppingChange{Purchased: &purchased, Version: item.Version})
	if err != nil {
		t.Fatal(err)
	}
	quantity := 4
	failures := make(chan error, 8)
	for index := 0; index < 8; index++ {
		go func() {
			_, err := shopping.Restock(context.Background(), item.ID, models.ShoppingRestock{PurchaseID: *item.PurchaseID, Quantity: &quantity})
			failures <- err
		}()
	}
	for index := 0; index < 8; index++ {
		if err := <-failures; err != nil {
			t.Fatal(err)
		}
	}
	updated, err := inventory.FindItemByID(context.Background(), stock.ID)
	if err != nil || updated.Quantity != 6 {
		t.Fatalf("expected six units after concurrent requests: %+v %v", updated, err)
	}
}

func TestShoppingLevelRestockAndRestore(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	user := newInventoryTestUser(t, repository.NewUserRepository(database))
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(context.Background(), models.InventoryArea{Name: "Laundry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := inventory.CreateItem(context.Background(), models.InventoryItem{AreaID: area.ID, Name: "Detergent", TrackingMode: "level", Level: 10, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	shopping := repository.NewShoppingRepository(database)
	item, err := shopping.Create(context.Background(), stock.Name, &stock.ID)
	if err != nil {
		t.Fatal(err)
	}
	purchased := true
	item, err = shopping.Update(context.Background(), item.ID, models.ShoppingChange{Purchased: &purchased, Version: item.Version})
	if err != nil {
		t.Fatal(err)
	}
	quantity := 2
	if _, err := shopping.Restock(context.Background(), item.ID, models.ShoppingRestock{PurchaseID: *item.PurchaseID, Quantity: &quantity}); !errors.Is(err, repository.ErrShoppingInvalid) {
		t.Fatalf("level item accepted quantity: %v", err)
	}
	invalid := 101
	if _, err := shopping.Restock(context.Background(), item.ID, models.ShoppingRestock{PurchaseID: *item.PurchaseID, Level: &invalid}); !errors.Is(err, repository.ErrShoppingInvalid) {
		t.Fatalf("invalid percentage: %v", err)
	}
	level := 80
	request := models.ShoppingRestock{PurchaseID: *item.PurchaseID, Level: &level}
	item, err = shopping.Restock(context.Background(), item.ID, request)
	if err != nil {
		t.Fatal(err)
	}
	purchased = false
	item, err = shopping.Update(context.Background(), item.ID, models.ShoppingChange{Purchased: &purchased, Version: item.Version})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := shopping.Restock(context.Background(), item.ID, request); !errors.Is(err, repository.ErrShoppingConflict) {
		t.Fatalf("old purchase accepted: %v", err)
	}
	updated, err := inventory.FindItemByID(context.Background(), stock.ID)
	if err != nil || updated.Level != 80 {
		t.Fatalf("restore reversed inventory: %+v %v", updated, err)
	}
}
