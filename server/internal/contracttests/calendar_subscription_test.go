package contracttests

import (
	"context"
	"errors"
	"testing"
	"time"

	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type subscriptionFake struct {
	items map[string]models.ICalSubscription
}

func (fake *subscriptionFake) FindAll(context.Context) ([]models.ICalSubscription, error) {
	var result []models.ICalSubscription
	for _, item := range fake.items {
		result = append(result, item)
	}
	return result, nil
}
func (fake *subscriptionFake) FindByID(_ context.Context, id string) (models.ICalSubscription, error) {
	item, ok := fake.items[id]
	if !ok {
		return models.ICalSubscription{}, calendarapp.ErrNotFound
	}
	return item, nil
}
func (fake *subscriptionFake) Create(_ context.Context, item models.ICalSubscription) error {
	fake.items[item.ID] = item
	return nil
}
func (fake *subscriptionFake) UpdateColor(_ context.Context, id, color string) error {
	item, ok := fake.items[id]
	if ok {
		item.Color = color
		fake.items[id] = item
	}
	return nil
}
func (fake *subscriptionFake) Delete(_ context.Context, id string) error {
	delete(fake.items, id)
	return nil
}
func (fake *subscriptionFake) UpdateCache(_ context.Context, id, data string, at time.Time) error {
	item, ok := fake.items[id]
	if ok {
		item.CachedData = &data
		item.LastFetchedAt = &at
		fake.items[id] = item
	}
	return nil
}

type subscriptionContractStore interface {
	calendarapp.SubscriptionStore
	UpdateCache(context.Context, string, string, time.Time) error
}

func TestCalendarSubscriptionStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) subscriptionContractStore{
		"sqlite": func(t *testing.T) subscriptionContractStore {
			return repository.NewICalSubscriptionRepository(testutil.NewTestDatabase(t))
		},
		"fake": func(t *testing.T) subscriptionContractStore {
			return &subscriptionFake{items: map[string]models.ICalSubscription{}}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			ctx := context.Background()
			item := models.ICalSubscription{ID: "school", Name: "School", URL: "https://example.com/calendar", Color: "blue"}
			if err := store.Create(ctx, item); err != nil {
				t.Fatal(err)
			}
			stamp := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			if err := store.UpdateCache(ctx, item.ID, "feed", stamp); err != nil {
				t.Fatal(err)
			}
			if err := store.UpdateColor(ctx, item.ID, "red"); err != nil {
				t.Fatal(err)
			}
			got, err := store.FindByID(ctx, item.ID)
			if err != nil || got.Color != "red" || got.CachedData == nil || *got.CachedData != "feed" || got.LastFetchedAt == nil || !got.LastFetchedAt.Equal(stamp) {
				t.Fatalf("stored=%+v err=%v", got, err)
			}
			all, err := store.FindAll(ctx)
			if err != nil || len(all) != 1 || all[0].ID != item.ID {
				t.Fatalf("all=%+v err=%v", all, err)
			}
			if err := store.Delete(ctx, item.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.FindByID(ctx, item.ID); !errors.Is(err, calendarapp.ErrNotFound) {
				t.Fatalf("missing error=%v", err)
			}
		})
	}
}
