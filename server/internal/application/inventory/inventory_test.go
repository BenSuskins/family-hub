package inventory_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

type inventoryFake struct {
	area                  domain.Area
	readError, writeError error
}

func (fake *inventoryFake) FindAllAreas(context.Context) ([]domain.Area, error) {
	return []domain.Area{fake.area}, nil
}
func (fake *inventoryFake) FindAreaByID(context.Context, string) (domain.Area, error) {
	return fake.area, fake.readError
}
func (fake *inventoryFake) CreateArea(context.Context, domain.Area) (domain.Area, error) {
	return domain.Area{}, nil
}
func (fake *inventoryFake) UpdateArea(_ context.Context, area domain.Area) error {
	fake.area = area
	return fake.writeError
}
func (fake *inventoryFake) DeleteArea(context.Context, string) error { return nil }
func (fake *inventoryFake) FindItemByID(context.Context, string) (domain.Item, error) {
	return domain.Item{}, nil
}
func (fake *inventoryFake) CreateItem(context.Context, domain.Item) (domain.Item, error) {
	return domain.Item{}, nil
}
func (fake *inventoryFake) UpdateItem(context.Context, domain.Item) error { return nil }
func (fake *inventoryFake) DeleteItem(context.Context, string) error      { return nil }
func TestAreaUpdatePreservesFallbackAndCopies(t *testing.T) {
	store := &inventoryFake{area: domain.Area{ID: "pantry", Items: []domain.Item{{Name: "Milk"}}}}
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
	saved, err := service.SaveArea(context.Background(), domain.Area{ID: "pantry", Name: "Pantry"})
	if err != nil || saved.Name != "Pantry" {
		t.Fatalf("fallback %+v: %v", saved, err)
	}
}
