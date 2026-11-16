package repository_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/shopping"
	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/google/uuid"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type shoppingRollbackFixture struct {
	database  *sql.DB
	shopping  *app.Service
	inventory *repository.SQLiteInventoryRepository
	item      domain.Item
	stock     models.InventoryItem
}

func newShoppingRollbackFixture(t *testing.T) shoppingRollbackFixture {
	t.Helper()
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "rollback-user", Email: "rollback@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(ctx, models.InventoryArea{Name: "Pantry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := inventory.CreateItem(ctx, models.InventoryItem{AreaID: area.ID, Name: "Milk", TrackingMode: models.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	shopping := app.New(repository.NewShoppingRepository(database, repository.NewShoppingInventory), uuid.NewString, time.Now)
	item, err := shopping.Create(ctx, "Milk", &stock.ID)
	if err != nil {
		t.Fatal(err)
	}
	return shoppingRollbackFixture{database: database, shopping: shopping, inventory: inventory, item: item, stock: stock}
}

func (fixture shoppingRollbackFixture) assertItem(t *testing.T, want domain.Item) {
	t.Helper()
	items, err := fixture.shopping.List(context.Background())
	if err != nil || len(items) != 1 {
		t.Fatalf("shopping items = %+v, %v", items, err)
	}
	item := items[0]
	if item.ID != want.ID || item.Purchased != want.Purchased || item.Restocked != want.Restocked || item.Version != want.Version || (item.PurchaseID == nil) != (want.PurchaseID == nil) {
		t.Fatalf("shopping item = %+v, want %+v", item, want)
	}
	if item.PurchaseID != nil && *item.PurchaseID != *want.PurchaseID {
		t.Fatalf("purchase ID = %q, want %q", *item.PurchaseID, *want.PurchaseID)
	}
}

func TestShoppingPurchaseInsertRollsBackWhenItemUpdateFails(t *testing.T) {
	fixture := newShoppingRollbackFixture(t)
	ctx := context.Background()
	_, err := fixture.database.ExecContext(ctx, `CREATE TRIGGER fail_shopping_update BEFORE UPDATE ON shopping_items BEGIN SELECT RAISE(ABORT, 'injected shopping update failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	purchased := true
	if _, err := fixture.shopping.Update(ctx, fixture.item.ID, domain.Change{Purchased: &purchased, Version: fixture.item.Version}); err == nil || !strings.Contains(err.Error(), "injected shopping update failure") {
		t.Fatalf("update error = %v", err)
	}
	fixture.assertItem(t, fixture.item)
	var purchases int
	if err := fixture.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM shopping_purchases WHERE shopping_item_id = ?`, fixture.item.ID).Scan(&purchases); err != nil || purchases != 0 {
		t.Fatalf("purchase count = %d, %v", purchases, err)
	}
	if _, err := fixture.database.ExecContext(ctx, `DROP TRIGGER fail_shopping_update`); err != nil {
		t.Fatal(err)
	}
	item, err := fixture.shopping.Update(ctx, fixture.item.ID, domain.Change{Purchased: &purchased, Version: fixture.item.Version})
	if err != nil || !item.Purchased || item.PurchaseID == nil || item.Version != fixture.item.Version+1 {
		t.Fatalf("retry = %+v, %v", item, err)
	}
	if err := fixture.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM shopping_purchases WHERE shopping_item_id = ?`, fixture.item.ID).Scan(&purchases); err != nil || purchases != 1 {
		t.Fatalf("purchase count after retry = %d, %v", purchases, err)
	}
}

func TestShoppingRestockRollsBackAtEachWriteBoundary(t *testing.T) {
	for _, test := range []struct{ name, trigger string }{
		{"purchase claim", `CREATE TRIGGER fail_restock BEFORE UPDATE ON shopping_purchases BEGIN SELECT RAISE(ABORT, 'injected restock failure'); END`},
		{"shopping item", `CREATE TRIGGER fail_restock BEFORE UPDATE ON shopping_items BEGIN SELECT RAISE(ABORT, 'injected restock failure'); END`},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newShoppingRollbackFixture(t)
			ctx := context.Background()
			purchased := true
			item, err := fixture.shopping.Update(ctx, fixture.item.ID, domain.Change{Purchased: &purchased, Version: fixture.item.Version})
			if err != nil || item.PurchaseID == nil {
				t.Fatalf("purchase = %+v, %v", item, err)
			}
			if _, err := fixture.database.ExecContext(ctx, test.trigger); err != nil {
				t.Fatal(err)
			}
			quantity := 3
			request := domain.Restock{PurchaseID: *item.PurchaseID, Quantity: &quantity}
			if _, err := fixture.shopping.Restock(ctx, item.ID, request); err == nil || !strings.Contains(err.Error(), "injected restock failure") {
				t.Fatalf("restock error = %v", err)
			}
			stock, err := fixture.inventory.FindItemByID(ctx, fixture.stock.ID)
			if err != nil || stock.Quantity != fixture.stock.Quantity {
				t.Fatalf("stock after rollback = %+v, %v", stock, err)
			}
			fixture.assertItem(t, item)
			var unclaimed bool
			if err := fixture.database.QueryRowContext(ctx, `SELECT restocked_at IS NULL FROM shopping_purchases WHERE id = ?`, *item.PurchaseID).Scan(&unclaimed); err != nil || !unclaimed {
				t.Fatalf("purchase unclaimed = %v, %v", unclaimed, err)
			}
			if _, err := fixture.database.ExecContext(ctx, `DROP TRIGGER fail_restock`); err != nil {
				t.Fatal(err)
			}
			restocked, err := fixture.shopping.Restock(ctx, item.ID, request)
			if err != nil || !restocked.Restocked || restocked.Version != item.Version+1 {
				t.Fatalf("retry = %+v, %v", restocked, err)
			}
			stock, err = fixture.inventory.FindItemByID(ctx, fixture.stock.ID)
			if err != nil || stock.Quantity != fixture.stock.Quantity+quantity {
				t.Fatalf("stock after retry = %+v, %v", stock, err)
			}
			if err := fixture.database.QueryRowContext(ctx, `SELECT restocked_at IS NULL FROM shopping_purchases WHERE id = ?`, *item.PurchaseID).Scan(&unclaimed); err != nil || unclaimed {
				t.Fatalf("purchase unclaimed after retry = %v, %v", unclaimed, err)
			}
		})
	}
}
