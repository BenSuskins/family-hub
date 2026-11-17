package contracttests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/inventory"
	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type stockFake struct {
	mutex sync.Mutex
	item  domain.Item
}

func (fake *stockFake) ModifyStock(ctx context.Context, id string, change func(domain.Stock) (domain.Stock, error)) (domain.Item, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.Item{}, err
	}
	if id != fake.item.ID {
		return domain.Item{}, domain.ErrItemNotFound
	}
	stock, err := domain.RestoreStock(fake.item)
	if err != nil {
		stock = domain.RestoreLegacyStock(fake.item)
	}
	updated, err := change(stock)
	if err != nil {
		return domain.Item{}, err
	}
	snapshot := updated.Snapshot()
	fake.item.Quantity = snapshot.Quantity
	fake.item.Level = snapshot.Level
	fake.item.UpdatedAt = time.Now()
	return fake.item, nil
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
			area, err := store.CreateArea(ctx, domain.Area{Name: "Pantry", CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			item, err := store.CreateItem(ctx, domain.Item{AreaID: area.ID, Name: "Rice", TrackingMode: domain.TrackingModeCount, Quantity: 5, CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
			return app.NewStockService(store), item.ID
		},
		"fake": func(t *testing.T) (*app.StockService, string) {
			return app.NewStockService(&stockFake{item: domain.Item{ID: "item", Name: "Rice", TrackingMode: domain.TrackingModeCount, Quantity: 5}}), "item"
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
					_, err := service.Update(ctx, id, domain.StockChange{QuantityDelta: &delta})
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
			saved, err := service.Update(ctx, id, domain.StockChange{QuantityDelta: &zero})
			if err != nil || saved.Quantity != 25 || saved.Name != "Rice" || saved.UpdatedAt.IsZero() {
				t.Fatalf("concurrent stock: %+v %v", saved, err)
			}
			decrease := int64(-100)
			saved, err = service.Update(ctx, id, domain.StockChange{QuantityDelta: &decrease})
			if err != nil || saved.Quantity != 0 {
				t.Fatalf("clamp: %+v %v", saved, err)
			}
			if _, err := service.Update(ctx, "missing", domain.StockChange{QuantityDelta: &delta}); !errors.Is(err, domain.ErrItemNotFound) {
				t.Fatalf("missing: %v", err)
			}
			level := int64(50)
			if _, err := service.Update(ctx, id, domain.StockChange{Level: &level}); !errors.Is(err, domain.ErrInvalidStockChange) {
				t.Fatalf("wrong mode: %v", err)
			}
		})
	}
}
