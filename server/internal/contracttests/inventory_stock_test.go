package contracttests

import (
	"context"
	"errors"
	"sync"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type stockFake struct {
	mutex sync.Mutex
	item  models.InventoryItem
}

func (fake *stockFake) ModifyStock(ctx context.Context, id string, change func(models.InventoryItem) (models.InventoryItem, error)) (models.InventoryItem, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return models.InventoryItem{}, err
	}
	if id != fake.item.ID {
		return models.InventoryItem{}, domain.ErrItemNotFound
	}
	updated, err := change(fake.item)
	if err != nil {
		return models.InventoryItem{}, err
	}
	fake.item = updated
	return updated, nil
}
func TestStockOperationContract(t *testing.T) {
	factories := map[string]func(*testing.T) (*app.StockService, string){
		"sqlite": func(t *testing.T) (*app.StockService, string) {
			database := testutil.NewTestDatabase(t)
			ctx := context.Background()
			user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "stock-contract", Email: "stock@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			store := repository.NewInventoryRepository(database)
			area, err := store.CreateArea(ctx, models.InventoryArea{Name: "Pantry", CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			item, err := store.CreateItem(ctx, models.InventoryItem{AreaID: area.ID, Name: "Rice", TrackingMode: models.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			return app.NewStockService(store), item.ID
		},
		"fake": func(t *testing.T) (*app.StockService, string) {
			return app.NewStockService(&stockFake{item: models.InventoryItem{ID: "item", Name: "Rice", TrackingMode: models.TrackingModeCount, Quantity: 5}}), "item"
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			service, id := factory(t)
			ctx := context.Background()
			delta := int64(1)
			results := make(chan error, 20)
			start := make(chan struct{})
			for range 20 {
				go func() {
					<-start
					_, err := service.Update(ctx, id, models.InventoryStockChange{QuantityDelta: &delta})
					results <- err
				}()
			}
			close(start)
			for range 20 {
				if err := <-results; err != nil {
					t.Fatal(err)
				}
			}
			zero := int64(0)
			saved, err := service.Update(ctx, id, models.InventoryStockChange{QuantityDelta: &zero})
			if err != nil || saved.Quantity != 25 || saved.Name != "Rice" {
				t.Fatalf("concurrent stock: %+v %v", saved, err)
			}
			decrease := int64(-100)
			saved, err = service.Update(ctx, id, models.InventoryStockChange{QuantityDelta: &decrease})
			if err != nil || saved.Quantity != 0 {
				t.Fatalf("clamp: %+v %v", saved, err)
			}
			if _, err := service.Update(ctx, "missing", models.InventoryStockChange{QuantityDelta: &delta}); !errors.Is(err, domain.ErrItemNotFound) {
				t.Fatalf("missing: %v", err)
			}
			level := int64(50)
			if _, err := service.Update(ctx, id, models.InventoryStockChange{Level: &level}); !errors.Is(err, domain.ErrInvalidStockChange) {
				t.Fatalf("wrong mode: %v", err)
			}
		})
	}
}
