package shopping_test

import (
	"errors"
	"testing"

	"github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
)

func TestPurchaseTransition(t *testing.T) {
	original := models.ShoppingItem{ID: "item", Name: "Milk", Version: 4}
	purchased := true
	updated, newPurchase, err := shopping.ApplyChange(original, models.ShoppingChange{Version: 4, Purchased: &purchased}, "purchase")
	if err != nil || !newPurchase || !updated.Purchased || updated.PurchaseID == nil || *updated.PurchaseID != "purchase" {
		t.Fatalf("got %+v, purchase=%v: %v", updated, newPurchase, err)
	}
	if original.Purchased || original.PurchaseID != nil {
		t.Fatal("input changed")
	}
	if _, _, err := shopping.ApplyChange(original, models.ShoppingChange{Version: 3, Purchased: &purchased}, "purchase"); !errors.Is(err, shopping.ErrConflict) {
		t.Fatalf("stale version: %v", err)
	}
}
func TestRestockValidation(t *testing.T) {
	quantity, level := 2, 101
	if err := shopping.ValidateRestock(models.TrackingModeCount, models.ShoppingRestock{Quantity: &quantity}); err != nil {
		t.Fatal(err)
	}
	if err := shopping.ValidateRestock(models.TrackingModeLevel, models.ShoppingRestock{Level: &level}); !errors.Is(err, shopping.ErrInvalid) {
		t.Fatalf("invalid level: %v", err)
	}
}

func TestPurchaseTransitionCopiesReferences(t *testing.T) {
	inventoryID, purchaseID := "stock", "purchase"
	original := models.ShoppingItem{InventoryItemID: &inventoryID, PurchaseID: &purchaseID, Purchased: true, Version: 2}
	updated, _, err := shopping.ApplyChange(original, models.ShoppingChange{Version: 2}, "unused")
	if err != nil {
		t.Fatal(err)
	}
	*updated.InventoryItemID = "other-stock"
	*updated.PurchaseID = "other-purchase"
	if *original.InventoryItemID != "stock" || *original.PurchaseID != "purchase" {
		t.Fatal("transition shared mutable references")
	}
}

func TestRestockState(t *testing.T) {
	purchaseID, inventoryID := "purchase", "stock"
	item := models.ShoppingItem{Purchased: true, PurchaseID: &purchaseID, InventoryItemID: &inventoryID}
	already, err := shopping.CheckRestock(item, models.ShoppingRestock{PurchaseID: purchaseID})
	if err != nil || already {
		t.Fatalf("new restock: %v %v", already, err)
	}
	item.Restocked = true
	already, err = shopping.CheckRestock(item, models.ShoppingRestock{PurchaseID: purchaseID})
	if err != nil || !already {
		t.Fatalf("repeated restock: %v %v", already, err)
	}
	if _, err := shopping.CheckRestock(item, models.ShoppingRestock{PurchaseID: "old"}); !errors.Is(err, shopping.ErrConflict) {
		t.Fatalf("stale purchase: %v", err)
	}
}
