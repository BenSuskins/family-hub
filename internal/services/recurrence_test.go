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

	result := findNextWeekday(wednesday, []string{"friday"})
	if result.Weekday() != time.Friday {
		t.Errorf("expected Friday, got %v", result.Weekday())
	}

	result = findNextWeekday(wednesday, []string{"monday"})
	if result.Weekday() != time.Monday {
		t.Errorf("expected Monday, got %v", result.Weekday())
	}
	if result.Before(wednesday) {
		t.Errorf("result should be after base date")
	}
}

func TestExpandChoreOccurrences(t *testing.T) {
	rangeStart := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	makeChore := func(recurrenceType models.RecurrenceType, recurrenceValue string, dueDate *time.Time) models.Chore {
		return models.Chore{
			ID:              "test-chore",
			Name:            "Test Chore",
			RecurrenceType:  recurrenceType,
			RecurrenceValue: recurrenceValue,
			DueDate:         dueDate,
			Status:          models.ChoreStatusPending,
		}
	}

	date := func(year int, month time.Month, day int) *time.Time {
		t := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
		return &t
	}

	tests := []struct {
		name          string
		chore         models.Chore
		rangeStart    time.Time
		rangeEnd      time.Time
		expectedCount int
		checkDates    []time.Time
	}{
		{
			name:          "non-recurring returns empty",
			chore:         makeChore(models.RecurrenceNone, "", date(2025, 2, 5)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 0,
		},
		{
			name:          "nil due date returns empty",
			chore:         makeChore(models.RecurrenceDaily, "", nil),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 0,
		},
		{
			name:          "daily default interval",
			chore:         makeChore(models.RecurrenceDaily, `{}`, date(2025, 1, 30)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 28, // Feb 1-28 (skips Jan 30 due date, Jan 31 before range)
			checkDates: []time.Time{
				time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "daily with interval 3",
			chore:         makeChore(models.RecurrenceDaily, `{"interval": 3}`, date(2025, 1, 29)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 10, // Jan 29 +3=Feb 1, +6=Feb 4, +9=Feb 7, +12=Feb 10, +15=Feb 13, +18=Feb 16, +21=Feb 19, +24=Feb 22, +27=Feb 25, +30=Feb 28
			checkDates: []time.Time{
				time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "due date before range with gap",
			chore:         makeChore(models.RecurrenceDaily, `{"interval": 5}`, date(2025, 1, 1)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 5, // Jan 1,6,11,16,21,26,31,Feb 5,10,15,20,25 -> in Feb: 5,10,15,20,25
		},
		{
			name:          "weekly default",
			chore:         makeChore(models.RecurrenceWeekly, `{}`, date(2025, 1, 27)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 4, // Feb 3, 10, 17, 24
			checkDates: []time.Time{
				time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 17, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 24, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "weekly specific days mon and fri",
			chore:         makeChore(models.RecurrenceWeekly, `{"days": ["monday", "friday"]}`, date(2025, 1, 31)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 8, // Mon Feb 3, Fri Feb 7, Mon Feb 10, Fri Feb 14, Mon Feb 17, Fri Feb 21, Mon Feb 24, Fri Feb 28
			checkDates: []time.Time{
				time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 7, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "monthly default",
			chore:         makeChore(models.RecurrenceMonthly, `{}`, date(2025, 1, 15)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 1, // Feb 15
			checkDates: []time.Time{
				time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "monthly with day_of_month",
			chore:         makeChore(models.RecurrenceMonthly, `{"day_of_month": 10}`, date(2025, 1, 10)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 1, // Feb 10
			checkDates: []time.Time{
				time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "custom days",
			chore:         makeChore(models.RecurrenceCustom, `{"interval": 10, "unit": "days"}`, date(2025, 1, 22)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 3, // Feb 1, Feb 11, Feb 21
			checkDates: []time.Time{
				time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 11, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "custom weeks",
			chore:         makeChore(models.RecurrenceCustom, `{"interval": 2, "unit": "weeks"}`, date(2025, 1, 18)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 2, // Feb 1, Feb 15
			checkDates: []time.Time{
				time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "custom months",
			chore:         makeChore(models.RecurrenceCustom, `{"interval": 1, "unit": "months"}`, date(2025, 1, 5)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 1, // Feb 5
			checkDates: []time.Time{
				time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "calendar type daily",
			chore:         makeChore(models.RecurrenceCalendar, `{}`, date(2025, 1, 30)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 28, // Feb 1-28
		},
		{
			name:          "skips actual due date",
			chore:         makeChore(models.RecurrenceDaily, `{}`, date(2025, 2, 5)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 23, // Feb 6-28 (skips Feb 5 itself)
		},
		{
			name:          "due date after range returns empty",
			chore:         makeChore(models.RecurrenceDaily, `{}`, date(2025, 4, 1)),
			rangeStart:    rangeStart,
			rangeEnd:      rangeEnd,
			expectedCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			occurrences, err := ExpandChoreOccurrences(test.chore, test.rangeStart, test.rangeEnd)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(occurrences) != test.expectedCount {
				dates := make([]string, len(occurrences))
				for i, o := range occurrences {
					dates[i] = o.DueDate.Format("2006-01-02")
				}
				t.Fatalf("expected %d occurrences, got %d: %v", test.expectedCount, len(occurrences), dates)
			}
			for i, expectedDate := range test.checkDates {
				if i >= len(occurrences) {
					break
				}
				if occurrences[i].DueDate == nil {
					t.Errorf("occurrence %d has nil due date", i)
					continue
				}
				actual := *occurrences[i].DueDate
				if actual.Year() != expectedDate.Year() || actual.Month() != expectedDate.Month() || actual.Day() != expectedDate.Day() {
					t.Errorf("occurrence %d: expected %v, got %v", i, expectedDate.Format("2006-01-02"), actual.Format("2006-01-02"))
				}
			}
			for _, occurrence := range occurrences {
				if occurrence.ID != test.chore.ID {
					t.Errorf("occurrence ID should match source chore ID")
				}
				if occurrence.Name != test.chore.Name {
					t.Errorf("occurrence Name should match source chore")
				}
			}
		})
	}
}

func TestExpandChoreOccurrences_CapsAt366Iterations(t *testing.T) {
	dueDate := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	chore := models.Chore{
		ID:              "cap-test",
		Name:            "Cap Test",
		RecurrenceType:  models.RecurrenceDaily,
		RecurrenceValue: `{}`,
		DueDate:         &dueDate,
		Status:          models.ChoreStatusPending,
	}
	rangeStart := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	occurrences, err := ExpandChoreOccurrences(chore, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(occurrences) > 366 {
		t.Errorf("expected at most 366 occurrences, got %d", len(occurrences))
	}
}
