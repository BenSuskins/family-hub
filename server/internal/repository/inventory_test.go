package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestInventoryRepository_AreaCRUD(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, err := invRepo.CreateArea(ctx, models.InventoryArea{
		Name:            "Laundry cupboard",
		Icon:            "drop",
		Tint:            "blue",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating area: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.Items == nil {
		t.Fatal("expected non-nil Items slice")
	}

	found, err := invRepo.FindAreaByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding area: %v", err)
	}
	if found.Name != "Laundry cupboard" || found.Icon != "drop" || found.Tint != "blue" {
		t.Errorf("unexpected area: %+v", found)
	}

	found.Name = "Utility room"
	found.Icon = "sparkles"
	found.Tint = "green"
	if err := invRepo.UpdateArea(ctx, found); err != nil {
		t.Fatalf("updating area: %v", err)
	}
	updated, err := invRepo.FindAreaByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("re-finding area: %v", err)
	}
	if updated.Name != "Utility room" || updated.Icon != "sparkles" || updated.Tint != "green" {
		t.Errorf("update not persisted: %+v", updated)
	}

	if err := invRepo.DeleteArea(ctx, created.ID); err != nil {
		t.Fatalf("deleting area: %v", err)
	}
	if _, err := invRepo.FindAreaByID(ctx, created.ID); err == nil {
		t.Fatal("expected error finding deleted area")
	}
}

func TestInventoryRepository_CreateAreaDefaults(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, err := invRepo.CreateArea(ctx, models.InventoryArea{
		Name:            "Shed",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating area: %v", err)
	}
	if created.Icon != "box" || created.Tint != "blue" {
		t.Errorf("expected default icon 'box' and tint 'blue', got icon=%q tint=%q", created.Icon, created.Tint)
	}
}

func TestInventoryRepository_ItemCRUD(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	area, err := invRepo.CreateArea(ctx, models.InventoryArea{Name: "Bathroom", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatalf("creating area: %v", err)
	}

	item, err := invRepo.CreateItem(ctx, models.InventoryItem{
		AreaID:          area.ID,
		Name:            "Toilet roll",
		Quantity:        12,
		Unit:            "rolls",
		Par:             8,
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating item: %v", err)
	}
	if item.ID == "" {
		t.Fatal("expected non-empty item ID")
	}

	found, err := invRepo.FindItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("finding item: %v", err)
	}
	if found.Quantity != 12 || found.Unit != "rolls" || found.Par != 8 {
		t.Errorf("unexpected item: %+v", found)
	}

	found.Quantity = 3
	if err := invRepo.UpdateItem(ctx, found); err != nil {
		t.Fatalf("updating item: %v", err)
	}
	updated, err := invRepo.FindItemByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("re-finding item: %v", err)
	}
	if updated.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", updated.Quantity)
	}

	if err := invRepo.DeleteItem(ctx, item.ID); err != nil {
		t.Fatalf("deleting item: %v", err)
	}
	if _, err := invRepo.FindItemByID(ctx, item.ID); err == nil {
		t.Fatal("expected error finding deleted item")
	}
}

func TestInventoryRepository_FindAllAreasNestsItems(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	laundry, _ := invRepo.CreateArea(ctx, models.InventoryArea{Name: "Laundry", CreatedByUserID: user.ID})
	pantry, _ := invRepo.CreateArea(ctx, models.InventoryArea{Name: "Pantry", CreatedByUserID: user.ID})

	if _, err := invRepo.CreateItem(ctx, models.InventoryItem{AreaID: laundry.ID, Name: "Pods", Quantity: 45, Unit: "pods", Par: 20, CreatedByUserID: user.ID}); err != nil {
		t.Fatalf("creating item: %v", err)
	}
	if _, err := invRepo.CreateItem(ctx, models.InventoryItem{AreaID: pantry.ID, Name: "Pasta", Quantity: 6, Unit: "packs", Par: 3, CreatedByUserID: user.ID}); err != nil {
		t.Fatalf("creating item: %v", err)
	}
	if _, err := invRepo.CreateItem(ctx, models.InventoryItem{AreaID: pantry.ID, Name: "Olive oil", Quantity: 1, Unit: "bottles", Par: 2, CreatedByUserID: user.ID}); err != nil {
		t.Fatalf("creating item: %v", err)
	}

	areas, err := invRepo.FindAllAreas(ctx)
	if err != nil {
		t.Fatalf("finding all areas: %v", err)
	}
	if len(areas) != 2 {
		t.Fatalf("expected 2 areas, got %d", len(areas))
	}

	byName := map[string]models.InventoryArea{}
	for _, a := range areas {
		byName[a.Name] = a
	}
	if got := len(byName["Laundry"].Items); got != 1 {
		t.Errorf("expected 1 laundry item, got %d", got)
	}
	if got := len(byName["Pantry"].Items); got != 2 {
		t.Errorf("expected 2 pantry items, got %d", got)
	}
}

func TestInventoryRepository_DeleteAreaCascadesItems(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	area, _ := invRepo.CreateArea(ctx, models.InventoryArea{Name: "Cleaning closet", CreatedByUserID: user.ID})
	item, err := invRepo.CreateItem(ctx, models.InventoryItem{AreaID: area.ID, Name: "Bin bags", Quantity: 40, Unit: "bags", Par: 20, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatalf("creating item: %v", err)
	}

	if err := invRepo.DeleteArea(ctx, area.ID); err != nil {
		t.Fatalf("deleting area: %v", err)
	}
	if _, err := invRepo.FindItemByID(ctx, item.ID); err == nil {
		t.Fatal("expected item to be cascade-deleted with its area")
	}
}

func TestInventoryRepository_QuantityClampsAtZero(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	invRepo := repository.NewInventoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	area, _ := invRepo.CreateArea(ctx, models.InventoryArea{Name: "Shed", CreatedByUserID: user.ID})

	created, err := invRepo.CreateItem(ctx, models.InventoryItem{AreaID: area.ID, Name: "Screws", Quantity: -5, Par: -1, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatalf("creating item: %v", err)
	}
	if created.Quantity != 0 || created.Par != 0 {
		t.Errorf("expected negatives clamped to 0, got quantity=%d par=%d", created.Quantity, created.Par)
	}

	created.Quantity = -3
	if err := invRepo.UpdateItem(ctx, created); err != nil {
		t.Fatalf("updating item: %v", err)
	}
	updated, _ := invRepo.FindItemByID(ctx, created.ID)
	if updated.Quantity != 0 {
		t.Errorf("expected quantity clamped to 0 on update, got %d", updated.Quantity)
	}
}
