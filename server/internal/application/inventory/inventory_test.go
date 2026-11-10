package inventory_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	"github.com/bensuskins/family-hub/internal/models"
)

type inventoryFake struct {
	area                  models.InventoryArea
	readError, writeError error
}

func (fake *inventoryFake) FindAllAreas(context.Context) ([]models.InventoryArea, error) {
	return []models.InventoryArea{fake.area}, nil
}
func (fake *inventoryFake) FindAreaByID(context.Context, string) (models.InventoryArea, error) {
	return fake.area, fake.readError
}
func (fake *inventoryFake) CreateArea(context.Context, models.InventoryArea) (models.InventoryArea, error) {
	return models.InventoryArea{}, nil
}
func (fake *inventoryFake) UpdateArea(_ context.Context, area models.InventoryArea) error {
	fake.area = area
	return fake.writeError
}
func (fake *inventoryFake) DeleteArea(context.Context, string) error { return nil }
func (fake *inventoryFake) FindItemByID(context.Context, string) (models.InventoryItem, error) {
	return models.InventoryItem{}, nil
}
func (fake *inventoryFake) CreateItem(context.Context, models.InventoryItem) (models.InventoryItem, error) {
	return models.InventoryItem{}, nil
}
func (fake *inventoryFake) UpdateItem(context.Context, models.InventoryItem) error { return nil }
func (fake *inventoryFake) DeleteItem(context.Context, string) error               { return nil }
func TestAreaUpdatePreservesFallbackAndCopies(t *testing.T) {
	store := &inventoryFake{area: models.InventoryArea{ID: "pantry", Items: []models.InventoryItem{{Name: "Milk"}}}}
	service := app.New(store)
	areas, err := service.FindAllAreas(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	areas[0].Items[0].Name = "Changed"
	if store.area.Items[0].Name != "Milk" {
		t.Fatal("snapshot changed storage")
	}
	store.readError = errors.New("read failed")
	saved, err := service.SaveArea(context.Background(), models.InventoryArea{ID: "pantry", Name: "Pantry"})
	if err != nil || saved.Name != "Pantry" {
		t.Fatalf("fallback %+v: %v", saved, err)
	}
}
