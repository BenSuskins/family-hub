package shopping

import (
	"errors"
	"strings"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrConflict = errors.New("shopping item changed; refresh and try again")
	ErrInvalid  = errors.New("invalid shopping change")
	ErrNotFound = errors.New("shopping item not found")
)

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return "", ErrInvalid
	}
	return name, nil
}
func ApplyChange(item models.ShoppingItem, change models.ShoppingChange, purchaseID string) (models.ShoppingItem, bool, error) {
	if item.InventoryItemID != nil {
		value := *item.InventoryItemID
		item.InventoryItemID = &value
	}
	if item.PurchaseID != nil {
		value := *item.PurchaseID
		item.PurchaseID = &value
	}
	if item.Version != change.Version {
		return item, false, ErrConflict
	}
	if change.Name != nil {
		name, err := NormalizeName(*change.Name)
		if err != nil {
			return item, false, err
		}
		item.Name = name
	}
	createPurchase := false
	if change.Purchased != nil && item.Purchased != *change.Purchased {
		item.Purchased = *change.Purchased
		item.Restocked = false
		item.PurchaseID = nil
		if item.Purchased {
			item.PurchaseID = &purchaseID
			createPurchase = true
		}
	}
	return item, createPurchase, nil
}
func ValidateRestock(mode string, request models.ShoppingRestock) error {
	if mode == models.TrackingModeCount {
		if request.Quantity == nil || request.Level != nil || *request.Quantity <= 0 || *request.Quantity > 1000000 {
			return ErrInvalid
		}
		return nil
	}
	if request.Level == nil || request.Quantity != nil || *request.Level < 0 || *request.Level > 100 {
		return ErrInvalid
	}
	return nil
}

func CheckRestock(item models.ShoppingItem, request models.ShoppingRestock) (bool, error) {
	if !item.Purchased || item.PurchaseID == nil || *item.PurchaseID != request.PurchaseID {
		return false, ErrConflict
	}
	if item.Restocked {
		return true, nil
	}
	if item.InventoryItemID == nil {
		return false, ErrInvalid
	}
	return false, nil
}
