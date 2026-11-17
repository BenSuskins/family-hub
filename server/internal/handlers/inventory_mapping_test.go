package handlers

import (
	"encoding/json"
	"reflect"
	"testing"

	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

func TestInventoryResponseMappingPreservesLegacyJSON(t *testing.T) {
	response := inventoryAreaResponseFor(inventorydomain.Area{ID: "a", Name: "Pantry", Items: []inventorydomain.Item{{ID: "i", AreaID: "a", Name: "Milk", TrackingMode: "count", Quantity: 4}}})
	data, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var area map[string]json.RawMessage
	if err := json.Unmarshal(data, &area); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ID", "Name", "Icon", "Tint", "Items", "CreatedByUserID", "CreatedAt", "UpdatedAt"} {
		if _, ok := area[key]; !ok {
			t.Fatalf("missing area field %s: %s", key, data)
		}
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(area["Items"], &items); err != nil {
		t.Fatal(err)
	}
	expected := []string{"ID", "AreaID", "Name", "TrackingMode", "Quantity", "Level", "Unit", "LowAt", "CreatedByUserID", "CreatedAt", "UpdatedAt"}
	if len(items) != 1 || len(items[0]) != len(expected) {
		t.Fatalf("unexpected items: %s", area["Items"])
	}
	for _, key := range expected {
		if _, ok := items[0][key]; !ok {
			t.Fatalf("missing item field %s: %s", key, area["Items"])
		}
	}
	if string(items[0]["Quantity"]) != "4" {
		t.Fatalf("quantity: %s", items[0]["Quantity"])
	}
}
func TestInventoryResponseMappingPreservesEmptyAndNilCollections(t *testing.T) {
	if got := inventoryAreaResponses(nil); !reflect.DeepEqual(got, []inventoryAreaResponse{}) {
		t.Fatalf("areas = %#v", got)
	}
	empty := inventoryAreaResponseFor(inventorydomain.Area{Items: []inventorydomain.Item{}})
	if empty.Items == nil {
		t.Fatal("empty items became nil")
	}
	if inventoryAreaResponseFor(inventorydomain.Area{}).Items != nil {
		t.Fatal("nil items became empty")
	}
}
