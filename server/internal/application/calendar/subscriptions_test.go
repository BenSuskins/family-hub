package calendar_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bensuskins/family-hub/internal/application/calendar"
	"github.com/bensuskins/family-hub/internal/models"
)

type subscriptionStoreFake struct {
	item               models.ICalSubscription
	findErr, createErr error
	updates, deletes   int
}

func (s *subscriptionStoreFake) FindAll(context.Context) ([]models.ICalSubscription, error) {
	return []models.ICalSubscription{s.item}, nil
}
func (s *subscriptionStoreFake) FindByID(context.Context, string) (models.ICalSubscription, error) {
	return s.item, s.findErr
}
func (s *subscriptionStoreFake) Create(_ context.Context, item models.ICalSubscription) error {
	s.item = item
	return s.createErr
}
func (s *subscriptionStoreFake) UpdateColor(_ context.Context, _ string, color string) error {
	s.updates++
	s.item.Color = color
	return nil
}
func (s *subscriptionStoreFake) Delete(context.Context, string) error { s.deletes++; return nil }
func TestSubscriptionUpdateRequiresExistingCalendar(t *testing.T) {
	store := &subscriptionStoreFake{findErr: calendar.ErrNotFound}
	service := calendar.NewSubscriptions(store, nil, func() string { return "id" })
	_, err := service.UpdateColor(context.Background(), "missing", "red")
	if !errors.Is(err, calendar.ErrNotFound) || store.updates != 0 {
		t.Fatalf("error=%v updates=%d", err, store.updates)
	}
}
func TestSubscriptionCreateReportsReadFailure(t *testing.T) {
	store := &subscriptionStoreFake{findErr: errors.New("read failed")}
	service := calendar.NewSubscriptions(store, nil, func() string { return "new-id" })
	_, err := service.Create(context.Background(), "Family", "https://example.com/feed", "blue")
	if !errors.Is(err, calendar.ErrReadCreated) || store.item.ID != "new-id" {
		t.Fatalf("error=%v item=%+v", err, store.item)
	}
}

func TestSubscriptionsReturnIndependentCachedValues(t *testing.T) {
	cached := "original"
	store := &subscriptionStoreFake{item: models.ICalSubscription{ID: "school", CachedData: &cached}}
	service := calendar.NewSubscriptions(store, nil, nil)
	subscriptions, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*subscriptions[0].CachedData = "changed"
	if *store.item.CachedData != "original" {
		t.Fatal("caller changed stored cache")
	}
}
