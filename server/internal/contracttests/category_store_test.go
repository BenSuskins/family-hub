package contracttests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/categories"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type categoryStoreFake struct {
	items map[string]app.Category
	next  int
}

func (fake *categoryStoreFake) FindByID(ctx context.Context, id string) (app.Category, error) {
	if err := ctx.Err(); err != nil {
		return app.Category{}, err
	}
	item, ok := fake.items[id]
	if !ok {
		return app.Category{}, app.ErrNotFound
	}
	return item, nil
}
func (fake *categoryStoreFake) FindAll(ctx context.Context) ([]app.Category, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var items []app.Category
	for _, item := range fake.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}
func (fake *categoryStoreFake) Create(ctx context.Context, item app.Category) (app.Category, error) {
	if err := ctx.Err(); err != nil {
		return app.Category{}, err
	}
	fake.next++
	if item.ID == "" {
		item.ID = fmt.Sprint(fake.next)
	}
	if _, exists := fake.items[item.ID]; exists {
		return app.Category{}, errors.New("duplicate id")
	}
	item.CreatedAt = time.Now()
	fake.items[item.ID] = item
	return item, nil
}
func (fake *categoryStoreFake) Update(ctx context.Context, id, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item, ok := fake.items[id]; ok {
		item.Name = name
		fake.items[id] = item
	}
	return nil
}
func (fake *categoryStoreFake) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(fake.items, id)
	return nil
}

func TestCategoryStoreContract(t *testing.T) {
	for _, name := range []string{"fake", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			var store app.Store
			userID := "user"
			if name == "fake" {
				store = &categoryStoreFake{items: make(map[string]app.Category)}
			} else {
				db := testutil.NewTestDatabase(t)
				user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "category-contract", Email: "category@example.com", Name: "Category", Role: models.RoleAdmin})
				if err != nil {
					t.Fatal(err)
				}
				userID = user.ID
				store = repository.NewCategoryRepository(db)
			}
			ctx := context.Background()
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.FindAll(cancelled); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled list: %v", err)
			}
			if _, err := store.FindByID(cancelled, "missing"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled lookup: %v", err)
			}
			if _, err := store.Create(cancelled, app.Category{ID: "cancelled", Name: "Cancelled", CreatedByUserID: userID}); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled create: %v", err)
			}
			if err := store.Update(cancelled, "missing", "Name"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled update: %v", err)
			}
			if err := store.Delete(cancelled, "missing"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancelled delete: %v", err)
			}
			empty, err := store.FindAll(ctx)
			if err != nil || empty != nil {
				t.Fatalf("empty list = %#v, %v", empty, err)
			}
			first, err := store.Create(ctx, app.Category{ID: "first", Name: "Zebra", CreatedByUserID: userID})
			if err != nil {
				t.Fatal(err)
			}
			_, err = store.Create(ctx, app.Category{ID: "second", Name: "Alpha", CreatedByUserID: userID})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.Create(ctx, app.Category{ID: "first", Name: "duplicate", CreatedByUserID: userID}); err == nil {
				t.Fatal("duplicate ID succeeded")
			}
			list, err := store.FindAll(ctx)
			if err != nil || len(list) != 2 || list[0].Name != "Alpha" || list[1].Name != "Zebra" {
				t.Fatalf("list = %#v, %v", list, err)
			}
			if err := store.Update(ctx, first.ID, "Kitchen"); err != nil {
				t.Fatal(err)
			}
			item, err := store.FindByID(ctx, first.ID)
			if err != nil || item.Name != "Kitchen" || item.CreatedByUserID != userID || item.CreatedAt.IsZero() {
				t.Fatalf("updated = %#v, %v", item, err)
			}
			if err := store.Update(ctx, "missing", "Unused"); err != nil {
				t.Fatal(err)
			}
			if err := store.Delete(ctx, first.ID); err != nil {
				t.Fatal(err)
			}
			if err := store.Delete(ctx, "missing"); err != nil {
				t.Fatal(err)
			}
			if _, err := store.FindByID(ctx, first.ID); !errors.Is(err, app.ErrNotFound) {
				t.Fatalf("missing error = %v", err)
			}
		})
	}
}
