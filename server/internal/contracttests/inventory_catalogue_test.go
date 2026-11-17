package contracttests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	inventoryapp "github.com/bensuskins/family-hub/internal/application/inventory"
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

type inventoryCatalogueFake struct {
	mutex sync.Mutex
	users map[string]bool
	areas map[string]inventorydomain.Area
	items map[string]inventorydomain.Item
}

func copyCatalogueArea(area inventorydomain.Area) inventorydomain.Area {
	if area.Items != nil {
		area.Items = append([]inventorydomain.Item{}, area.Items...)
	}
	return area
}
func (fake *inventoryCatalogueFake) FindAllAreas(ctx context.Context) ([]inventorydomain.Area, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var areas []inventorydomain.Area
	for _, area := range fake.areas {
		area.Items = []inventorydomain.Item{}
		for _, item := range fake.items {
			if item.AreaID == area.ID {
				area.Items = append(area.Items, item)
			}
		}
		sort.Slice(area.Items, func(i, j int) bool { return area.Items[i].Name < area.Items[j].Name })
		areas = append(areas, area)
	}
	sort.Slice(areas, func(i, j int) bool { return areas[i].Name < areas[j].Name })
	return areas, nil
}
func (fake *inventoryCatalogueFake) FindAreaByID(ctx context.Context, id string) (inventorydomain.Area, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return inventorydomain.Area{}, err
	}
	area, ok := fake.areas[id]
	if !ok {
		return inventorydomain.Area{}, inventorydomain.ErrAreaNotFound
	}
	area.Items = []inventorydomain.Item{}
	for _, item := range fake.items {
		if item.AreaID == id {
			area.Items = append(area.Items, item)
		}
	}
	sort.Slice(area.Items, func(i, j int) bool { return area.Items[i].Name < area.Items[j].Name })
	return area, nil
}
func (fake *inventoryCatalogueFake) CreateArea(ctx context.Context, area inventorydomain.Area) (inventorydomain.Area, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return inventorydomain.Area{}, err
	}
	if area.ID == "" {
		area.ID = uuid.NewString()
	}
	if _, ok := fake.areas[area.ID]; ok {
		return inventorydomain.Area{}, errors.New("duplicate area")
	}
	if !fake.users[area.CreatedByUserID] {
		return inventorydomain.Area{}, errors.New("missing user")
	}
	if area.Icon == "" {
		area.Icon = "box"
	}
	if area.Tint == "" {
		area.Tint = "blue"
	}
	now := time.Now()
	area.CreatedAt = now
	area.UpdatedAt = now
	if area.Items == nil {
		area.Items = []inventorydomain.Item{}
	}
	fake.areas[area.ID] = copyCatalogueArea(area)
	return copyCatalogueArea(area), nil
}
func (fake *inventoryCatalogueFake) UpdateArea(ctx context.Context, area inventorydomain.Area) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	existing, ok := fake.areas[area.ID]
	if ok {
		existing.Name = area.Name
		existing.Icon = area.Icon
		existing.Tint = area.Tint
		existing.UpdatedAt = time.Now()
		fake.areas[area.ID] = existing
	}
	return nil
}
func (fake *inventoryCatalogueFake) DeleteArea(ctx context.Context, id string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(fake.areas, id)
	for key, item := range fake.items {
		if item.AreaID == id {
			delete(fake.items, key)
		}
	}
	return nil
}
func (fake *inventoryCatalogueFake) FindItemByID(ctx context.Context, id string) (inventorydomain.Item, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return inventorydomain.Item{}, err
	}
	item, ok := fake.items[id]
	if !ok {
		return inventorydomain.Item{}, inventorydomain.ErrItemNotFound
	}
	return item, nil
}
func (fake *inventoryCatalogueFake) CreateItem(ctx context.Context, item inventorydomain.Item) (inventorydomain.Item, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return inventorydomain.Item{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if _, ok := fake.items[item.ID]; ok {
		return inventorydomain.Item{}, errors.New("duplicate item")
	}
	if _, ok := fake.areas[item.AreaID]; !ok {
		return inventorydomain.Item{}, errors.New("missing area")
	}
	if !fake.users[item.CreatedByUserID] {
		return inventorydomain.Item{}, errors.New("missing user")
	}
	item = inventorydomain.NormalizeItem(item)
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now
	fake.items[item.ID] = item
	return item, nil
}
func (fake *inventoryCatalogueFake) UpdateItem(ctx context.Context, item inventorydomain.Item) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	existing, ok := fake.items[item.ID]
	if ok {
		item = inventorydomain.NormalizeItem(item)
		existing.Name = item.Name
		existing.TrackingMode = item.TrackingMode
		existing.Quantity = item.Quantity
		existing.Level = item.Level
		existing.Unit = item.Unit
		existing.LowAt = item.LowAt
		existing.UpdatedAt = time.Now()
		fake.items[item.ID] = existing
	}
	return nil
}
func (fake *inventoryCatalogueFake) DeleteItem(ctx context.Context, id string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(fake.items, id)
	return nil
}

