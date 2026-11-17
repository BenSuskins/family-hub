package shopping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/shopping"
	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
)

type shoppingFake struct {
	calls            int
	item             domain.Item
	saved            bool
	insertedPurchase bool
}

func (fake *shoppingFake) FindAll(context.Context) ([]domain.Item, error) {
	return []domain.Item{fake.item}, nil
}
func (fake *shoppingFake) Delete(context.Context, string) error { return nil }
func (fake *shoppingFake) ClearPurchased(context.Context) error { return nil }
func (fake *shoppingFake) Transact(_ context.Context, operation func(app.Transaction) (bool, error)) error {
	_, err := operation(fake)
	return err
}
func (fake *shoppingFake) Find(context.Context, string) (domain.Aggregate, error) {
	aggregate, err := domain.Restore(fake.item)
	if err != nil {
		aggregate = domain.RestoreLegacy(fake.item)
	}
	return aggregate, nil
}
func (fake *shoppingFake) FindActive(context.Context, string) (domain.Item, bool, error) {
	return domain.Item{}, false, nil
}
func (fake *shoppingFake) Insert(_ context.Context, item domain.Item) (domain.Item, error) {
	fake.calls++
	fake.item = item
	return item, nil
}
func (fake *shoppingFake) InsertPurchase(context.Context, string, string) error {
	fake.insertedPurchase = true
	return nil
}
func (fake *shoppingFake) Save(_ context.Context, item domain.Item) error {
	if item.PurchaseID != nil && !fake.insertedPurchase {
		return errors.New("purchase was not inserted before item save")
	}
	fake.saved = true
	fake.item = item
	return nil
}
func (fake *shoppingFake) Claim(context.Context, string, time.Time) (bool, error) { return true, nil }
func (fake *shoppingFake) MarkRestocked(context.Context, string) error            { return nil }
func (fake *shoppingFake) Inventory() app.Inventory                               { return fake }
func (fake *shoppingFake) TrackingMode(context.Context, string) (string, error)   { return "count", nil }
func (fake *shoppingFake) ApplyRestock(context.Context, string, string, *int, *int, time.Time) error {
	return nil
}
func TestCreateNormalizesBeforeStorage(t *testing.T) {
	fake := &shoppingFake{}
	service := app.New(fake, func() string { return "id" }, time.Now)
	if _, err := service.Create(context.Background(), "   ", nil); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
	if fake.calls != 0 {
		t.Fatal("invalid name reached storage")
	}
	item, err := service.Create(context.Background(), "  Milk  ", nil)
	if err != nil || item.Name != "Milk" || fake.item.Name != "Milk" {
		t.Fatalf("got %+v: %v", item, err)
	}
}
func TestPurchaseInsertsPurchaseBeforeSaving(t *testing.T) {
	fake := &shoppingFake{item: domain.Item{ID: "item", Name: "Milk", Version: 4}}
	service := app.New(fake, func() string { return "purchase" }, time.Now)
	purchased := true
	item, err := service.Update(context.Background(), "item", domain.Change{Version: 4, Purchased: &purchased})
	if err != nil || !fake.insertedPurchase || !fake.saved || item.PurchaseID == nil || *item.PurchaseID != "purchase" || item.Version != 5 {
		t.Fatalf("item %+v, error %v", item, err)
	}
}

func TestLegacyStoredItemRemainsUpdatableWithoutRepair(t *testing.T) {
	fake := &shoppingFake{item: domain.Item{ID: "legacy", Name: "", Restocked: true, Version: 4}}
	service := app.New(fake, func() string { return "unused" }, time.Now)
	item, err := service.Update(context.Background(), "legacy", domain.Change{Version: 4})
	if err != nil || item.Name != "" || !item.Restocked || item.Version != 5 {
		t.Fatalf("legacy update: %+v, %v", item, err)
	}
}
