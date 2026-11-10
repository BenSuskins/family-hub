package calendar_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/calendar"
	"github.com/bensuskins/family-hub/internal/models"
)

type storeFake struct{ subscriptions []models.ICalSubscription }

func (store *storeFake) FindAll(context.Context) ([]models.ICalSubscription, error) {
	return store.subscriptions, nil
}
func (store *storeFake) FindByID(context.Context, string) (models.ICalSubscription, error) {
	return store.subscriptions[0], nil
}
func (store *storeFake) UpdateCache(context.Context, string, string, time.Time) error { return nil }

type feedFake struct {
	fetched int
	events  []models.Event
}

func (feed *feedFake) Fetch(context.Context, string) (string, error) {
	feed.fetched++
	return "", errors.New("offline")
}
func (feed *feedFake) Parse(string, string, string) ([]models.Event, error) { return feed.events, nil }
func TestStaleCacheRemainsAvailableWhenFeedFails(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cached := "old feed"
	last := now.Add(-time.Hour)
	store := &storeFake{subscriptions: []models.ICalSubscription{{ID: "a", CachedData: &cached, LastFetchedAt: &last}}}
	feed := &feedFake{events: []models.Event{{ID: "in", StartTime: now}, {ID: "out", StartTime: now.Add(3 * time.Hour)}}}
	service := calendar.New(store, feed, func() time.Time { return now })
	events, err := service.FetchForRange(context.Background(), now, now.Add(time.Hour))
	if err != nil || len(events) != 1 || events[0].ID != "in" {
		t.Fatalf("events=%+v error=%v", events, err)
	}
}

func TestFreshCacheDoesNotFetchFeed(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cached := "cached feed"
	last := now.Add(-29 * time.Minute)
	store := &storeFake{subscriptions: []models.ICalSubscription{{ID: "a", CachedData: &cached, LastFetchedAt: &last}}}
	feed := &feedFake{events: []models.Event{{ID: "event", StartTime: now}}}
	service := calendar.New(store, feed, func() time.Time { return now })
	events, err := service.FetchForRange(context.Background(), now, now.Add(time.Hour))
	if err != nil || feed.fetched != 0 || len(events) != 1 {
		t.Fatalf("events=%+v fetches=%d error=%v", events, feed.fetched, err)
	}
}
