package commute

import (
	"context"
	"time"
)

type DisplaySchedule struct {
	Enabled    bool
	Weekdays   []int
	Start, End string
}
type DisplayPreferences struct {
	Home, Work     string
	Count          int
	ToWork, ToHome DisplaySchedule
}
type DisplayTrain struct {
	ServiceID, Operator, Status                                                                                         string
	Platform, ScheduledDeparture, ExpectedDeparture, ScheduledArrival, ExpectedArrival, CancellationReason, DelayReason *string
}
type DisplayBoard struct {
	Origin, Destination    string
	GeneratedAt, FetchedAt time.Time
	Timezone               string
	Trains                 []DisplayTrain
}
type DisplayTimezone struct {
	Value                  string
	ReadError, DecodeError error
}
type DisplayObserver struct {
	RailFailure           func(error)
	TimezoneReadError     func(error)
	TimezoneDecodeError   func(error)
	TimezoneLocationError func(error)
}

type DisplayPreferencesSource interface {
	Preferences(context.Context, string) (DisplayPreferences, error)
}
type DisplayTimezoneSource interface {
	Timezone(context.Context) DisplayTimezone
}
type DisplayRail interface {
	Fetch(context.Context, string, string, int) (DisplayBoard, error)
}
type DisplayResult struct {
	Status, Timezone                                              string
	Boards                                                        []DisplayBoard
	Failures                                                      []error
	TimezoneReadError, TimezoneDecodeError, TimezoneLocationError error
}
type DisplayCommute struct {
	preferences DisplayPreferencesSource
	timezone    DisplayTimezoneSource
	rail        DisplayRail
	now         func() time.Time
}

func NewDisplayCommute(preferences DisplayPreferencesSource, timezone DisplayTimezoneSource, rail DisplayRail, now func() time.Time) *DisplayCommute {
	return &DisplayCommute{preferences, timezone, rail, now}
}
func (service *DisplayCommute) Trains(ctx context.Context, token string, observer DisplayObserver) (DisplayResult, error) {
	result := DisplayResult{Boards: []DisplayBoard{}}
	preferences, err := service.preferences.Preferences(ctx, token)
	if err != nil {
		return result, err
	}
	if !validDisplayStation(preferences.Home) || !validDisplayStation(preferences.Work) {
		result.Status = "setup_required"
		return result, nil
	}
	timezone := service.timezone.Timezone(ctx)
	result.TimezoneReadError = timezone.ReadError
	result.TimezoneDecodeError = timezone.DecodeError
	if timezone.ReadError != nil && observer.TimezoneReadError != nil {
		observer.TimezoneReadError(timezone.ReadError)
	}
	if timezone.DecodeError != nil && observer.TimezoneDecodeError != nil {
		observer.TimezoneDecodeError(timezone.DecodeError)
	}
	if timezone.Value == "" {
		timezone.Value = "Europe/London"
	}
	location, err := time.LoadLocation(timezone.Value)
	if err != nil {
		result.TimezoneLocationError = err
		if observer.TimezoneLocationError != nil {
			observer.TimezoneLocationError(err)
		}
		location = time.UTC
	}
	result.Timezone = location.String()
	now := service.now().In(location)
	routes := []struct {
		schedule            DisplaySchedule
		origin, destination string
	}{{preferences.ToWork, preferences.Home, preferences.Work}, {preferences.ToHome, preferences.Work, preferences.Home}}
	for _, route := range routes {
		if !activeDisplayWindow(route.schedule, now) {
			continue
		}
		board, err := service.rail.Fetch(ctx, route.origin, route.destination, preferences.Count)
		if err != nil {
			result.Failures = append(result.Failures, err)
			if observer.RailFailure != nil {
				observer.RailFailure(err)
			}
			continue
		}
		result.Boards = append(result.Boards, copyDisplayBoard(board))
	}
	switch {
	case len(result.Boards) == 0 && len(result.Failures) > 0:
		result.Status = "unavailable"
	case len(result.Boards) > 0 && len(result.Failures) > 0:
		result.Status = "partial"
	case len(result.Boards) == 0:
		result.Status = "outside_schedule"
	default:
		result.Status = "available"
	}
	return result, nil
}
func validDisplayStation(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
func activeDisplayWindow(schedule DisplaySchedule, now time.Time) bool {
	if !schedule.Enabled {
		return false
	}
	start, e1 := time.Parse("15:04", schedule.Start)
	end, e2 := time.Parse("15:04", schedule.End)
	if e1 != nil || e2 != nil || len(schedule.Weekdays) == 0 {
		return false
	}
	weekday := int(now.Weekday()) + 1
	minute := now.Hour()*60 + now.Minute()
	first, last := start.Hour()*60+start.Minute(), end.Hour()*60+end.Minute()
	selected := func(day int) bool {
		for _, v := range schedule.Weekdays {
			if v == day {
				return true
			}
		}
		return false
	}
	if first < last {
		return selected(weekday) && minute >= first && minute < last
	}
	if minute >= first {
		return selected(weekday)
	}
	previous := weekday - 1
	if previous == 0 {
		previous = 7
	}
	return minute < last && selected(previous)
}
func copyDisplayBoard(board DisplayBoard) DisplayBoard {
	if board.Trains != nil {
		board.Trains = append([]DisplayTrain{}, board.Trains...)
		for i := range board.Trains {
			t := &board.Trains[i]
			t.Platform = copyDisplayString(t.Platform)
			t.ScheduledDeparture = copyDisplayString(t.ScheduledDeparture)
			t.ExpectedDeparture = copyDisplayString(t.ExpectedDeparture)
			t.ScheduledArrival = copyDisplayString(t.ScheduledArrival)
			t.ExpectedArrival = copyDisplayString(t.ExpectedArrival)
			t.CancellationReason = copyDisplayString(t.CancellationReason)
			t.DelayReason = copyDisplayString(t.DelayReason)
		}
	}
	return board
}
func copyDisplayString(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
