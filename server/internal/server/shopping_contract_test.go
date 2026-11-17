package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

func TestOpenAPIShoppingTransitionsContract(t *testing.T) {
	fixture := newContractFixture(t)
	for _, request := range []struct{ method, path, token, body string }{
		{http.MethodGet, "/api/shopping", "", ""},
		{http.MethodGet, "/api/shopping", "display", ""},
		{http.MethodGet, "/api/display/shopping", "member", ""},
		{http.MethodPost, "/api/shopping/items", "", `{"name":"Milk"}`},
	} {
		response := fixture.request(t, request.method, request.path, request.token, request.body, false)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s %q: %d %s", request.method, request.path, request.token, response.Code, response.Body.String())
		}
	}
	list := fixture.request(t, http.MethodGet, "/api/display/shopping", "display", "", true)
	if list.Code != http.StatusOK || list.Body.String() != "[]\n" {
		t.Fatalf("empty display list: %d %s", list.Code, list.Body.String())
	}
	unlinked := fixture.request(t, http.MethodPost, "/api/shopping/items", "member", `{"name":"Note"}`, true)
	if unlinked.Code != http.StatusCreated {
		t.Fatalf("unlinked: %d %s", unlinked.Code, unlinked.Body.String())
	}
	assertShoppingJSON(t, unlinked.Body.Bytes(), true)
	areaResponse := fixture.request(t, http.MethodPost, "/api/inventory/areas", "member", `{"name":"Pantry"}`, true)
	if areaResponse.Code != http.StatusCreated {
		t.Fatalf("area: %d %s", areaResponse.Code, areaResponse.Body.String())
	}
	var area inventorydomain.Area
	if err := json.Unmarshal(areaResponse.Body.Bytes(), &area); err != nil {
		t.Fatal(err)
	}
	stockResponse := fixture.request(t, http.MethodPost, "/api/inventory/areas/"+area.ID+"/items", "member", `{"name":"Milk","trackingMode":"count","quantity":5}`, true)
	if stockResponse.Code != http.StatusCreated {
		t.Fatalf("stock: %d %s", stockResponse.Code, stockResponse.Body.String())
	}
	var stock inventorydomain.Item
	if err := json.Unmarshal(stockResponse.Body.Bytes(), &stock); err != nil {
		t.Fatal(err)
	}
	create := fixture.request(t, http.MethodPost, "/api/shopping/items", "member", fmt.Sprintf(`{"name":"Milk","inventoryItemID":%q}`, stock.ID), true)
	if create.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", create.Code, create.Body.String())
	}
	item := assertShoppingJSON(t, create.Body.Bytes(), false)
	path := "/api/shopping/items/" + item.ID
	noop := fixture.request(t, http.MethodPatch, path, "member", fmt.Sprintf(`{"version":%d}`, item.Version), true)
	if noop.Code != http.StatusOK {
		t.Fatalf("no-op: %d %s", noop.Code, noop.Body.String())
	}
	item = assertShoppingJSON(t, noop.Body.Bytes(), false)
	if item.Version != 1 || item.Purchased || item.PurchaseID != nil {
		t.Fatalf("no-op version: %+v", item)
	}
	stale := fixture.request(t, http.MethodPatch, path, "member", `{"name":"stale","version":0}`, true)
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale version: %d %s", stale.Code, stale.Body.String())
	}
	purchase := fixture.request(t, http.MethodPatch, path, "member", fmt.Sprintf(`{"purchased":true,"version":%d}`, item.Version), true)
	if purchase.Code != http.StatusOK {
		t.Fatalf("purchase: %d %s", purchase.Code, purchase.Body.String())
	}
	item = assertShoppingJSON(t, purchase.Body.Bytes(), false)
	if !item.Purchased || item.PurchaseID == nil || item.Version != 2 {
		t.Fatalf("purchase: %+v", item)
	}
	restockBody := fmt.Sprintf(`{"purchaseID":%q,"quantity":2}`, *item.PurchaseID)
	staleID := fixture.request(t, http.MethodPost, path+"/restock", "member", `{"purchaseID":"stale","quantity":2}`, true)
	if staleID.Code != http.StatusConflict {
		t.Fatalf("stale purchase: %d %s", staleID.Code, staleID.Body.String())
	}
	for index := 0; index < 2; index++ {
		response := fixture.request(t, http.MethodPost, path+"/restock", "member", restockBody, true)
		if response.Code != http.StatusOK {
			t.Fatalf("restock %d: %d %s", index, response.Code, response.Body.String())
		}
		item = assertShoppingJSON(t, response.Body.Bytes(), false)
		if !item.Restocked || item.Version != 3 {
			t.Fatalf("restock %d: %+v", index, item)
		}
	}
	inventory := fixture.request(t, http.MethodGet, "/api/inventory", "member", "", true)
	if inventory.Code != http.StatusOK {
		t.Fatalf("inventory: %d %s", inventory.Code, inventory.Body.String())
	}
	var areas []inventorydomain.Area
	if err := json.Unmarshal(inventory.Body.Bytes(), &areas); err != nil || len(areas) != 1 || len(areas[0].Items) != 1 || areas[0].Items[0].Quantity != 7 {
		t.Fatalf("inventory after repeated restock: %s, %v", inventory.Body.String(), err)
	}
}

type shoppingContractItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	InventoryItemID *string `json:"inventoryItemID"`
	Purchased       bool    `json:"purchased"`
	PurchaseID      *string `json:"purchaseID"`
	Restocked       bool    `json:"restocked"`
	Version         int     `json:"version"`
}

func assertShoppingJSON(t *testing.T, data []byte, unlinked bool) shoppingContractItem {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	keys := []string{"id", "name", "inventoryItemID", "purchased", "purchaseID", "restocked", "version"}
	if len(fields) != len(keys) {
		t.Fatalf("unexpected shopping fields: %s", data)
	}
	for _, key := range keys {
		if _, ok := fields[key]; !ok {
			t.Fatalf("missing %q: %s", key, data)
		}
	}
	if unlinked && (string(fields["inventoryItemID"]) != "null" || string(fields["purchaseID"]) != "null") {
		t.Fatalf("unlinked nulls: %s", data)
	}
	var result shoppingContractItem
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
