package categories

import (
	"context"
	"errors"
	"testing"
)

type categoryFake struct {
	item        Category
	items       []Category
	updateError error
	reloadError error
}

func (fake *categoryFake) FindAll(context.Context) ([]Category, error) { return fake.items, nil }
func (fake *categoryFake) FindByID(context.Context, string) (Category, error) {
	return fake.item, fake.reloadError
}
func (fake *categoryFake) Create(_ context.Context, item Category) (Category, error) {
	fake.item = item
	return item, nil
}
func (fake *categoryFake) Update(_ context.Context, id, name string) error {
	if fake.updateError != nil {
		return fake.updateError
	}
	fake.item.ID = id
	fake.item.Name = name
	return nil
}
func (fake *categoryFake) Delete(context.Context, string) error { return nil }

func TestUpdateDistinguishesWriteAndReloadFailures(t *testing.T) {
	failure := errors.New("storage failure")
	fake := &categoryFake{updateError: failure}
	service := New(fake)
	if _, err := service.Update(context.Background(), "one", "Kitchen"); !errors.Is(err, failure) || errors.Is(err, ErrReload) {
		t.Fatalf("update error = %v", err)
	}
	fake.updateError = nil
	fake.reloadError = failure
	if _, err := service.Update(context.Background(), "one", "Kitchen"); !errors.Is(err, failure) || !errors.Is(err, ErrReload) {
		t.Fatalf("reload error = %v", err)
	}
	fake.reloadError = nil
	item, err := service.Update(context.Background(), "one", "Kitchen")
	if err != nil || item.ID != "one" || item.Name != "Kitchen" {
		t.Fatalf("update = %+v, %v", item, err)
	}
}
func TestCreateKeepsCreator(t *testing.T) {
	service := New(&categoryFake{})
	item, err := service.Create(context.Background(), "Kitchen", "creator")
	if err != nil || item.Name != "Kitchen" || item.CreatedByUserID != "creator" {
		t.Fatalf("create = %+v, %v", item, err)
	}
}

func TestListDoesNotExposeStoredSlice(t *testing.T) {
	fake := &categoryFake{items: []Category{{Name: "Kitchen"}}}
	list, err := New(fake).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	list[0].Name = "Changed"
	if fake.items[0].Name != "Kitchen" {
		t.Fatal("list exposed stored slice")
	}
}
