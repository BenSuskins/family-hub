package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type ICalFetcher struct {
	subRepo  repository.ICalSubscriptionRepository
	cacheTTL time.Duration
	client   *http.Client
}

func NewICalFetcher(subRepo repository.ICalSubscriptionRepository) *ICalFetcher {
	return &ICalFetcher{
		subRepo:  subRepo,
		cacheTTL: 30 * time.Minute,
		client:   NewSafeHTTPClient(10 * time.Second),
	}
}

func (fetcher *ICalFetcher) ForceRefreshByID(ctx context.Context, id string) error {
	sub, err := fetcher.subRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("finding subscription: %w", err)
	}
	data, err := fetcher.fetchURL(sub.URL)
	if err != nil {
		return fmt.Errorf("fetching url: %w", err)
	}
	return fetcher.subRepo.UpdateCache(ctx, sub.ID, data, time.Now())
}

func (fetcher *ICalFetcher) FetchForRange(ctx context.Context, start, end time.Time) ([]models.Event, error) {
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
			inRange := (event.StartTime.Equal(start) || event.StartTime.After(start)) && event.StartTime.Before(end)
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

func (fetcher *ICalFetcher) fetchSubscription(ctx context.Context, sub models.ICalSubscription) ([]models.Event, error) {
	needsFetch := sub.LastFetchedAt == nil || time.Since(*sub.LastFetchedAt) > fetcher.cacheTTL

	if needsFetch {
		data, err := fetcher.fetchURL(sub.URL)
		if err != nil {
			slog.Warn("fetching ical url", "url", sub.URL, "error", err)
		} else {
			now := time.Now()
			if updateErr := fetcher.subRepo.UpdateCache(ctx, sub.ID, data, now); updateErr != nil {
				slog.Error("updating ical cache", "error", updateErr)
			}
			sub.CachedData = &data
			sub.LastFetchedAt = &now
		}
	}

	if sub.CachedData == nil {
		return nil, fmt.Errorf("no cached data for subscription %q", sub.Name)
	}

	return parseICalData(*sub.CachedData, sub.ID, sub.Color)
}

const maxICalBodyBytes = 10 * 1024 * 1024 // 10 MB

func (fetcher *ICalFetcher) fetchURL(rawURL string) (string, error) {
	if err := ValidateExternalURL(rawURL); err != nil {
		return "", fmt.Errorf("blocked URL: %w", err)
	}

	resp, err := fetcher.client.Get(rawURL)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxICalBodyBytes))
	if err != nil {
		return "", fmt.Errorf("reading body: %w", err)
	}
	return string(data), nil
}

func parseICalData(data string, subscriptionID string, subscriptionColor string) ([]models.Event, error) {
	cal, err := ical.ParseCalendar(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing ical: %w", err)
	}

	var events []models.Event
	for _, e := range cal.Events() {
		event, err := convertICalEvent(e, subscriptionID, subscriptionColor)
		if err != nil {
			slog.Debug("skipping ical event", "error", err)
			continue
		}
		events = append(events, event)
	}
	return events, nil
}

func eventPropertyValue(event *ical.VEvent, property ical.ComponentProperty, fallback string) string {
	if prop := event.GetProperty(property); prop != nil {
		return prop.Value
	}
	return fallback
}

func convertICalEvent(e *ical.VEvent, subscriptionID string, subscriptionColor string) (models.Event, error) {
	uid := subscriptionID + "-" + eventPropertyValue(e, ical.ComponentPropertyUniqueId, "unknown")
	title := eventPropertyValue(e, ical.ComponentPropertySummary, "(No title)")
	description := eventPropertyValue(e, ical.ComponentPropertyDescription, "")
	location := eventPropertyValue(e, ical.ComponentPropertyLocation, "")

	dtStartProp := e.GetProperty(ical.ComponentPropertyDtStart)
	if dtStartProp == nil {
		return models.Event{}, fmt.Errorf("missing DTSTART for event %q", title)
	}

	allDay := isAllDayProperty(dtStartProp)

	var startTime time.Time
	var err error
	if allDay {
		startTime, err = e.GetAllDayStartAt()
	} else {
		startTime, err = e.GetStartAt()
	}
	if err != nil {
		return models.Event{}, fmt.Errorf("parsing DTSTART for event %q: %w", title, err)
	}

	var endTime *time.Time
	if allDay {
		if t, err := e.GetAllDayEndAt(); err == nil {
			endTime = &t
		}
	} else {
		if t, err := e.GetEndAt(); err == nil {
			endTime = &t
		}
	}

	return models.Event{
		ID:          uid,
		Title:       title,
		Description: description,
		Location:    location,
		StartTime:   startTime,
		EndTime:     endTime,
		AllDay:      allDay,
		Color:       subscriptionColor,
	}, nil
}

func isAllDayProperty(prop *ical.IANAProperty) bool {
	for _, values := range prop.ICalParameters {
		for _, v := range values {
			if strings.EqualFold(v, "DATE") {
				return true
			}
		}
	}
	// Fallback: date-only values have exactly 8 chars (YYYYMMDD)
	return len(strings.TrimSpace(prop.Value)) == 8
}
