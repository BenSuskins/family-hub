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
