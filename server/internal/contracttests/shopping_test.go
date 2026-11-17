package contracttests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"

	app "github.com/bensuskins/family-hub/internal/application/shopping"
	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

type fakeShoppingState struct {
	items     map[string]domain.Item
	created   map[string]string
	stocks    map[string]inventorydomain.Item
	purchases map[string]fakePurchase
}
type fakePurchase struct {
	owner   string
	claimed bool
}
type shoppingFake struct {
	mutex  sync.Mutex
	state  fakeShoppingState
	failAt string
}
type shoppingTransaction struct {
	state  *fakeShoppingState
	failAt string
}

func newShoppingFake(stock ...inventorydomain.Item) *shoppingFake {
	state := fakeShoppingState{items: map[string]domain.Item{}, created: map[string]string{}, stocks: map[string]inventorydomain.Item{}, purchases: map[string]fakePurchase{}}
	for _, item := range stock {
		state.stocks[item.ID] = item
	}
	return &shoppingFake{state: state}
}
func copyShopping(item domain.Item) domain.Item {
	if item.InventoryItemID != nil {
		v := *item.InventoryItemID
		item.InventoryItemID = &v
	}
	if item.PurchaseID != nil {
		v := *item.PurchaseID
		item.PurchaseID = &v
	}
	return item
}
func cloneShoppingState(source fakeShoppingState) fakeShoppingState {
	result := fakeShoppingState{items: map[string]domain.Item{}, created: map[string]string{}, stocks: map[string]inventorydomain.Item{}, purchases: map[string]fakePurchase{}}
	for id, item := range source.items {
		result.items[id] = copyShopping(item)
	}
	for id, created := range source.created {
		result.created[id] = created
	}
	for id, stock := range source.stocks {
		result.stocks[id] = stock
	}
	for id, purchase := range source.purchases {
		result.purchases[id] = purchase
	}
	return result
}
func (fake *shoppingFake) FindAll(ctx context.Context) ([]domain.Item, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := make([]domain.Item, 0, len(fake.state.items))
	for _, item := range fake.state.items {
		items = append(items, copyShopping(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Purchased != items[j].Purchased {
			return !items[i].Purchased
		}
		a, b := fake.state.created[items[i].ID], fake.state.created[items[j].ID]
		if a != b {
			return a < b
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}
func (fake *shoppingFake) Delete(ctx context.Context, id string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := fake.state.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(fake.state.items, id)
	delete(fake.state.created, id)
	for purchaseID, purchase := range fake.state.purchases {
		if purchase.owner == id {
			delete(fake.state.purchases, purchaseID)
		}
	}
	return nil
}
func (fake *shoppingFake) ClearPurchased(ctx context.Context) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for id, item := range fake.state.items {
		if item.Purchased {
			delete(fake.state.items, id)
			delete(fake.state.created, id)
			for purchaseID, purchase := range fake.state.purchases {
				if purchase.owner == id {
					delete(fake.state.purchases, purchaseID)
				}
			}
		}
	}
	return nil
}
func (fake *shoppingFake) Transact(ctx context.Context, work func(app.Transaction) (bool, error)) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	state := cloneShoppingState(fake.state)
	commit, err := work(&shoppingTransaction{state: &state, failAt: fake.failAt})
	if err != nil {
		return err
	}
	if !commit {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	fake.state = state
	return nil
}
func (tx *shoppingTransaction) Find(ctx context.Context, id string) (domain.Aggregate, error) {
	if err := ctx.Err(); err != nil {
		return domain.Aggregate{}, err
	}
	item, ok := tx.state.items[id]
	if !ok {
		return domain.Aggregate{}, domain.ErrNotFound
	}
	aggregate, err := domain.Restore(item)
	if err != nil {
		return domain.RestoreLegacy(item), nil
	}
	return aggregate, nil
}

func (tx *shoppingTransaction) FindActive(ctx context.Context, id string) (domain.Item, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, false, err
	}
	for _, item := range tx.state.items {
		if !item.Purchased && item.InventoryItemID != nil && *item.InventoryItemID == id {
			return copyShopping(item), true, nil
		}
	}
	return domain.Item{}, false, nil
}
func (tx *shoppingTransaction) Insert(ctx context.Context, item domain.Item) (domain.Item, error) {
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	if _, exists := tx.state.items[item.ID]; exists {
		return domain.Item{}, domain.ErrConflict
	}
	if item.InventoryItemID != nil {
		if _, exists := tx.state.stocks[*item.InventoryItemID]; !exists {
			return domain.Item{}, errors.New("foreign key constraint")
		}
		if _, found, err := tx.FindActive(ctx, *item.InventoryItemID); err != nil {
			return domain.Item{}, err
		} else if found {
			return domain.Item{}, domain.ErrConflict
		}
	}
	item.Purchased = false
	item.PurchaseID = nil
	item.Restocked = false
	item.Version = 0
	tx.state.items[item.ID] = copyShopping(item)
	tx.state.created[item.ID] = time.Now().UTC().Format("2006-01-02 15:04:05")
	return copyShopping(item), nil
}
func (tx *shoppingTransaction) InsertPurchase(ctx context.Context, purchaseID, itemID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := tx.state.purchases[purchaseID]; ok {
		return errors.New("duplicate purchase")
	}
	if _, ok := tx.state.items[itemID]; !ok {
		return errors.New("foreign key constraint")
	}
	tx.state.purchases[purchaseID] = fakePurchase{owner: itemID}
	return nil
}
func (tx *shoppingTransaction) Save(ctx context.Context, item domain.Item) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx.failAt == "save" {
		return fmt.Errorf("injected save failure")
	}
	current, ok := tx.state.items[item.ID]
	if !ok || current.Version != item.Version {
		return domain.ErrConflict
	}
	item.InventoryItemID = current.InventoryItemID
	if !item.Purchased && item.InventoryItemID != nil {
		for id, other := range tx.state.items {
			if id != item.ID && !other.Purchased && other.InventoryItemID != nil && *other.InventoryItemID == *item.InventoryItemID {
				return domain.ErrConflict
			}
		}
	}
	item.Version++
	tx.state.items[item.ID] = copyShopping(item)
	return nil
}
func (tx *shoppingTransaction) Claim(ctx context.Context, purchaseID string, at time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if tx.failAt == "claim" {
		return false, fmt.Errorf("injected claim failure")
	}
	purchase, ok := tx.state.purchases[purchaseID]
	if !ok || purchase.claimed {
		return false, nil
	}
	purchase.claimed = true
	tx.state.purchases[purchaseID] = purchase
	return true, nil
}
func (tx *shoppingTransaction) MarkRestocked(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if tx.failAt == "mark" {
		return fmt.Errorf("injected mark failure")
	}
	item, ok := tx.state.items[id]
	if ok {
		item.Restocked = true
		item.Version++
		tx.state.items[id] = item
	}
	return nil
}
func (tx *shoppingTransaction) Inventory() app.Inventory { return tx }
func (tx *shoppingTransaction) TrackingMode(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	stock, ok := tx.state.stocks[id]
	if !ok {
		return "", domain.ErrNotFound
	}
	return stock.TrackingMode, nil
}
func (tx *shoppingTransaction) ApplyRestock(ctx context.Context, id, mode string, quantity, level *int, at time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stock, ok := tx.state.stocks[id]
	if !ok {
		return nil
	}
	if mode == inventorydomain.TrackingModeCount {
		stock.Quantity += *quantity
	} else {
		stock.Level = *level
	}
	tx.state.stocks[id] = stock
	return nil
}

type shoppingFixture struct {
	service     *app.Service
	inventoryID string
	stock       func() (inventorydomain.Item, error)
}

func TestShoppingRestockContract(t *testing.T) {
	factories := map[string]func(*testing.T) shoppingFixture{
		"sqlite": func(t *testing.T) shoppingFixture {
			database := testutil.NewTestDatabase(t)
			ctx := context.Background()
			user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "shop-contract", Email: "shop@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			inventory := repository.NewInventoryRepository(database)
			area, err := inventory.CreateArea(ctx, inventorydomain.Area{Name: "Pantry", CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			item, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: "Milk", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			return shoppingFixture{app.New(repository.NewShoppingRepository(database, repository.NewShoppingInventory), uuid.NewString, time.Now), item.ID, func() (inventorydomain.Item, error) { return inventory.FindItemByID(ctx, item.ID) }}
		},
		"fake": func(t *testing.T) shoppingFixture {
			fake := newShoppingFake(inventorydomain.Item{ID: "stock", Name: "Milk", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 5})
			return shoppingFixture{app.New(fake, uuid.NewString, time.Now), "stock", func() (inventorydomain.Item, error) {
				fake.mutex.Lock()
				defer fake.mutex.Unlock()
				return fake.state.stocks["stock"], nil
			}}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			fixture := factory(t)
			ctx := context.Background()
			item, err := fixture.service.Create(ctx, "  Milk  ", &fixture.inventoryID)
			if err != nil || item.Name != "Milk" {
				t.Fatalf("create %+v %v", item, err)
			}
			duplicate, err := fixture.service.Create(ctx, "Milk", &fixture.inventoryID)
			if err != nil || duplicate.ID != item.ID {
				t.Fatalf("duplicate %+v %v", duplicate, err)
			}
			purchased := true
			item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil {
				t.Fatalf("purchase %+v %v", item, err)
			}
			purchaseID := *item.PurchaseID
			invalid := 0
			if _, err := fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: purchaseID, Quantity: &invalid}); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("invalid restock: %v", err)
			}
			quantity := 2
			results := make(chan error, 10)
			start := make(chan struct{})
			for range 10 {
				go func() {
					<-start
					_, err := fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: purchaseID, Quantity: &quantity})
					results <- err
				}()
			}
			close(start)
			for range 10 {
				if err := <-results; err != nil {
					t.Fatal(err)
				}
			}
			stock, err := fixture.stock()
			if err != nil || stock.Quantity != 7 {
				t.Fatalf("stock %+v %v", stock, err)
			}
			items, err := fixture.service.List(ctx)
			if err != nil || len(items) != 1 || !items[0].Restocked {
				t.Fatalf("shopping %+v %v", items, err)
			}
			purchased = false
			item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: items[0].Version, Purchased: &purchased})
			if err != nil {
				t.Fatal(err)
			}
			purchased = true
			item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil || *item.PurchaseID == purchaseID {
				t.Fatalf("repurchase %+v %v", item, err)
			}
			if _, err := fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: purchaseID, Quantity: &quantity}); !errors.Is(err, domain.ErrConflict) {
				t.Fatalf("old purchase: %v", err)
			}
			if err := fixture.service.Delete(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("missing: %v", err)
			}
		})
	}
}

