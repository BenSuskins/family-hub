package commute_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/commute"
)

type displayPreferencesFake struct {
	value app.DisplayPreferences
	err   error
}

func (fake displayPreferencesFake) Preferences(context.Context, string) (app.DisplayPreferences, error) {
	return fake.value, fake.err
}

type displayTimezoneFake struct{ result app.DisplayTimezone }

func (fake displayTimezoneFake) Timezone(context.Context) app.DisplayTimezone { return fake.result }

type displayRailFake struct {
	calls int
	err   error
}

func (fake *displayRailFake) Fetch(context.Context, string, string, int) (app.DisplayBoard, error) {
	fake.calls++
	return app.DisplayBoard{}, fake.err
}
func TestDisplayCommuteStatusAndSchedule(t *testing.T) {
	now := time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC)
	preferences := app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}, ToHome: app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}}
	rail := &displayRailFake{err: errors.New("offline")}
	result, err := app.NewDisplayCommute(displayPreferencesFake{value: preferences}, displayTimezoneFake{app.DisplayTimezone{Value: "UTC"}}, rail, func() time.Time { return now }).Trains(context.Background(), "token", app.DisplayObserver{})
	if err != nil || result.Status != "unavailable" || len(result.Failures) != 2 || rail.calls != 2 {
		t.Fatalf("result %+v, calls %d, err %v", result, rail.calls, err)
	}
}

func TestDisplayCommuteSetupSkipsDependencies(t *testing.T) {
	rail := &displayRailFake{}
	result, err := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "kel", Work: "LST"}}, displayTimezoneFake{}, rail, time.Now).Trains(context.Background(), "token", app.DisplayObserver{})
	if err != nil || result.Status != "setup_required" || result.Timezone != "" || result.Boards == nil || rail.calls != 0 {
		t.Fatalf("result %+v, calls %d, err %v", result, rail.calls, err)
	}
}
func TestDisplayCommuteOvernightAndInvalidTimezoneFallback(t *testing.T) {
	schedule := app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "23:00", End: "02:00"}
	rail := &displayRailFake{}
	service := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: schedule}}, displayTimezoneFake{app.DisplayTimezone{Value: "invalid/timezone"}}, rail, func() time.Time { return time.Date(2026, 11, 17, 1, 30, 0, 0, time.UTC) })
	result, err := service.Trains(context.Background(), "token", app.DisplayObserver{})
	if err != nil || result.Status != "available" || result.Timezone != "UTC" || result.TimezoneLocationError == nil || rail.calls != 1 {
		t.Fatalf("result %+v, calls %d, err %v", result, rail.calls, err)
	}
}
func TestDisplayCommuteMissingTimezoneDefaultsLondon(t *testing.T) {
	sourceErr := errors.New("read failed")
	rail := &displayRailFake{}
	service := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "KEL", Work: "LST"}}, displayTimezoneFake{app.DisplayTimezone{ReadError: sourceErr}}, rail, func() time.Time { return time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC) })
	result, err := service.Trains(context.Background(), "token", app.DisplayObserver{})
	if err != nil || result.Timezone != "Europe/London" || !errors.Is(result.TimezoneReadError, sourceErr) || result.Status != "outside_schedule" || result.Boards == nil {
		t.Fatalf("result %+v, err %v", result, err)
	}
}

type displayPartialRail struct {
	calls int
	board app.DisplayBoard
}

func (fake *displayPartialRail) Fetch(context.Context, string, string, int) (app.DisplayBoard, error) {
	fake.calls++
	if fake.calls == 2 {
		return app.DisplayBoard{}, errors.New("offline")
	}
	return fake.board, nil
}
func TestDisplayCommutePartialAndDefensiveCopies(t *testing.T) {
	platform := "1"
	rail := &displayPartialRail{board: app.DisplayBoard{Trains: []app.DisplayTrain{{Platform: &platform}}}}
	schedule := app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}
	service := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: schedule, ToHome: schedule}}, displayTimezoneFake{app.DisplayTimezone{Value: "UTC"}}, rail, func() time.Time { return time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC) })
	result, err := service.Trains(context.Background(), "token", app.DisplayObserver{})
	if err != nil || result.Status != "partial" || len(result.Failures) != 1 || len(result.Boards) != 1 {
		t.Fatalf("result %+v err %v", result, err)
	}
	*result.Boards[0].Trains[0].Platform = "changed"
	if platform != "1" {
		t.Fatal("result exposed source pointer")
	}
}

type blockingDisplayRail struct {
	calls   int
	entered chan struct{}
	release chan struct{}
}

func (fake *blockingDisplayRail) Fetch(ctx context.Context, _, _ string, _ int) (app.DisplayBoard, error) {
	fake.calls++
	if fake.calls == 1 {
		return app.DisplayBoard{}, errors.New("first failed")
	}
	close(fake.entered)
	select {
	case <-fake.release:
		return app.DisplayBoard{}, nil
	case <-ctx.Done():
		return app.DisplayBoard{}, ctx.Err()
	}
}
func TestDisplayCommuteReportsFailureBeforeNextFetchCompletes(t *testing.T) {
	schedule := app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}
	rail := &blockingDisplayRail{entered: make(chan struct{}), release: make(chan struct{})}
	service := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: schedule, ToHome: schedule}}, displayTimezoneFake{app.DisplayTimezone{Value: "UTC"}}, rail, func() time.Time { return time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC) })
	failures := make(chan error, 1)
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, err := service.Trains(ctx, "token", app.DisplayObserver{RailFailure: func(err error) { failures <- err }})
		done <- err
	}()
	select {
	case <-rail.entered:
	case <-time.After(time.Second):
		t.Fatal("second fetch did not start")
	}
	select {
	case err := <-failures:
		if err.Error() != "first failed" {
			t.Fatalf("error %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first failure was not reported while second fetch was blocked")
	}
	select {
	case err := <-done:
		t.Fatalf("operation returned while second fetch blocked: %v", err)
	default:
	}
	close(rail.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("operation did not finish after release")
	}
}

type blockingFirstRail struct {
	entered chan struct{}
	release chan struct{}
}

func (fake blockingFirstRail) Fetch(ctx context.Context, _, _ string, _ int) (app.DisplayBoard, error) {
	close(fake.entered)
	select {
	case <-fake.release:
		return app.DisplayBoard{}, nil
	case <-ctx.Done():
		return app.DisplayBoard{}, ctx.Err()
	}
}
func TestDisplayCommuteReportsTimezoneBeforeRailCompletes(t *testing.T) {
	schedule := app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}
	failure := errors.New("settings unavailable")
	rail := blockingFirstRail{entered: make(chan struct{}), release: make(chan struct{})}
	service := app.NewDisplayCommute(displayPreferencesFake{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: schedule}}, displayTimezoneFake{app.DisplayTimezone{ReadError: failure}}, rail, func() time.Time { return time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC) })
	events := make(chan error, 1)
	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, err := service.Trains(ctx, "token", app.DisplayObserver{TimezoneReadError: func(err error) { events <- err }})
		done <- err
	}()
	select {
	case <-rail.entered:
	case <-time.After(time.Second):
		t.Fatal("rail fetch did not start")
	}
	select {
	case got := <-events:
		if !errors.Is(got, failure) {
			t.Fatalf("error %v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timezone error not reported while rail blocked")
	}
	close(rail.release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("operation did not finish")
	}
}
