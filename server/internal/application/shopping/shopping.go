package shopping

import (
	"context"
	"time"

	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
)

type Store interface {
	FindAll(context.Context) ([]domain.Item, error)
	Delete(context.Context, string) error
	ClearPurchased(context.Context) error
	Transact(context.Context, func(Transaction) (bool, error)) error
}
type Transaction interface {
	Find(context.Context, string) (domain.Aggregate, error)
	FindActive(context.Context, string) (domain.Item, bool, error)
	Insert(context.Context, domain.Item) (domain.Item, error)
	InsertPurchase(context.Context, string, string) error
	Save(context.Context, domain.Item) error
	Claim(context.Context, string, time.Time) (bool, error)
	MarkRestocked(context.Context, string) error
	Inventory() Inventory
}
type Inventory interface {
	TrackingMode(context.Context, string) (string, error)
	ApplyRestock(context.Context, string, string, *int, *int, time.Time) error
}
type Service struct {
	store Store
	newID func() string
	now   func() time.Time
}

func New(store Store, newID func() string, now func() time.Time) *Service {
	return &Service{store: store, newID: newID, now: now}
}
func (service *Service) List(ctx context.Context) ([]domain.Item, error) {
	items, err := service.store.FindAll(ctx)
	if err != nil || items == nil {
		return items, err
	}
	result := make([]domain.Item, len(items))
	for index, item := range items {
		result[index] = copyItem(item)
	}
	return result, nil
}
func (service *Service) Create(ctx context.Context, name string, inventoryID *string) (domain.Item, error) {
	inventoryID = copyString(inventoryID)
	normalized, err := domain.NormalizeName(name)
	if err != nil {
		return domain.Item{}, err
	}
	var item domain.Item
	err = service.store.Transact(ctx, func(tx Transaction) (bool, error) {
		if inventoryID != nil {
			if _, err := tx.Inventory().TrackingMode(ctx, *inventoryID); err != nil {
				return false, err
			}
			existing, found, err := tx.FindActive(ctx, *inventoryID)
			if err != nil {
				return false, err
			}
			if found {
				item = copyItem(existing)
				return false, nil
			}
		}
		aggregate, err := domain.New(service.newID(), normalized, inventoryID)
		if err != nil {
			return false, err
		}
		item, err = tx.Insert(ctx, aggregate.Snapshot())
		return err == nil, err
	})
	return copyItem(item), err
}
func (service *Service) Update(ctx context.Context, id string, change domain.Change) (domain.Item, error) {
	var item domain.Item
	err := service.store.Transact(ctx, func(tx Transaction) (bool, error) {
		var err error
		aggregate, err := tx.Find(ctx, id)
		item = aggregate.Snapshot()
		if err != nil {
			return false, err
		}
		var createPurchase bool
		aggregate, createPurchase, err = aggregate.Change(change, service.newID())
		item = aggregate.Snapshot()
		if err != nil {
			return false, err
		}
		if createPurchase {
			if err = tx.InsertPurchase(ctx, *item.PurchaseID, id); err != nil {
				return false, err
			}
		}
		if err = tx.Save(ctx, item); err != nil {
			return false, err
		}
		item = aggregate.Saved().Snapshot()
		return true, nil
	})
	return copyItem(item), err
}
func (service *Service) Delete(ctx context.Context, id string) error {
	return service.store.Delete(ctx, id)
}
func (service *Service) ClearPurchased(ctx context.Context) error {
	return service.store.ClearPurchased(ctx)
}
func (service *Service) Restock(ctx context.Context, id string, request domain.Restock) (domain.Item, error) {
	request.Quantity = copyInt(request.Quantity)
	request.Level = copyInt(request.Level)
	var item domain.Item
	err := service.store.Transact(ctx, func(tx Transaction) (bool, error) {
		var err error
		aggregate, err := tx.Find(ctx, id)
		item = aggregate.Snapshot()
		if err != nil {
			return false, err
		}
		already, err := aggregate.CheckRestock(request)
		if err != nil || already {
			return false, err
		}
		mode, err := tx.Inventory().TrackingMode(ctx, *item.InventoryItemID)
		if err != nil {
			return false, err
		}
		if err = domain.ValidateRestock(mode, request); err != nil {
			return false, err
		}
		if err = tx.Inventory().ApplyRestock(ctx, *item.InventoryItemID, mode, request.Quantity, request.Level, service.now()); err != nil {
			return false, err
		}
		claimed, err := tx.Claim(ctx, request.PurchaseID, service.now())
		if err != nil {
			return false, err
		}
		if !claimed {
			return false, domain.ErrConflict
		}
		if err = tx.MarkRestocked(ctx, id); err != nil {
			return false, err
		}
		aggregate, err = aggregate.Restocked(request)
		if err != nil {
			return false, err
		}
		item = aggregate.Snapshot()
		return true, nil
	})
	return copyItem(item), err
}
func copyString(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
func copyItem(item domain.Item) domain.Item {
	item.InventoryItemID = copyString(item.InventoryItemID)
	item.PurchaseID = copyString(item.PurchaseID)
	return item
}

func copyInt(value *int) *int {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