type transactionFixture struct {
	store             app.Store
	service           *app.Service
	firstID, secondID string
	stock             func(string) (inventorydomain.Item, error)
	fail              func(string) error
	clearFailure      func() error
	purchases         func(string) (int, bool, error)
	legacy            func(string) error
	setLevel          func(string) error
}

func transactionFixtureFor(t *testing.T, backend string) transactionFixture {
	t.Helper()
	if backend == "fake" {
		fake := newShoppingFake(inventorydomain.Item{ID: "one", Name: "Milk", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 5}, inventorydomain.Item{ID: "two", Name: "Bread", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 2})
		return transactionFixture{store: fake, service: app.New(fake, uuid.NewString, time.Now), firstID: "one", secondID: "two", stock: func(id string) (inventorydomain.Item, error) {
			fake.mutex.Lock()
			defer fake.mutex.Unlock()
			return fake.state.stocks[id], nil
		}, fail: func(stage string) error {
			fake.mutex.Lock()
			defer fake.mutex.Unlock()
			fake.failAt = stage
			return nil
		}, clearFailure: func() error { fake.mutex.Lock(); defer fake.mutex.Unlock(); fake.failAt = ""; return nil }, setLevel: func(id string) error {
			fake.mutex.Lock()
			defer fake.mutex.Unlock()
			stock := fake.state.stocks[id]
			stock.TrackingMode = inventorydomain.TrackingModeLevel
			stock.Level = 20
			fake.state.stocks[id] = stock
			return nil
		}, legacy: func(id string) error {
			fake.mutex.Lock()
			defer fake.mutex.Unlock()
			item := fake.state.items[id]
			item.Purchased = true
			item.PurchaseID = nil
			fake.state.items[id] = item
			return nil
		}, purchases: func(id string) (int, bool, error) {
			fake.mutex.Lock()
			defer fake.mutex.Unlock()
			if id == "" {
				return len(fake.state.purchases), false, nil
			}
			purchase, ok := fake.state.purchases[id]
			if !ok {
				return 0, false, nil
			}
			return 1, purchase.claimed, nil
		}}
	}
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "tx-contract", Email: "tx@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	inventory := repository.NewInventoryRepository(database)
	area, err := inventory.CreateArea(ctx, inventorydomain.Area{Name: "Pantry", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	first, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: "Milk", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	second, err := inventory.CreateItem(ctx, inventorydomain.Item{AreaID: area.ID, Name: "Bread", TrackingMode: inventorydomain.TrackingModeCount, Quantity: 2, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	store := repository.NewShoppingRepository(database, repository.NewShoppingInventory)
	return transactionFixture{store: store, service: app.New(store, uuid.NewString, time.Now), firstID: first.ID, secondID: second.ID, stock: func(id string) (inventorydomain.Item, error) { return inventory.FindItemByID(ctx, id) }, fail: func(stage string) error {
		table := "shopping_items"
		if stage == "claim" {
			table = "shopping_purchases"
		}
		_, err := database.ExecContext(ctx, "CREATE TRIGGER fail_contract BEFORE UPDATE ON "+table+" BEGIN SELECT RAISE(ABORT, 'injected transaction failure'); END")
		return err
	}, clearFailure: func() error { _, err := database.ExecContext(ctx, "DROP TRIGGER fail_contract"); return err }, setLevel: func(id string) error {
		_, err := database.ExecContext(ctx, "UPDATE inventory_items SET tracking_mode=?, level=? WHERE id=?", inventorydomain.TrackingModeLevel, 20, id)
		return err
	}, legacy: func(id string) error {
		_, err := database.ExecContext(ctx, "UPDATE shopping_items SET purchased=1, purchase_id=NULL WHERE id=?", id)
		return err
	}, purchases: func(id string) (int, bool, error) {
		if id == "" {
			var count int
			err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM shopping_purchases").Scan(&count)
			return count, false, err
		}
		var claimed bool
		err := database.QueryRowContext(ctx, "SELECT restocked_at IS NOT NULL FROM shopping_purchases WHERE id=?", id).Scan(&claimed)
		return 1, claimed, err
	}}
}

func TestShoppingTransactionRollbackContract(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		for _, stage := range []string{"save", "claim", "mark"} {
			t.Run(backend+"/"+stage, func(t *testing.T) {
				fixture := transactionFixtureFor(t, backend)
				ctx := context.Background()
				item, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
				if err != nil {
					t.Fatal(err)
				}
				purchased := true
				if stage != "save" {
					item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
					if err != nil || item.PurchaseID == nil {
						t.Fatalf("purchase = %+v, %v", item, err)
					}
				}
				if err := fixture.fail(stage); err != nil {
					t.Fatal(err)
				}
				quantity := 3
				if stage == "save" {
					_, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
				} else {
					_, err = fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: *item.PurchaseID, Quantity: &quantity})
				}
				if err == nil {
					t.Fatal("expected injected failure")
				}
				list, err := fixture.service.List(ctx)
				if err != nil || len(list) != 1 || list[0].Purchased != item.Purchased || list[0].Restocked || list[0].Version != item.Version {
					t.Fatalf("item after rollback = %+v, %v", list, err)
				}
				stock, err := fixture.stock(fixture.firstID)
				if err != nil || stock.Quantity != 5 {
					t.Fatalf("stock after rollback = %+v, %v", stock, err)
				}
				purchaseID := ""
				if item.PurchaseID != nil {
					purchaseID = *item.PurchaseID
				}
				count, claimed, err := fixture.purchases(purchaseID)
				wantCount := 0
				if stage != "save" {
					wantCount = 1
				}
				if err != nil || count != wantCount || claimed {
					t.Fatalf("purchase after rollback count=%d claimed=%v err=%v", count, claimed, err)
				}
				if err := fixture.clearFailure(); err != nil {
					t.Fatal(err)
				}
				if stage == "save" {
					item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
					if err != nil || item.PurchaseID == nil {
						t.Fatalf("retry purchase = %+v, %v", item, err)
					}
				} else {
					item, err = fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: *item.PurchaseID, Quantity: &quantity})
					if err != nil || !item.Restocked {
						t.Fatalf("retry restock = %+v, %v", item, err)
					}
					stock, err = fixture.stock(fixture.firstID)
					if err != nil || stock.Quantity != 8 {
						t.Fatalf("stock after retry = %+v, %v", stock, err)
					}
				}
			})
		}
	}
}