func TestInventoryCatalogueContract(t *testing.T) {
	factories := map[string]func(*testing.T) (inventoryapp.Store, string){
		"fake": func(t *testing.T) (inventoryapp.Store, string) {
			return &inventoryCatalogueFake{users: map[string]bool{"user": true}, areas: map[string]inventorydomain.Area{}, items: map[string]inventorydomain.Item{}}, "user"
		},
		"sqlite": func(t *testing.T) (inventoryapp.Store, string) {
			db := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "inventory-catalogue", Email: "catalogue@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewInventoryRepository(db), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, user := factory(t)
			service := inventoryapp.New(store)
			ctx := context.Background()
			empty, err := service.FindAllAreas(ctx)
			if err != nil || empty != nil {
				t.Fatalf("empty areas = %+v, %v", empty, err)
			}
			if _, err := service.CreateArea(ctx, inventorydomain.Area{ID: "bad-creator", Name: "Bad", CreatedByUserID: "missing-user"}); err == nil {
				t.Fatal("area with missing creator accepted")
			}
			generated, err := service.CreateArea(ctx, inventorydomain.Area{Name: "Temporary", CreatedByUserID: user})
			if err != nil || generated.ID == "" {
				t.Fatalf("generated area: %+v, %v", generated, err)
			}
			if err := service.DeleteArea(ctx, generated.ID); err != nil {
				t.Fatal(err)
			}
			concurrentArea, err := service.CreateArea(ctx, inventorydomain.Area{ID: "concurrent", Name: "Concurrent", CreatedByUserID: user})
			if err != nil {
				t.Fatal(err)
			}
			failures := make(chan error, 8)
			start := make(chan struct{})
			for index := range 8 {
				go func() {
					<-start
					_, err := service.CreateItem(ctx, inventorydomain.Item{ID: fmt.Sprintf("concurrent-%d", index), AreaID: concurrentArea.ID, Name: fmt.Sprintf("Item %d", index), CreatedByUserID: user})
					failures <- err
				}()
			}
			close(start)
			for range 8 {
				if err := <-failures; err != nil {
					t.Fatal(err)
				}
			}
			got, err := service.FindAreaByID(ctx, concurrentArea.ID)
			if err != nil || len(got.Items) != 8 {
				t.Fatalf("concurrent items = %+v, %v", got.Items, err)
			}
			if err := service.DeleteArea(ctx, concurrentArea.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := service.FindAreaByID(ctx, "missing"); !errors.Is(err, inventorydomain.ErrAreaNotFound) {
				t.Fatalf("missing area: %v", err)
			}
			if _, err := service.FindItemByID(ctx, "missing"); !errors.Is(err, inventorydomain.ErrItemNotFound) {
				t.Fatalf("missing item: %v", err)
			}
			pantry, err := service.CreateArea(ctx, inventorydomain.Area{ID: "pantry", Name: "Pantry", CreatedByUserID: user})
			if err != nil || pantry.Icon != "box" || pantry.Tint != "blue" || pantry.Items == nil || len(pantry.Items) != 0 {
				t.Fatalf("create area: %+v, %v", pantry, err)
			}
			if _, err := service.CreateArea(ctx, inventorydomain.Area{ID: "pantry", Name: "duplicate", CreatedByUserID: user}); err == nil {
				t.Fatal("duplicate area accepted")
			}
			_, err = service.CreateArea(ctx, inventorydomain.Area{ID: "bathroom", Name: "Bathroom", CreatedByUserID: user})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.CreateItem(ctx, inventorydomain.Item{ID: "bad-creator", AreaID: pantry.ID, Name: "Bad", CreatedByUserID: "missing-user"}); err == nil {
				t.Fatal("item with missing creator accepted")
			}
			generatedItem, err := service.CreateItem(ctx, inventorydomain.Item{AreaID: pantry.ID, Name: "Temporary", CreatedByUserID: user})
			if err != nil || generatedItem.ID == "" {
				t.Fatalf("generated item: %+v, %v", generatedItem, err)
			}
			if err := service.DeleteItem(ctx, generatedItem.ID); err != nil {
				t.Fatal(err)
			}
			milk, err := service.CreateItem(ctx, inventorydomain.Item{ID: "milk", AreaID: pantry.ID, Name: "Milk", TrackingMode: "legacy", Quantity: -2, Level: 101, LowAt: -1, CreatedByUserID: user})
			if err != nil || milk.TrackingMode != inventorydomain.TrackingModeCount || milk.Quantity != 0 || milk.Level != 100 || milk.LowAt != 0 {
				t.Fatalf("normalization: %+v, %v", milk, err)
			}
			if _, err := service.CreateItem(ctx, inventorydomain.Item{ID: "milk", AreaID: pantry.ID, Name: "Duplicate", CreatedByUserID: user}); err == nil {
				t.Fatal("duplicate item accepted")
			}
			if _, err := service.CreateItem(ctx, inventorydomain.Item{ID: "unknown", AreaID: "missing", Name: "Bad", CreatedByUserID: user}); err == nil {
				t.Fatal("missing parent accepted")
			}
			_, err = service.CreateItem(ctx, inventorydomain.Item{ID: "bread", AreaID: pantry.ID, Name: "Bread", TrackingMode: inventorydomain.TrackingModeCount, CreatedByUserID: user})
			if err != nil {
				t.Fatal(err)
			}
			areas, err := service.FindAllAreas(ctx)
			if err != nil || len(areas) != 2 || areas[0].Name != "Bathroom" || areas[1].Name != "Pantry" || areas[0].Items == nil || len(areas[1].Items) != 2 || areas[1].Items[0].Name != "Bread" {
				t.Fatalf("areas order: %+v, %v", areas, err)
			}
			areas[1].Items[0].Name = "Changed"
			again, err := service.FindAreaByID(ctx, pantry.ID)
			if err != nil || again.Items[0].Name != "Bread" {
				t.Fatalf("copy: %+v, %v", again, err)
			}
			pantry.Name = "Kitchen"
			updated, err := service.SaveArea(ctx, pantry)
			if err != nil || updated.Name != "Kitchen" || len(updated.Items) != 2 {
				t.Fatalf("save area: %+v, %v", updated, err)
			}
			milk.Name = "Milk 2"
			milk.TrackingMode = inventorydomain.TrackingModeLevel
			milk.Level = 150
			milk.LowAt = 150
			milk, err = service.SaveItem(ctx, milk)
			if err != nil || milk.Name != "Milk 2" || milk.Level != 100 || milk.LowAt != 100 {
				t.Fatalf("save item: %+v, %v", milk, err)
			}
			if err := service.DeleteItem(ctx, "bread"); err != nil {
				t.Fatal(err)
			}
			if _, err := service.FindItemByID(ctx, "bread"); !errors.Is(err, inventorydomain.ErrItemNotFound) {
				t.Fatalf("deleted item: %v", err)
			}
			if err := service.DeleteArea(ctx, "pantry"); err != nil {
				t.Fatal(err)
			}
			if _, err := service.FindItemByID(ctx, "milk"); !errors.Is(err, inventorydomain.ErrItemNotFound) {
				t.Fatalf("cascade: %v", err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := service.FindAllAreas(canceled); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel: %v", err)
			}
		})
	}
}
