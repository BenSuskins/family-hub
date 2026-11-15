package dashboard

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

type fakeSources struct {
	calls          []string
	location       *time.Location
	collections    []BinCollection
	today, overdue []Chore
	week, meals    []Meal
	errAt          string
	from, to, date string
}

func (fake *fakeSources) fail(name string) error {
	fake.calls = append(fake.calls, name)
	if fake.errAt == name {
		return errors.New(name)
	}
	return nil
}
func (fake *fakeSources) Location(context.Context) (*time.Location, error) {
	return fake.location, fake.fail("location")
}
func (fake *fakeSources) Collections(context.Context) ([]BinCollection, error) {
	return fake.collections, fake.fail("bins")
}
func (fake *fakeSources) Due(context.Context) ([]Chore, []Chore, error) {
	return fake.today, fake.overdue, fake.fail("chores")
}
func (fake *fakeSources) Between(_ context.Context, from, to string) ([]Meal, error) {
	fake.from, fake.to = from, to
	return fake.week, fake.fail("week")
}
func (fake *fakeSources) On(_ context.Context, date string) ([]Meal, error) {
	fake.date = date
	return fake.meals, fake.fail("today")
}
func TestDashboardOrdersSourcesAndUsesExistingDateRules(t *testing.T) {
	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 3, 31, 23, 30, 0, 0, time.UTC)
	fake := &fakeSources{location: location, collections: []BinCollection{{Date: "2024-04-02", Name: "Food"}, {Date: "2024-04-01", Name: "Other"}}, today: []Chore{{ID: "chore"}}, week: []Meal{{Name: "Lunch"}}}
	result, err := New(fake, fake, fake, fake, func() time.Time { return now }).Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.calls, []string{"location", "bins", "chores", "week", "today"}) {
		t.Fatalf("calls: %v", fake.calls)
	}
	if fake.from != "2024-03-31" || fake.to != "2024-04-07" || fake.date != "2024-03-31" {
		t.Fatalf("meal dates %s %s %s", fake.from, fake.to, fake.date)
	}
	if len(result.BinCollectionsTomorrow) != 1 || result.BinCollectionsTomorrow[0].Name != "Food" || len(result.ChoresToday) != 1 || result.MealsThisWeek != 1 {
		t.Fatalf("stats: %+v", result)
	}
	if result.ChoresOverdue == nil || result.TodayMeals == nil {
		t.Fatal("empty output is nil")
	}
}

func TestDashboardStopsAtFirstFailure(t *testing.T) {
	tests := []struct {
		name   string
		target error
		calls  []string
	}{
		{"location", ErrLocation, []string{"location"}},
		{"bins", ErrBins, []string{"location", "bins"}},
		{"chores", ErrChores, []string{"location", "bins", "chores"}},
		{"week", ErrWeeklyMeals, []string{"location", "bins", "chores", "week"}},
		{"today", ErrTodayMeals, []string{"location", "bins", "chores", "week", "today"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeSources{location: time.UTC, errAt: test.name}
			_, err := New(fake, fake, fake, fake, time.Now).Stats(context.Background())
			if !errors.Is(err, test.target) {
				t.Fatalf("error %v, want %v", err, test.target)
			}
			if !reflect.DeepEqual(fake.calls, test.calls) {
				t.Fatalf("calls %v, want %v", fake.calls, test.calls)
			}
		})
	}
}

func TestDashboardCopiesMutableValues(t *testing.T) {
	id := "person"
	recipe := "recipe"
	date := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	fake := &fakeSources{today: []Chore{{AssignedToUserID: &id, DueDate: &date, EligibleAssignees: []string{"person"}}}, meals: []Meal{{RecipeID: &recipe}}}
	result, err := New(nil, nil, fake, fake, time.Now).Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*result.ChoresToday[0].AssignedToUserID = "other"
	*result.ChoresToday[0].DueDate = result.ChoresToday[0].DueDate.Add(time.Hour)
	result.ChoresToday[0].EligibleAssignees[0] = "other"
	*result.TodayMeals[0].RecipeID = "other"
	if id != "person" || date.Hour() != 0 || fake.today[0].EligibleAssignees[0] != "person" || recipe != "recipe" {
		t.Fatal("dashboard exposed source state")
	}
}

func TestDashboardWithoutOptionalSourcesReturnsEmptyLists(t *testing.T) {
	fake := &fakeSources{}
	result, err := New(nil, nil, fake, nil, time.Now).Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fake.calls, []string{"chores"}) {
		t.Fatalf("calls: %v", fake.calls)
	}
	if result.ChoresToday == nil || result.ChoresOverdue == nil || result.TodayMeals == nil || result.BinCollectionsTomorrow == nil {
		t.Fatal("expected non-nil lists")
	}
}
