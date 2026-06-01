package services

import (
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

func TestCalculateNextDueDate_None(t *testing.T) {
	chore := models.Chore{
		RecurrenceType: models.RecurrenceNone,
	}
	result, err := CalculateNextDueDate(chore, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestCalculateNextDueDate_Daily(t *testing.T) {
	tests := []struct {
		name            string
		recurrenceValue string
		recurOnComplete bool
		dueDate         time.Time
		completedAt     time.Time
		expectedDays    int
	}{
		{
			name:            "daily default interval",
			recurrenceValue: `{}`,
			recurOnComplete: true,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedDays:    1,
		},
		{
			name:            "every 3 days",
			recurrenceValue: `{"interval": 3}`,
			recurOnComplete: true,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedDays:    3,
		},
		{
			name:            "daily fixed schedule from due date",
			recurrenceValue: `{"interval": 1}`,
			recurOnComplete: false,
			dueDate:         time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			completedAt:     time.Date(2025, 1, 16, 10, 0, 0, 0, time.UTC),
			expectedDays:    1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dueDate := test.dueDate
			chore := models.Chore{
				RecurrenceType:  models.RecurrenceDaily,
				RecurrenceValue: test.recurrenceValue,
				RecurOnComplete: test.recurOnComplete,
			}
			if !dueDate.IsZero() {
				chore.DueDate = &dueDate
			}

			result, err := CalculateNextDueDate(chore, test.completedAt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			var baseDate time.Time
			if test.recurOnComplete {
				baseDate = test.completedAt
			} else {
				baseDate = dueDate
			}
			expectedDate := baseDate.AddDate(0, 0, test.expectedDays)

			if test.recurOnComplete {
				if result.Day() != expectedDate.Day() || result.Month() != expectedDate.Month() {
					t.Errorf("expected %v, got %v", expectedDate, *result)
				}
			}
		})
	}
}

func TestCalculateNextDueDate_Weekly(t *testing.T) {
	tests := []struct {
		name            string
		recurrenceValue string
		completedAt     time.Time
		expectedWeekday time.Weekday
	}{
		{
			name:            "weekly default",
			recurrenceValue: `{}`,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			name:            "specific days",
			recurrenceValue: `{"days": ["monday", "friday"]}`,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedWeekday: time.Friday,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chore := models.Chore{
				RecurrenceType:  models.RecurrenceWeekly,
				RecurrenceValue: test.recurrenceValue,
				RecurOnComplete: true,
			}

			result, err := CalculateNextDueDate(chore, test.completedAt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			if result.Before(test.completedAt) {
				t.Errorf("next due date %v should be after completion %v", *result, test.completedAt)
			}

			if test.expectedWeekday != 0 {
				if result.Weekday() != test.expectedWeekday {
					t.Errorf("expected weekday %v, got %v", test.expectedWeekday, result.Weekday())
				}
			}
		})
	}
}

func TestCalculateNextDueDate_Monthly(t *testing.T) {
	chore := models.Chore{
		RecurrenceType:  models.RecurrenceMonthly,
		RecurrenceValue: `{"interval": 1}`,
		RecurOnComplete: true,
	}
	completedAt := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	result, err := CalculateNextDueDate(chore, completedAt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}

	if result.Month() != time.February {
		t.Errorf("expected February, got %v", result.Month())
	}
}

func TestCalculateNextDueDate_Custom(t *testing.T) {
	tests := []struct {
		name            string
		recurrenceValue string
		completedAt     time.Time
		expectedOffset  int
	}{
		{
			name:            "every 5 days",
			recurrenceValue: `{"interval": 5, "unit": "days"}`,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedOffset:  5,
		},
		{
			name:            "every 2 weeks",
			recurrenceValue: `{"interval": 2, "unit": "weeks"}`,
			completedAt:     time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			expectedOffset:  14,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chore := models.Chore{
				RecurrenceType:  models.RecurrenceCustom,
				RecurrenceValue: test.recurrenceValue,
				RecurOnComplete: true,
			}

			result, err := CalculateNextDueDate(chore, test.completedAt)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			expectedDate := test.completedAt.AddDate(0, 0, test.expectedOffset)
			if result.Day() != expectedDate.Day() || result.Month() != expectedDate.Month() {
				t.Errorf("expected %v, got %v", expectedDate, *result)
			}
		})
	}
}

func TestFindNextWeekday(t *testing.T) {
	wednesday := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	result := findNextWeekday(wednesday, []string{"friday"}, 1)
	if result.Weekday() != time.Friday {
		t.Errorf("expected Friday, got %v", result.Weekday())
	}

	result = findNextWeekday(wednesday, []string{"monday"}, 1)
	if result.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", result.Weekday())
	}
	if result.Before(wednesday) {
		t.Errorf("result should be after base date")
	}
}

func TestAdvanceMonthly_Overflow(t *testing.T) {
	tests := []struct {
		name       string
		from       time.Time
		interval   int
		dayOfMonth int
		want       time.Time
	}{
		{
			name:     "jan 31 plus one month clamps to feb 28 (non-leap)",
			from:     time.Date(2025, time.January, 31, 9, 0, 0, 0, time.UTC),
			interval: 1,
			want:     time.Date(2025, time.February, 28, 9, 0, 0, 0, time.UTC),
		},
		{
			name:     "jan 31 plus one month clamps to feb 29 (leap)",
			from:     time.Date(2024, time.January, 31, 9, 0, 0, 0, time.UTC),
			interval: 1,
			want:     time.Date(2024, time.February, 29, 9, 0, 0, 0, time.UTC),
		},
		{
			name:     "mar 31 plus one month clamps to apr 30",
			from:     time.Date(2025, time.March, 31, 9, 0, 0, 0, time.UTC),
			interval: 1,
			want:     time.Date(2025, time.April, 30, 9, 0, 0, 0, time.UTC),
		},
		{
			name:     "jan 15 plus one month stays feb 15 (no skip)",
			from:     time.Date(2025, time.January, 15, 9, 0, 0, 0, time.UTC),
			interval: 1,
			want:     time.Date(2025, time.February, 15, 9, 0, 0, 0, time.UTC),
		},
		{
			name:       "day_of_month 31 in february clamps, never skips to march",
			from:       time.Date(2025, time.January, 15, 9, 0, 0, 0, time.UTC),
			interval:   1,
			dayOfMonth: 31,
			want:       time.Date(2025, time.February, 28, 9, 0, 0, 0, time.UTC),
		},
		{
			name:     "december plus one month rolls year",
			from:     time.Date(2025, time.December, 10, 9, 0, 0, 0, time.UTC),
			interval: 1,
			want:     time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := RecurrenceConfig{Interval: test.interval, DayOfMonth: test.dayOfMonth}
			got := advanceToNextOccurrence(test.from, models.RecurrenceMonthly, config)
			if !got.Equal(test.want) {
				t.Errorf("advanceToNextOccurrence(%v) = %v, want %v", test.from, got, test.want)
			}
		})
	}
}

