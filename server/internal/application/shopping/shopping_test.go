package shopping_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/shopping"
	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
)

type shoppingFake struct {
	calls int
	name  string
}

func (fake *shoppingFake) FindAll(context.Context) ([]models.ShoppingItem, error) { return nil, nil }
func (fake *shoppingFake) Create(_ context.Context, name string, _ *string) (models.ShoppingItem, error) {
	fake.calls++
	fake.name = name
	return models.ShoppingItem{Name: name}, nil
}
func (fake *shoppingFake) Update(context.Context, string, models.ShoppingChange) (models.ShoppingItem, error) {
	return models.ShoppingItem{}, nil
}
func (fake *shoppingFake) Delete(context.Context, string) error { return nil }
func (fake *shoppingFake) ClearPurchased(context.Context) error { return nil }
func (fake *shoppingFake) Restock(context.Context, string, models.ShoppingRestock) (models.ShoppingItem, error) {
	return models.ShoppingItem{}, nil
}
func TestCreateNormalizesBeforeStorage(t *testing.T) {
	fake := &shoppingFake{}
	service := app.New(fake)
	if _, err := service.Create(context.Background(), "   ", nil); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
	if fake.calls != 0 {
		t.Fatal("invalid name reached storage")
	}
	item, err := service.Create(context.Background(), "  Milk  ", nil)
	if err != nil || item.Name != "Milk" || fake.name != "Milk" {
		t.Fatalf("got %+v: %v", item, err)
	}
}
