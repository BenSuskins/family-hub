package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type Store interface {
	FindByID(context.Context, string) (models.ICalSubscription, error)
	FindAll(context.Context) ([]models.ICalSubscription, error)
	UpdateCache(context.Context, string, string, time.Time) error
}
type Feed interface {
	Fetch(context.Context, string) (string, error)
	Parse(string, string, string) ([]models.Event, error)
}
type Service struct {
	subRepo      Store
	feed         Feed
	now          func() time.Time
	cacheTTL     time.Duration
	refreshMutex sync.Mutex
	refreshing   map[string]chan struct{}
	lifecycle    context.Context
	cancel       context.CancelFunc
	stopping     bool
	background   sync.WaitGroup
}

func New(ctx context.Context, store Store, feed Feed, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	lifecycle, cancel := context.WithCancel(ctx)
	return &Service{subRepo: store, feed: feed, now: now, cacheTTL: 30 * time.Minute, refreshing: make(map[string]chan struct{}), lifecycle: lifecycle, cancel: cancel}
}
func (fetcher *Service) ForceRefreshByID(ctx context.Context, id string) error {
	sub, err := fetcher.subRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("finding subscription: %w", err)
	}
	for {
		done, started, allowed := fetcher.beginRefresh(sub.ID, false)
		if !allowed {
			return context.Canceled
		}
		if started {
			defer fetcher.endRefresh(sub.ID)
			return fetcher.refresh(ctx, sub)
		}
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (fetcher *Service) FetchForRange(ctx context.Context, start, end time.Time) ([]models.Event, error) {
	subs, err := fetcher.subRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading subscriptions: %w", err)
	}

	var events []models.Event
	for _, sub := range subs {
		subEvents, err := fetcher.fetchSubscription(ctx, sub)
		if err != nil {
			slog.Warn("skipping ical subscription", "name", sub.Name, "error", err)
			continue
		}
		for _, event := range subEvents {
			inRange := !event.StartTime.Before(start) && event.StartTime.Before(end)
			if event.EndTime != nil && event.EndTime.After(event.StartTime) {
				inRange = event.StartTime.Before(end) && event.EndTime.After(start)
			}
			if inRange {
				events = append(events, event)
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].StartTime.Before(events[j].StartTime)
	})

	return events, nil
}

func (fetcher *Service) fetchSubscription(ctx context.Context, sub models.ICalSubscription) ([]models.Event, error) {
	needsFetch := sub.LastFetchedAt == nil || fetcher.now().Sub(*sub.LastFetchedAt) > fetcher.cacheTTL

	if needsFetch && sub.CachedData != nil {
		if _, started, _ := fetcher.beginRefresh(sub.ID, true); started {
			go func() {
				defer fetcher.background.Done()
				defer fetcher.endRefresh(sub.ID)
				refreshContext, cancel := context.WithTimeout(fetcher.lifecycle, 30*time.Second)
				defer cancel()
				if err := fetcher.refresh(refreshContext, sub); err != nil && fetcher.lifecycle.Err() == nil {
					slog.Warn("refreshing ical subscription", "name", sub.Name, "error", err)
				}
			}()
		}
	} else if needsFetch {
		data, err := fetcher.feed.Fetch(ctx, sub.URL)
		if err != nil {
			slog.Warn("fetching ical url", "url", sub.URL, "error", err)
		} else {
			now := fetcher.now()
			if updateErr := fetcher.subRepo.UpdateCache(ctx, sub.ID, data, now); updateErr != nil {
				slog.Error("updating ical cache", "error", updateErr)
			}
			sub.CachedData = &data
		}
	}

	if sub.CachedData == nil {
		return nil, fmt.Errorf("no cached data for subscription %q", sub.Name)
	}

	return fetcher.feed.Parse(*sub.CachedData, sub.ID, sub.Color)
}

func (fetcher *Service) refresh(ctx context.Context, sub models.ICalSubscription) error {
	data, err := fetcher.feed.Fetch(ctx, sub.URL)
	if err != nil {
		return fmt.Errorf("fetching url: %w", err)
	}
	return fetcher.subRepo.UpdateCache(ctx, sub.ID, data, fetcher.now())
}

func (fetcher *Service) beginRefresh(id string, background bool) (<-chan struct{}, bool, bool) {
	fetcher.refreshMutex.Lock()
	defer fetcher.refreshMutex.Unlock()
	if fetcher.stopping || fetcher.lifecycle.Err() != nil {
		return nil, false, false
	}
	if done, ok := fetcher.refreshing[id]; ok {
		return done, false, true
	}
	done := make(chan struct{})
	fetcher.refreshing[id] = done
	if background {
		fetcher.background.Add(1)
	}
	return done, true, true
}

func (fetcher *Service) endRefresh(id string) {
	fetcher.refreshMutex.Lock()
	defer fetcher.refreshMutex.Unlock()
	close(fetcher.refreshing[id])
	delete(fetcher.refreshing, id)
}

func (fetcher *Service) Shutdown(ctx context.Context) error {
	fetcher.refreshMutex.Lock()
	fetcher.stopping = true
	fetcher.cancel()
	fetcher.refreshMutex.Unlock()
	done := make(chan struct{})
	go func() { fetcher.background.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
