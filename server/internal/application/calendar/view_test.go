package calendar

import (
	"context"
	"errors"
	"testing"
	"time"
)

type viewFake struct {
	start, end                       time.Time
	forecastErr, eventsErr, mealsErr error
}

func (fake *viewFake) Location(context.Context) (*time.Location, error) { return time.UTC, nil }
func (fake *viewFake) Chores(_ context.Context, start, end time.Time) ([]ViewChore, error) {
	fake.start = start
	fake.end = end
	return nil, nil
}
func (fake *viewFake) Forecast(context.Context, time.Time, time.Time) ([]ViewChore, error) {
	return nil, fake.forecastErr
}
func (fake *viewFake) Events(context.Context, time.Time, time.Time) ([]ViewEvent, error) {
	return nil, fake.eventsErr
}
func (fake *viewFake) Meals(context.Context, string, string) ([]ViewMeal, error) {
	return nil, fake.mealsErr
}
func TestViewRetainsPartialFailuresAndDayRange(t *testing.T) {
	fake := &viewFake{eventsErr: errors.New("events"), mealsErr: errors.New("meals")}
	service := NewView(fake, fake, fake, fake, func() time.Time { return time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC) })
	result, err := service.List(context.Background(), ViewQuery{View: "day", Date: "2026-04-30"})
	if err != nil || result.EventsError == nil || result.MealsError == nil || result.Chores == nil || result.Events == nil || result.Meals == nil {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if fake.start.Format(time.RFC3339) != "2026-04-30T00:00:00Z" || !fake.end.Equal(time.Date(2026, 4, 30, 23, 59, 59, 999999999, time.UTC)) {
		t.Fatalf("range %v to %v", fake.start, fake.end)
	}
}
func TestViewStopsOnForecastFailure(t *testing.T) {
	fake := &viewFake{forecastErr: errors.New("failure")}
	_, err := NewView(fake, fake, fake, fake, time.Now).List(context.Background(), ViewQuery{})
	if !errors.Is(err, ErrViewForecast) {
		t.Fatalf("got %v", err)
	}
}

type viewCopies struct {
	chore ViewChore
	event ViewEvent
	meal  ViewMeal
}

func (fake *viewCopies) Chores(context.Context, time.Time, time.Time) ([]ViewChore, error) {
	return []ViewChore{fake.chore}, nil
}
func (fake *viewCopies) Forecast(context.Context, time.Time, time.Time) ([]ViewChore, error) {
	return nil, nil
}
func (fake *viewCopies) Events(context.Context, time.Time, time.Time) ([]ViewEvent, error) {
	return []ViewEvent{fake.event}, nil
}
func (fake *viewCopies) Meals(context.Context, string, string) ([]ViewMeal, error) {
	return []ViewMeal{fake.meal}, nil
}
func TestViewDefensivelyCopiesResults(t *testing.T) {
	category, recipe := "category", "recipe"
	end := time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC)
	fake := &viewCopies{chore: ViewChore{CategoryID: &category, EligibleAssignees: []string{"first"}}, event: ViewEvent{EndTime: &end, CategoryID: &category}, meal: ViewMeal{RecipeID: &recipe}}
	result, err := NewView(nil, fake, fake, fake, time.Now).List(context.Background(), ViewQuery{})
	if err != nil {
		t.Fatal(err)
	}
	*result.Chores[0].CategoryID = "changed"
	result.Chores[0].EligibleAssignees[0] = "changed"
	*result.Events[0].CategoryID = "changed"
	*result.Events[0].EndTime = result.Events[0].EndTime.Add(time.Hour)
	*result.Meals[0].RecipeID = "changed"
	if category != "category" || recipe != "recipe" || fake.chore.EligibleAssignees[0] != "first" || end.Hour() != 1 {
		t.Fatal("returned value changed source")
	}
}

func TestViewRangeAndInvalidInputs(t *testing.T) {
	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		query     ViewQuery
		start     string
		last      string
		wantError error
	}{
		{ViewQuery{View: "week", Date: "2026-03-29"}, "2026-03-23T00:00:00Z", "2026-03-29T22:59:59.999999999Z", nil},
		{ViewQuery{View: "unknown", Month: "2026-04"}, "2026-03-31T23:00:00Z", "2026-04-30T22:59:59.999999999Z", nil},
		{ViewQuery{View: "day", Date: "wrong"}, "", "", ErrViewDate},
		{ViewQuery{Month: "wrong"}, "", "", ErrViewMonth},
	}
	for _, test := range tests {
		t.Run(test.query.View+test.query.Date+test.query.Month, func(t *testing.T) {
			fake := &viewFake{}
			_, err := NewView(viewFixedLocation{location}, fake, nil, nil, func() time.Time { return time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC) }).List(context.Background(), test.query)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("err %v", err)
			}
			if test.wantError == nil && (fake.start.UTC().Format(time.RFC3339Nano) != test.start || fake.end.UTC().Format(time.RFC3339Nano) != test.last) {
				t.Fatalf("range %v %v", fake.start, fake.end)
			}
		})
	}
}

type viewFixedLocation struct{ location *time.Location }

func (fixed viewFixedLocation) Location(context.Context) (*time.Location, error) {
	return fixed.location, nil
}

type viewBlockingMeals struct {
	entered chan struct{}
	release chan struct{}
}

func (blocked viewBlockingMeals) Meals(context.Context, string, string) ([]ViewMeal, error) {
	close(blocked.entered)
	<-blocked.release
	return nil, nil
}
func TestViewReportsFeedFailureBeforeBlockingMeals(t *testing.T) {
	fake := &viewFake{eventsErr: errors.New("feed down")}
	blocked := viewBlockingMeals{make(chan struct{}), make(chan struct{})}
	reported := make(chan ViewFailureSource, 1)
	done := make(chan error, 1)
	go func() {
		_, err := NewView(nil, fake, fake, blocked, time.Now).List(context.Background(), ViewQuery{OnFailure: func(source ViewFailureSource, _ error) { reported <- source }})
		done <- err
	}()
	defer func() {
		close(blocked.release)
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("view failed: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("view did not finish")
		}
	}()
	select {
	case <-blocked.entered:
	case <-time.After(time.Second):
		t.Fatal("meals not called")
	}
	select {
	case source := <-reported:
		if source != ViewFailureEvents {
			t.Fatalf("source %q", source)
		}
	default:
		t.Fatal("feed failure not reported before meals blocked")
	}
}
func TestViewReportsLocationFailureBeforeInvalidQuery(t *testing.T) {
	fake := &viewFake{}
	reported := false
	_, err := NewView(viewFailLocation{}, fake, nil, nil, time.Now).List(context.Background(), ViewQuery{View: "day", Date: "bad", OnFailure: func(source ViewFailureSource, _ error) { reported = source == ViewFailureLocation }})
	if !reported || !errors.Is(err, ErrViewDate) {
		t.Fatalf("reported=%v err=%v", reported, err)
	}
}

type viewFailLocation struct{}

func (viewFailLocation) Location(context.Context) (*time.Location, error) {
	return nil, errors.New("location failed")
}
