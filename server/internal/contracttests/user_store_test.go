package contracttests

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/users"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type userStoreFake struct{ items map[string]app.User }

func cloneContractUser(item app.User) app.User {
	if item.OnboardedAt != nil {
		value := *item.OnboardedAt
		item.OnboardedAt = &value
	}
	return item
}
func (fake *userStoreFake) List(ctx context.Context) ([]app.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var items []app.User
	for _, item := range fake.items {
		items = append(items, cloneContractUser(item))
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].ID < items[j].ID
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}
func (fake *userStoreFake) Find(ctx context.Context, id string) (app.User, error) {
	if err := ctx.Err(); err != nil {
		return app.User{}, err
	}
	item, ok := fake.items[id]
	if !ok {
		return app.User{}, app.ErrNotFound
	}
	return cloneContractUser(item), nil
}
func (fake *userStoreFake) SetRole(ctx context.Context, id string, role app.Role) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item, ok := fake.items[id]; ok {
		item.Role = role
		item.UpdatedAt = time.Now()
		fake.items[id] = item
	}
	return nil
}
func (fake *userStoreFake) SetAvatar(ctx context.Context, id, data string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item, ok := fake.items[id]; ok {
		item.AvatarURL = "/avatar/" + id
		item.UpdatedAt = time.Now()
		fake.items[id] = item
	}
	return nil
}
func (fake *userStoreFake) ClearAvatar(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item, ok := fake.items[id]; ok {
		item.AvatarURL = ""
		item.UpdatedAt = time.Now()
		fake.items[id] = item
	}
	return nil
}

func TestUserStoreContract(t *testing.T) {
	for _, name := range []string{"fake", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			var store app.Store
			if name == "fake" {
				now := time.Now()
				store = &userStoreFake{items: map[string]app.User{
					"first":  {ID: "first", OIDCSubject: "first-subject", Email: "first@example.com", Name: "Zebra", Role: app.RoleMember, CreatedAt: now, UpdatedAt: now, OnboardedAt: &now},
					"second": {ID: "second", OIDCSubject: "second-subject", Email: "second@example.com", Name: "Alpha", Role: app.RoleMember, CreatedAt: now, UpdatedAt: now},
				}}
			} else {
				db := testutil.NewTestDatabase(t)
				adapter := repository.NewUserRepository(db)
				for _, item := range []models.User{
					{ID: "first", OIDCSubject: "first-subject", Email: "first@example.com", Name: "Zebra", Role: models.RoleMember},
					{ID: "second", OIDCSubject: "second-subject", Email: "second@example.com", Name: "Alpha", Role: models.RoleMember},
				} {
					if _, err := adapter.Create(ctx, item); err != nil {
						t.Fatal(err)
					}
				}
				if err := adapter.MarkOnboarded(ctx, "first"); err != nil {
					t.Fatal(err)
				}
				store = adapter
			}
			list, err := store.List(ctx)
			if err != nil || len(list) != 2 || list[0].ID != "second" || list[1].ID != "first" {
				t.Fatalf("list = %+v, %v", list, err)
			}
			if list[1].OnboardedAt == nil {
				t.Fatal("onboarding timestamp missing")
			}
			*list[1].OnboardedAt = time.Time{}
			found, err := store.Find(ctx, "first")
			if err != nil || found.ID != "first" || found.OnboardedAt == nil || found.OnboardedAt.IsZero() {
				t.Fatalf("find = %+v, %v", found, err)
			}
			if _, err := store.Find(ctx, "missing"); !errors.Is(err, app.ErrNotFound) {
				t.Fatalf("missing = %v", err)
			}
			if err := store.SetRole(ctx, "first", app.RoleAdmin); err != nil {
				t.Fatal(err)
			}
			found, err = store.Find(ctx, "first")
			if err != nil || found.Role != app.RoleAdmin {
				t.Fatalf("role = %+v, %v", found, err)
			}
			if err := store.SetAvatar(ctx, "first", "data:image/png;base64,AAAA"); err != nil {
				t.Fatal(err)
			}
			found, err = store.Find(ctx, "first")
			if err != nil || found.AvatarURL != "/avatar/first" {
				t.Fatalf("avatar = %+v, %v", found, err)
			}
			if err := store.ClearAvatar(ctx, "first"); err != nil {
				t.Fatal(err)
			}
			found, err = store.Find(ctx, "first")
			if err != nil || found.AvatarURL != "" {
				t.Fatalf("cleared avatar = %+v, %v", found, err)
			}
			if err := store.SetRole(ctx, "missing", app.RoleAdmin); err != nil {
				t.Fatalf("missing role: %v", err)
			}
			if err := store.SetAvatar(ctx, "missing", "data"); err != nil {
				t.Fatalf("missing avatar: %v", err)
			}
			if err := store.ClearAvatar(ctx, "missing"); err != nil {
				t.Fatalf("missing clear: %v", err)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.Find(cancelled, "first"); err == nil {
				t.Fatal("cancelled lookup succeeded")
			}
			if err := store.SetRole(cancelled, "first", app.RoleMember); err == nil {
				t.Fatal("cancelled role update succeeded")
			}
		})
	}
}
