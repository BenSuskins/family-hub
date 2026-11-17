package repository_test

import (
	"context"
	"errors"
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"testing"
	"time"

	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestShoppingInventoryUsesCallerTransaction(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "inventory-transaction", Email: "inventory-transaction@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(ctx, inventorydomain.Area{Name: "Pantry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: "Rice", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 4, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := repository.NewShoppingInventory(transaction)
	mode, err := adapter.TrackingMode(ctx, item.ID)
	if err != nil || mode != inventorydomain.TrackingModeCount {
		t.Fatalf("mode %q: %v", mode, err)
	}
	if _, err = adapter.TrackingMode(ctx, "missing"); !errors.Is(err, shoppingdomain.ErrNotFound) {
		t.Fatalf("missing item: %v", err)
	}
	quantity := 3
	if err = adapter.ApplyRestock(ctx, item.ID, mode, &quantity, nil, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	var inTransaction int
	if err = transaction.QueryRowContext(ctx, "SELECT quantity FROM inventory_items WHERE id=?", item.ID).Scan(&inTransaction); err != nil || inTransaction != 7 {
		t.Fatalf("in transaction %d: %v", inTransaction, err)
	}
	if err = transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	saved, err := inventory.FindItemByID(ctx, item.ID)
	if err != nil || saved.Quantity != 4 {
		t.Fatalf("after rollback %+v: %v", saved, err)
	}
}

func TestShoppingInventoryRestockPreservesLegacyModeHandling(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "inventory-level", Email: "inventory-level@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(ctx, inventorydomain.Area{Name: "Pantry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	item, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: "Soap", TrackingMode: inventorydomain.TrackingModeLevel, Level: 20, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	adapter := repository.NewShoppingInventory(transaction)
	level := 80
	if err := adapter.ApplyRestock(ctx, item.ID, "legacy-mode", nil, &level, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := transaction.Commit(); err != nil {
		t.Fatal(err)
	}
	saved, err := inventory.FindItemByID(ctx, item.ID)
	if err != nil || saved.Level != 80 {
		t.Fatalf("saved %+v: %v", saved, err)
	}
}
