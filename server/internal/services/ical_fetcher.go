package services

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/bensuskins/family-hub/internal/models"
)

type ICalAdapter struct{ client *http.Client }

func NewICalAdapter(client *http.Client) *ICalAdapter {
	if client == nil {
		client = NewSafeHTTPClient(10 * time.Second)
	}
	return &ICalAdapter{client: client}
}

const maxICalBodyBytes = 10 * 1024 * 1024 // 10 MB

func (fetcher *ICalAdapter) Fetch(ctx context.Context, rawURL string) (string, error) {
	if err := ValidateExternalURL(rawURL); err != nil {
		return "", fmt.Errorf("blocked URL: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	resp, err := fetcher.client.Do(request)
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

func (fetcher *ICalAdapter) Parse(data string, subscriptionID string, subscriptionColor string) ([]models.Event, error) {
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
