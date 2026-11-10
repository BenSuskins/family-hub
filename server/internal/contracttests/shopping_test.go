package contracttests

import (
	"context"
	"errors"
	"sync"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/shopping"
	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

type shoppingFake struct {
	mutex sync.Mutex
	items map[string]models.ShoppingItem
	stock models.InventoryItem
}

func copyShopping(item models.ShoppingItem) models.ShoppingItem {
	if item.InventoryItemID != nil {
		value := *item.InventoryItemID
		item.InventoryItemID = &value
	}
	if item.PurchaseID != nil {
		value := *item.PurchaseID
		item.PurchaseID = &value
	}
	return item
}
func (fake *shoppingFake) FindAll(ctx context.Context) ([]models.ShoppingItem, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := []models.ShoppingItem{}
	for _, item := range fake.items {
		items = append(items, copyShopping(item))
	}
	return items, nil
}
func (fake *shoppingFake) Create(ctx context.Context, name string, id *string) (models.ShoppingItem, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return models.ShoppingItem{}, err
	}
	if id != nil {
		if *id != fake.stock.ID {
			return models.ShoppingItem{}, domain.ErrNotFound
		}
		for _, item := range fake.items {
			if !item.Purchased && item.InventoryItemID != nil && *item.InventoryItemID == *id {
				return copyShopping(item), nil
			}
		}
	}
	item := models.ShoppingItem{ID: uuid.NewString(), Name: name, InventoryItemID: id}
	fake.items[item.ID] = copyShopping(item)
	return copyShopping(item), nil
}
func (fake *shoppingFake) Update(ctx context.Context, id string, change models.ShoppingChange) (models.ShoppingItem, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return models.ShoppingItem{}, err
	}
	item, found := fake.items[id]
	if !found {
		return item, domain.ErrNotFound
	}
	item, _, err := domain.ApplyChange(item, change, uuid.NewString())
	if err != nil {
		return item, err
	}
	item.Version++
	fake.items[id] = copyShopping(item)
	return copyShopping(item), nil
}
func (fake *shoppingFake) Delete(ctx context.Context, id string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, found := fake.items[id]; !found {
		return domain.ErrNotFound
	}
	delete(fake.items, id)
	return nil
}
func (fake *shoppingFake) ClearPurchased(ctx context.Context) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for id, item := range fake.items {
		if item.Purchased {
			delete(fake.items, id)
		}
	}
	return nil
}
func (fake *shoppingFake) Restock(ctx context.Context, id string, request models.ShoppingRestock) (models.ShoppingItem, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return models.ShoppingItem{}, err
	}
	item, found := fake.items[id]
	if !found {
		return item, domain.ErrNotFound
	}
	alreadyRestocked, err := domain.CheckRestock(item, request)
	if err != nil {
		return item, err
	}
	if alreadyRestocked {
		return copyShopping(item), nil
	}
	if err := domain.ValidateRestock(fake.stock.TrackingMode, request); err != nil {
		return item, err
	}
	if fake.stock.TrackingMode == models.TrackingModeCount {
		fake.stock.Quantity += *request.Quantity
	} else {
		fake.stock.Level = *request.Level
	}
	item.Restocked = true
	item.Version++
	fake.items[id] = copyShopping(item)
	return copyShopping(item), nil
}

type shoppingFixture struct {
	service     *app.Service
	inventoryID string
	stock       func() (models.InventoryItem, error)
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
			area, err := inventory.CreateArea(ctx, models.InventoryArea{Name: "Pantry", CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			item, err := inventory.CreateItem(ctx, models.InventoryItem{AreaID: area.ID, Name: "Milk", TrackingMode: models.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			return shoppingFixture{app.New(repository.NewShoppingRepository(database)), item.ID, func() (models.InventoryItem, error) { return inventory.FindItemByID(ctx, item.ID) }}
		},
		"fake": func(t *testing.T) shoppingFixture {
			fake := &shoppingFake{items: map[string]models.ShoppingItem{}, stock: models.InventoryItem{ID: "stock", Name: "Milk", TrackingMode: models.TrackingModeCount, Quantity: 5}}
			return shoppingFixture{app.New(fake), "stock", func() (models.InventoryItem, error) {
				fake.mutex.Lock()
				defer fake.mutex.Unlock()
				return fake.stock, nil
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
			item, err = fixture.service.Update(ctx, item.ID, models.ShoppingChange{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil {
				t.Fatalf("purchase %+v %v", item, err)
			}
			purchaseID := *item.PurchaseID
			invalid := 0
			if _, err := fixture.service.Restock(ctx, item.ID, models.ShoppingRestock{PurchaseID: purchaseID, Quantity: &invalid}); !errors.Is(err, domain.ErrInvalid) {
				t.Fatalf("invalid restock: %v", err)
			}
			quantity := 2
			results := make(chan error, 10)
			start := make(chan struct{})
			for range 10 {
				go func() {
					<-start
					_, err := fixture.service.Restock(ctx, item.ID, models.ShoppingRestock{PurchaseID: purchaseID, Quantity: &quantity})
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
			item, err = fixture.service.Update(ctx, item.ID, models.ShoppingChange{Version: items[0].Version, Purchased: &purchased})
			if err != nil {
				t.Fatal(err)
			}
			purchased = true
			item, err = fixture.service.Update(ctx, item.ID, models.ShoppingChange{Version: item.Version, Purchased: &purchased})
			if err != nil || item.PurchaseID == nil || *item.PurchaseID == purchaseID {
				t.Fatalf("repurchase %+v %v", item, err)
			}
			if _, err := fixture.service.Restock(ctx, item.ID, models.ShoppingRestock{PurchaseID: purchaseID, Quantity: &quantity}); !errors.Is(err, domain.ErrConflict) {
				t.Fatalf("old purchase: %v", err)
			}
			if err := fixture.service.Delete(ctx, "missing"); !errors.Is(err, domain.ErrNotFound) {
				t.Fatalf("missing: %v", err)
			}
		})
	}
}