func TestShoppingOrderingAndCancellationContract(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			fixture := transactionFixtureFor(t, backend)
			ctx := context.Background()
			first, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
			if err != nil {
				t.Fatal(err)
			}
			second, err := fixture.service.Create(ctx, "Bread", &fixture.secondID)
			if err != nil {
				t.Fatal(err)
			}
			purchased := true
			first, err = fixture.service.Update(ctx, first.ID, domain.Change{Version: first.Version, Purchased: &purchased})
			if err != nil {
				t.Fatal(err)
			}
			list, err := fixture.service.List(ctx)
			if err != nil || len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
				t.Fatalf("ordered list = %+v, %v", list, err)
			}
			*list[0].InventoryItemID = "mutated"
			list[1].Purchased = false
			list, err = fixture.service.List(ctx)
			if err != nil || *list[0].InventoryItemID != fixture.secondID || !list[1].Purchased {
				t.Fatalf("stored values changed = %+v, %v", list, err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := fixture.service.Create(canceled, "Eggs", nil); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled create = %v", err)
			}
			if _, err := fixture.service.Update(canceled, first.ID, domain.Change{Version: first.Version, Purchased: &purchased}); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled update = %v", err)
			}
			if _, err := fixture.service.List(canceled); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled list = %v", err)
			}
			list, err = fixture.service.List(ctx)
			if err != nil || len(list) != 2 {
				t.Fatalf("items after cancellation = %+v, %v", list, err)
			}
			if err := fixture.service.ClearPurchased(ctx); err != nil {
				t.Fatal(err)
			}
			list, err = fixture.service.List(ctx)
			if err != nil || len(list) != 1 || list[0].ID != second.ID {
				t.Fatalf("items after clearing purchased = %+v, %v", list, err)
			}
			count, _, err := fixture.purchases("")
			if err != nil || count != 0 {
				t.Fatalf("purchases after clear = %d, %v", count, err)
			}
		})
	}
}

