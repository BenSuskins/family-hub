package commute

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var ErrUnresolvedDestination = errors.New("destination is unresolved")

type DepartureCalendar interface {
	FetchForRange(context.Context, time.Time, time.Time) ([]models.Event, error)
}
type DrivingProvider interface {
	DrivingSeconds(context.Context, models.HomeInfo, string) (int, error)
}
type DepartureService struct {
	calendar DepartureCalendar
	driving  DrivingProvider
	now      func() time.Time
}

func NewDepartureService(calendar DepartureCalendar, driving DrivingProvider, now func() time.Time) *DepartureService {
	return &DepartureService{calendar: calendar, driving: driving, now: now}
}
func (service *DepartureService) Departures(ctx context.Context, home models.HomeInfo) (models.Departures, error) {
	result := models.Departures{Status: "ok", Items: []models.Departure{}}
	location, err := time.LoadLocation(home.Timezone)
	if err != nil {
		return result, errors.New("invalid household timezone")
	}
	now := service.now()
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)
	events, err := service.calendar.FetchForRange(ctx, start, end)
	if err != nil {
		return result, errors.New("calendar unavailable")
	}
	counts := departureEvaluationCounts{calendarEvents: len(events)}
	defer func() {
		slog.Info("departure estimates evaluated",
			"calendar_events", counts.calendarEvents,
			"estimates", counts.estimates,
			"skipped_all_day", counts.skippedAllDay,
			"skipped_started", counts.skippedStarted,
			"skipped_outside_today", counts.skippedOutsideToday,
			"skipped_missing_location", counts.skippedMissingLocation,
			"skipped_online_location", counts.skippedOnlineLocation,
			"skipped_unresolved_destination", counts.skippedUnresolvedDestination,
			"route_failures", counts.routeFailures,
		)
	}()
	events = append([]models.Event(nil), events...)
	sort.SliceStable(events, func(first, second int) bool { return events[first].StartTime.Before(events[second].StartTime) })
	for _, event := range events {
		if event.AllDay {
			counts.skippedAllDay++
			continue
		}
		if !event.StartTime.After(now) {
			counts.skippedStarted++
			continue
		}
		if !event.StartTime.Before(end) {
			counts.skippedOutsideToday++
			continue
		}
		switch destinationSkipReason(event.Location) {
		case "missing_location":
			counts.skippedMissingLocation++
			continue
		case "online_location":
			counts.skippedOnlineLocation++
			continue
		}
		seconds, err := service.driving.DrivingSeconds(ctx, home, event.Location)
		if errors.Is(err, ErrUnresolvedDestination) {
			counts.skippedUnresolvedDestination++
			continue
		}
		if err != nil {
			counts.routeFailures++
			return models.Departures{}, err
		}
		departure := models.Departure{EventID: event.ID, Title: event.Title, Destination: event.Location, StartTime: event.StartTime, DrivingSeconds: seconds, FetchedAt: now, LeaveAt: event.StartTime.Add(-time.Duration(seconds)*time.Second - time.Duration(home.ArrivalMarginMinutes)*time.Minute)}
		result.Items = append(result.Items, departure)
		counts.estimates++
		if result.SelectedEventID == nil {
			identifier := event.ID
			result.SelectedEventID = &identifier
		}
	}
	return result, nil
}

type departureEvaluationCounts struct {
	calendarEvents               int
	estimates                    int
	skippedAllDay                int
	skippedStarted               int
	skippedOutsideToday          int
	skippedMissingLocation       int
	skippedOnlineLocation        int
	skippedUnresolvedDestination int
	routeFailures                int
}

func destinationSkipReason(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "missing_location"
	}
	for _, online := range []string{"http://", "https://", "zoom", "teams.microsoft", "meet.google", "online", "virtual", "microsoft teams", "telephone", "phone call"} {
		if strings.Contains(value, online) {
			return "online_location"
		}
	}
	return ""
}
