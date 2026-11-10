package calendar_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/calendar"
	"github.com/bensuskins/family-hub/internal/models"
)

type refreshStore struct {
	mu      sync.Mutex
	sub     models.ICalSubscription
	updated chan struct{}
}

func (s *refreshStore) FindAll(context.Context) ([]models.ICalSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return []models.ICalSubscription{s.sub}, nil
}
func (s *refreshStore) FindByID(context.Context, string) (models.ICalSubscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sub, nil
}
func (s *refreshStore) UpdateCache(_ context.Context, _ string, data string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sub.CachedData = &data
	s.sub.LastFetchedAt = &at
	s.updated <- struct{}{}
	return nil
}

type refreshFeed struct {
	started chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
	now     time.Time
}

func (f *refreshFeed) Fetch(ctx context.Context, _ string) (string, error) {
	f.mu.Lock()
	f.calls++
	if f.calls == 1 {
		close(f.started)
	}
	f.mu.Unlock()
	select {
	case <-f.release:
		return "new", nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
func (f *refreshFeed) Parse(data, _, _ string) ([]models.Event, error) {
	return []models.Event{{ID: data, StartTime: f.now}}, nil
}

func TestStaleFeedReturnsImmediatelyAndRefreshesOnce(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	old := "old"
	last := now.Add(-time.Hour)
	store := &refreshStore{sub: models.ICalSubscription{ID: "a", CachedData: &old, LastFetchedAt: &last}, updated: make(chan struct{}, 4)}
	feed := &refreshFeed{started: make(chan struct{}), release: make(chan struct{}), now: now}
	service := calendar.New(store, feed, func() time.Time { return now })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan []models.Event, 1)
	go func() { events, _ := service.FetchForRange(ctx, now, now.Add(time.Hour)); result <- events }()
	select {
	case events := <-result:
		if len(events) != 1 || events[0].ID != "old" {
			t.Fatalf("events=%v", events)
		}
	case <-time.After(time.Second):
		close(feed.release)
		t.Fatal("stale request waited for refresh")
	}
	select {
	case <-feed.started:
	case <-time.After(time.Second):
		t.Fatal("refresh did not start")
	}
	cancel()
	events, err := service.FetchForRange(context.Background(), now, now.Add(time.Hour))
	if err != nil || len(events) != 1 || events[0].ID != "old" {
		t.Fatalf("events=%v err=%v", events, err)
	}
	feed.mu.Lock()
	calls := feed.calls
	feed.mu.Unlock()
	if calls != 1 {
		t.Fatalf("fetches=%d", calls)
	}
	close(feed.release)
	select {
	case <-store.updated:
	case <-time.After(time.Second):
		t.Fatal("refresh did not complete after request cancellation")
	}
	events, err = service.FetchForRange(context.Background(), now, now.Add(time.Hour))
	if err != nil || len(events) != 1 || events[0].ID != "new" {
		t.Fatalf("events=%v err=%v", events, err)
	}
}

func TestForceRefreshWaitsForBackgroundRefresh(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	old := "old"
	last := now.Add(-time.Hour)
	store := &refreshStore{sub: models.ICalSubscription{ID: "a", CachedData: &old, LastFetchedAt: &last}, updated: make(chan struct{}, 4)}
	feed := &refreshFeed{started: make(chan struct{}), release: make(chan struct{}), now: now}
	service := calendar.New(store, feed, func() time.Time { return now })
	if _, err := service.FetchForRange(context.Background(), now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	<-feed.started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.ForceRefreshByID(ctx, "a"); err != context.Canceled {
		t.Fatalf("force refresh error=%v", err)
	}
	done := make(chan error, 1)
	go func() { done <- service.ForceRefreshByID(context.Background(), "a") }()
	close(feed.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("force refresh did not complete")
	}
	feed.mu.Lock()
	defer feed.mu.Unlock()
	if feed.calls != 2 {
		t.Fatalf("fetches=%d", feed.calls)
	}
}