func TestShoppingCannotRestorePurchasedItemWhenActiveDuplicateExists(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			fixture := transactionFixtureFor(t, backend)
			ctx := context.Background()
			first, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
			if err != nil {
				t.Fatal(err)
			}
			purchased := true
			first, err = fixture.service.Update(ctx, first.ID, domain.Change{Version: first.Version, Purchased: &purchased})
			if err != nil {
				t.Fatal(err)
			}
			second, err := fixture.service.Create(ctx, "More milk", &fixture.firstID)
			if err != nil || second.ID == first.ID {
				t.Fatalf("second item = %+v, %v", second, err)
			}
			purchased = false
			if _, err := fixture.service.Update(ctx, first.ID, domain.Change{Version: first.Version, Purchased: &purchased}); !errors.Is(err, domain.ErrConflict) {
				t.Fatalf("restore duplicate = %v", err)
			}
			list, err := fixture.service.List(ctx)
			if err != nil || len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID || !list[1].Purchased {
				t.Fatalf("list after conflict = %+v, %v", list, err)
			}
		})
	}
}

func TestShoppingPurchaseInsertionErrorsAreNotConflictMapped(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			fixture := transactionFixtureFor(t, backend)
			ctx := context.Background()
			item, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
			if err != nil {
				t.Fatal(err)
			}
			purchased := true
			item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil {
				t.Fatalf("purchase = %+v, %v", item, err)
			}
			err = fixture.store.Transact(ctx, func(tx app.Transaction) (bool, error) { return true, tx.InsertPurchase(ctx, *item.PurchaseID, item.ID) })
			if err == nil || errors.Is(err, domain.ErrConflict) {
				t.Fatalf("duplicate purchase error = %v", err)
			}
			err = fixture.store.Transact(ctx, func(tx app.Transaction) (bool, error) {
				return true, tx.InsertPurchase(ctx, uuid.NewString(), "missing")
			})
			if err == nil || errors.Is(err, domain.ErrConflict) {
				t.Fatalf("missing item error = %v", err)
			}
		})
	}
}

