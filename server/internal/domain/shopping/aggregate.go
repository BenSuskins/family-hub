package shopping

import "errors"

var ErrInvalidState = errors.New("invalid shopping item state")

type Aggregate struct{ item Item }

func New(id, name string, inventoryID *string) (Aggregate, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return Aggregate{}, err
	}
	return Restore(Item{ID: id, Name: normalized, InventoryItemID: inventoryID})
}

func Restore(item Item) (Aggregate, error) {
	name, err := NormalizeName(item.Name)
	if item.ID == "" || err != nil || name != item.Name || item.Version < 0 || (item.Purchased != (item.PurchaseID != nil)) || (item.Restocked && (!item.Purchased || item.InventoryItemID == nil)) {
		return Aggregate{}, ErrInvalidState
	}
	return Aggregate{item: cloneItem(item)}, nil
}

// RestoreLegacy preserves records that predate the current invariants.
func RestoreLegacy(item Item) Aggregate    { return Aggregate{item: cloneItem(item)} }
func (aggregate Aggregate) Snapshot() Item { return cloneItem(aggregate.item) }
func (aggregate Aggregate) Change(change Change, purchaseID string) (Aggregate, bool, error) {
	item, created, err := ApplyChange(aggregate.item, change, purchaseID)
	return Aggregate{item: cloneItem(item)}, created, err
}
func (aggregate Aggregate) CheckRestock(request Restock) (bool, error) {
	return CheckRestock(aggregate.item, request)
}
func (aggregate Aggregate) Saved() Aggregate {
	item := aggregate.Snapshot()
	item.Version++
	return Aggregate{item: item}
}
func (aggregate Aggregate) Restocked(request Restock) (Aggregate, error) {
	already, err := aggregate.CheckRestock(request)
	if err != nil {
		return aggregate, err
	}
	if already {
		return aggregate, nil
	}
	item := aggregate.Snapshot()
	item.Restocked = true
	item.Version++
	return Aggregate{item: item}, nil
}

func cloneItem(item Item) Item {
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
