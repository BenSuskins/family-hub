package shopping_test

import (
	"errors"
	"testing"

	shopping "github.com/bensuskins/family-hub/internal/domain/shopping"
)

func TestAggregateRestorationAndSnapshots(t *testing.T) {
	inventoryID, purchaseID := "stock", "purchase"
	stored := shopping.Item{ID: "item", Name: "Milk", InventoryItemID: &inventoryID, Purchased: true, PurchaseID: &purchaseID}
	aggregate, err := shopping.Restore(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := aggregate.Snapshot()
	*snapshot.InventoryItemID = "other"
	*snapshot.PurchaseID = "other"
	if *stored.InventoryItemID != "stock" || *aggregate.Snapshot().PurchaseID != "purchase" {
		t.Fatal("snapshot shared stored values")
	}
	legacy := shopping.Item{ID: "legacy", Name: "Milk", Restocked: true}
	if _, err := shopping.Restore(legacy); !errors.Is(err, shopping.ErrInvalidState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	restored := shopping.RestoreLegacy(legacy)
	if got := restored.Snapshot(); got != legacy {
		t.Fatalf("legacy state changed: %+v", got)
	}
}

func TestAggregateTransitionsAreImmutable(t *testing.T) {
	aggregate, err := shopping.New("item", "  Milk  ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.Snapshot().Name != "Milk" {
		t.Fatal("name was not normalized")
	}
	purchased := true
	updated, created, err := aggregate.Change(shopping.Change{Purchased: &purchased}, "purchase")
	if err != nil || !created || !updated.Snapshot().Purchased || aggregate.Snapshot().Purchased {
		t.Fatalf("purchase transition: %v %v", created, err)
	}
	saved := updated.Saved()
	if saved.Snapshot().Version != 1 || updated.Snapshot().Version != 0 {
		t.Fatal("save changed original")
	}
	if _, _, err := saved.Change(shopping.Change{Version: 0}, "unused"); !errors.Is(err, shopping.ErrConflict) {
		t.Fatalf("stale version: %v", err)
	}
}
func TestLegacyAggregateKeepsRestockSemantics(t *testing.T) {
	purchase := "purchase"
	legacy := shopping.RestoreLegacy(shopping.Item{ID: "old", Name: "", Purchased: true, PurchaseID: &purchase, Restocked: true})
	already, err := legacy.CheckRestock(shopping.Restock{PurchaseID: purchase})
	if err != nil || !already {
		t.Fatalf("legacy restock: %v %v", already, err)
	}
	if _, err := legacy.CheckRestock(shopping.Restock{PurchaseID: "stale"}); !errors.Is(err, shopping.ErrConflict) {
		t.Fatalf("stale purchase: %v", err)
	}
}

func TestAggregateRejectsInvalidRestockTransition(t *testing.T) {
	aggregate, err := shopping.New("item", "Milk", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := aggregate.Restocked(shopping.Restock{PurchaseID: "missing"}); !errors.Is(err, shopping.ErrConflict) {
		t.Fatalf("got %v", err)
	}
}
