package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAPIInventoryBoundaryContract(t *testing.T) {
	fixture := newContractFixture(t)
	request := func(method, path, token, body string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr = "198.19.17.23:1234"
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		out := httptest.NewRecorder()
		fixture.application.router.ServeHTTP(out, r)
		return out
	}
	for _, example := range []struct {
		path, token string
		status      int
	}{
		{"/api/inventory", "", 401}, {"/api/inventory", "display", 401}, {"/api/display/inventory", "member", 401},
	} {
		response := request(http.MethodGet, example.path, example.token, "")
		if response.Code != example.status {
			t.Fatalf("%s token %q: %d %s", example.path, example.token, response.Code, response.Body.String())
		}
	}
	empty := fixture.request(t, http.MethodGet, "/api/inventory", "member", "", true)
	if empty.Code != http.StatusOK || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("empty inventory: %d %s", empty.Code, empty.Body.String())
	}
	areaResponse := fixture.request(t, http.MethodPost, "/api/inventory/areas", "member", `{"name":"Pantry"}`, true)
	if areaResponse.Code != http.StatusCreated {
		t.Fatalf("create area: %d %s", areaResponse.Code, areaResponse.Body.String())
	}
	var area map[string]json.RawMessage
	if err := json.Unmarshal(areaResponse.Body.Bytes(), &area); err != nil {
		t.Fatal(err)
	}
	fields := []string{"ID", "Name", "Icon", "Tint", "Items", "CreatedByUserID", "CreatedAt", "UpdatedAt"}
	if len(area) != len(fields) {
		t.Fatalf("area keys %s", areaResponse.Body.String())
	}
	for _, field := range fields {
		if _, ok := area[field]; !ok {
			t.Fatalf("missing %s: %s", field, areaResponse.Body.String())
		}
	}
	if string(area["Items"]) != "[]" || string(area["Icon"]) != `"box"` || string(area["Tint"]) != `"blue"` {
		t.Fatalf("area defaults %s", areaResponse.Body.String())
	}
	var areaID string
	if err := json.Unmarshal(area["ID"], &areaID); err != nil {
		t.Fatal(err)
	}
	itemResponse := fixture.request(t, http.MethodPost, "/api/inventory/areas/"+areaID+"/items", "member", `{"name":"Milk","trackingMode":"count","quantity":3}`, true)
	if itemResponse.Code != http.StatusCreated {
		t.Fatalf("create item: %d %s", itemResponse.Code, itemResponse.Body.String())
	}
	var item map[string]json.RawMessage
	if err := json.Unmarshal(itemResponse.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	itemFields := []string{"ID", "AreaID", "Name", "TrackingMode", "Quantity", "Level", "Unit", "LowAt", "CreatedByUserID", "CreatedAt", "UpdatedAt"}
	if len(item) != len(itemFields) {
		t.Fatalf("item keys %s", itemResponse.Body.String())
	}
	for _, field := range itemFields {
		if _, ok := item[field]; !ok {
			t.Fatalf("missing %s: %s", field, itemResponse.Body.String())
		}
	}
	var itemID string
	if err := json.Unmarshal(item["ID"], &itemID); err != nil {
		t.Fatal(err)
	}
	display := fixture.request(t, http.MethodGet, "/api/display/inventory", "display", "", true)
	if display.Code != http.StatusOK || !strings.Contains(display.Body.String(), `"Name":"Milk"`) {
		t.Fatalf("display: %d %s", display.Code, display.Body.String())
	}
	for _, example := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPut, "/api/inventory/areas/missing", `{`, 404},
		{http.MethodPost, "/api/inventory/areas/missing/items", `{`, 404},
		{http.MethodPut, "/api/inventory/items/missing", `{`, 404},
		{http.MethodPatch, "/api/inventory/items/missing/stock", `{}`, 400},
		{http.MethodPatch, "/api/inventory/items/missing/stock", `{"quantity":1}`, 404},
		{http.MethodPatch, "/api/inventory/items/" + itemID + "/stock", `{"quantity":-1}`, 400},
	} {
		response := request(example.method, example.path, "member", example.body)
		if response.Code != example.status {
			t.Fatalf("%s %s: got %d want %d: %s", example.method, example.path, response.Code, example.status, response.Body.String())
		}
	}
	displayUpdate := fixture.request(t, http.MethodPut, "/api/display/inventory/items/"+itemID+"/stock", "display", `{"quantityDelta":-2}`, true)
	if displayUpdate.Code != http.StatusOK {
		t.Fatalf("display stock: %d %s", displayUpdate.Code, displayUpdate.Body.String())
	}
	if err := json.Unmarshal(displayUpdate.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if string(item["Quantity"]) != "1" {
		t.Fatalf("display stock result: %s", displayUpdate.Body.String())
	}
	updated := fixture.request(t, http.MethodPut, "/api/inventory/areas/"+areaID, "member", `{"name":"Renamed"}`, true)
	if updated.Code != http.StatusOK {
		t.Fatalf("update area: %d %s", updated.Code, updated.Body.String())
	}
	if err := json.Unmarshal(updated.Body.Bytes(), &area); err != nil {
		t.Fatal(err)
	}
	if string(area["Icon"]) != `"box"` || string(area["Tint"]) != `"blue"` {
		t.Fatalf("empty icon/tint must preserve values: %s", updated.Body.String())
	}
}