func TestAdvanceWeekly_MultiDayWithInterval(t *testing.T) {
	// Every 2 weeks on Monday & Thursday, starting Monday 2025-01-06.
	config := RecurrenceConfig{Interval: 2, Days: []string{"monday", "thursday"}}
	cursor := time.Date(2025, time.January, 6, 8, 0, 0, 0, time.UTC) // Monday

	want := []time.Time{
		time.Date(2025, time.January, 9, 8, 0, 0, 0, time.UTC),  // Thu, same fortnight
		time.Date(2025, time.January, 20, 8, 0, 0, 0, time.UTC), // Mon, +1 week gap
		time.Date(2025, time.January, 23, 8, 0, 0, 0, time.UTC), // Thu
		time.Date(2025, time.February, 3, 8, 0, 0, 0, time.UTC), // Mon, +1 week gap
	}

	for i, w := range want {
		cursor = advanceToNextOccurrence(cursor, models.RecurrenceWeekly, config)
		if !cursor.Equal(w) {
			t.Fatalf("occurrence %d = %v (%s), want %v (%s)", i, cursor, cursor.Weekday(), w, w.Weekday())
		}
	}
}

func TestAdvanceWeekly_SingleDayWithInterval(t *testing.T) {
	// Every 2 weeks on Monday.
	config := RecurrenceConfig{Interval: 2, Days: []string{"monday"}}
	from := time.Date(2025, time.January, 6, 8, 0, 0, 0, time.UTC) // Monday
	got := advanceToNextOccurrence(from, models.RecurrenceWeekly, config)
	want := time.Date(2025, time.January, 20, 8, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

