package repository_test

import (
	"context"
	"fmt"
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"testing"
	"time"

	inventoryapp "github.com/bensuskins/family-hub/internal/application/inventory"
	shoppingapp "github.com/bensuskins/family-hub/internal/application/shopping"
	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

func TestRestockAndStockChangePreserveBothUpdates(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "restock-stock-race", Email: "restock-stock@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(ctx, inventorydomain.Area{Name: "Pantry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	shopping := shoppingapp.New(repository.NewShoppingRepository(database, repository.NewShoppingInventory), uuid.NewString, time.Now)
	stock := inventoryapp.NewStockService(inventory)
	for iteration := range 8 {
		item, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: fmt.Sprintf("Item %d", iteration), TrackingMode: inventorydomain.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
		if err != nil {
			t.Fatal(err)
		}
		entry, err := shopping.Create(ctx, item.Name, &item.ID)
		if err != nil {
			t.Fatal(err)
		}
		purchased := true
		entry, err = shopping.Update(ctx, entry.ID, shoppingdomain.Change{Version: entry.Version, Purchased: &purchased})
		if err != nil || entry.PurchaseID == nil {
			t.Fatalf("purchase %+v: %v", entry, err)
		}
		quantity, delta := 2, int64(3)
		request := shoppingdomain.Restock{PurchaseID: *entry.PurchaseID, Quantity: &quantity}
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() { <-start; _, err := shopping.Restock(ctx, entry.ID, request); results <- err }()
		go func() {
			<-start
			_, err := stock.Update(ctx, item.ID, inventorydomain.StockChange{QuantityDelta: &delta})
			results <- err
		}()
		close(start)
		for range 2 {
			select {
			case err := <-results:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatalf("concurrent operations did not finish: %v", ctx.Err())
			}
		}
		updated, err := inventory.FindItemByID(ctx, item.ID)
		if err != nil || updated.Quantity != 10 {
			t.Fatalf("stock %+v: %v", updated, err)
		}
		if _, err := shopping.Restock(ctx, entry.ID, request); err != nil {
			t.Fatal(err)
		}
		updated, err = inventory.FindItemByID(ctx, item.ID)
		if err != nil || updated.Quantity != 10 {
			t.Fatalf("repeated restock changed stock %+v: %v", updated, err)
		}
	}
}