func TestShoppingPreservesLegacyStateOnUnrelatedUpdate(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			fixture := transactionFixtureFor(t, backend)
			ctx := context.Background()
			item, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
			if err != nil {
				t.Fatal(err)
			}
			if err := fixture.legacy(item.ID); err != nil {
				t.Fatal(err)
			}
			name := "Milk and cream"
			updated, err := fixture.service.Update(ctx, item.ID, domain.Change{Name: &name, Version: item.Version})
			if err != nil {
				t.Fatal(err)
			}
			if updated.Name != name || !updated.Purchased || updated.PurchaseID != nil || updated.Version != item.Version+1 {
				t.Fatalf("legacy update = %+v", updated)
			}
			list, err := fixture.service.List(ctx)
			if err != nil || len(list) != 1 || !list[0].Purchased || list[0].PurchaseID != nil || list[0].Name != name {
				t.Fatalf("restored item = %+v, %v", list, err)
			}
		})
	}
}

func TestShoppingLevelRestockContract(t *testing.T) {
	for _, backend := range []string{"fake", "sqlite"} {
		t.Run(backend, func(t *testing.T) {
			fixture := transactionFixtureFor(t, backend)
			ctx := context.Background()
			if err := fixture.setLevel(fixture.firstID); err != nil {
				t.Fatal(err)
			}
			item, err := fixture.service.Create(ctx, "Milk", &fixture.firstID)
			if err != nil {
				t.Fatal(err)
			}
			purchased := true
			item, err = fixture.service.Update(ctx, item.ID, domain.Change{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil {
				t.Fatalf("purchase = %+v, %v", item, err)
			}
			quantity := 2
			if _, err := fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: *item.PurchaseID, Quantity: &quantity}); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("quantity for level = %v", err)
			}
			invalid := 101
			if _, err := fixture.service.Restock(ctx, item.ID, domain.Restock{PurchaseID: *item.PurchaseID, Level: &invalid}); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("invalid level = %v", err)
			}
			level := 75
			request := domain.Restock{PurchaseID: *item.PurchaseID, Level: &level}
			updated, err := fixture.service.Restock(ctx, item.ID, request)
			if err != nil || !updated.Restocked {
				t.Fatalf("restock = %+v, %v", updated, err)
			}
			_, err = fixture.service.Restock(ctx, item.ID, request)
			if err != nil {
				t.Fatal(err)
			}
			stock, err := fixture.stock(fixture.firstID)
			if err != nil || stock.Level != 75 || stock.Quantity != 5 {
				t.Fatalf("level stock = %+v, %v", stock, err)
			}
		})
	}
}
