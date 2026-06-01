package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type RecurrenceConfig struct {
	Interval int      `json:"interval,omitempty"`
	Unit     string   `json:"unit,omitempty"`
	Days     []string `json:"days,omitempty"`
	DayOfMonth int    `json:"day_of_month,omitempty"`
	Pattern  string   `json:"pattern,omitempty"`
}

func parseConfig(recurrenceValue string) (RecurrenceConfig, error) {
	var config RecurrenceConfig
	if recurrenceValue != "" {
		if err := json.Unmarshal([]byte(recurrenceValue), &config); err != nil {
			return config, fmt.Errorf("parsing recurrence config: %w", err)
		}
	}
	return config, nil
}

func intervalOrDefault(interval int) int {
	if interval <= 0 {
		return 1
	}
	return interval
}

func advanceToNextOccurrence(from time.Time, recurrenceType models.RecurrenceType, config RecurrenceConfig) time.Time {
	interval := intervalOrDefault(config.Interval)

	switch recurrenceType {
	case models.RecurrenceDaily:
		return from.AddDate(0, 0, interval)

	case models.RecurrenceWeekly:
		if len(config.Days) > 0 {
			return findNextWeekday(from, config.Days, interval)
		}
		return from.AddDate(0, 0, 7*interval)

	case models.RecurrenceMonthly:
		return addMonthsClamped(from, interval, config.DayOfMonth)

	case models.RecurrenceCustom:
		switch config.Unit {
		case "days":
			return from.AddDate(0, 0, interval)
		case "weeks":
			return from.AddDate(0, 0, 7*interval)
		case "months":
			return addMonthsClamped(from, interval, config.DayOfMonth)
		default:
			return from.AddDate(0, 0, interval)
		}

	case models.RecurrenceCalendar:
		// Intentional alias of daily for now. Real ICS/calendar-driven
		// scheduling is not yet implemented.
		// TODO: drive `calendar` occurrences from a linked iCal feed.
		return from.AddDate(0, 0, 1)

	default:
		return from
	}
}

// CalculateNextDueDate returns the next due date after a completion. It is only
// valid for RecurOnComplete chores, where the schedule advances relative to the
// completion time. Fixed-schedule (non-RecurOnComplete) chores are materialized
// ahead of time via SeedFutureOccurrences and never route through here.
func CalculateNextDueDate(chore models.Chore, completedAt time.Time) (*time.Time, error) {
	if chore.RecurrenceType == models.RecurrenceNone {
		return nil, nil
	}

	baseDate := completedAt
	if !chore.RecurOnComplete && chore.DueDate != nil {
		baseDate = *chore.DueDate
	}

	config, err := parseConfig(chore.RecurrenceValue)
	if err != nil {
		return nil, err
	}

	nextDate := advanceToNextOccurrence(baseDate, chore.RecurrenceType, config)
	return &nextDate, nil
}

const maxExpansionIterations = 366

// findNextWeekday returns the next occurrence on one of targetDays after `from`.
// interval applies a multi-week gap: when the next matching weekday falls into a
// later calendar week than `from`, the result is pushed forward interval-1 extra
// weeks. This makes "every 2 weeks on Mon & Thu" yield Mon, Thu, then a one-week
// gap, then Mon, Thu again.
func findNextWeekday(from time.Time, targetDays []string, interval int) time.Time {
	interval = intervalOrDefault(interval)

	dayMap := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	var targets []time.Weekday
	for _, day := range targetDays {
		if weekday, ok := dayMap[day]; ok {
			targets = append(targets, weekday)
		}
	}

	if len(targets) == 0 {
		return from.AddDate(0, 0, 7*interval)
	}

	for offset := 1; offset <= 7; offset++ {
		candidate := from.AddDate(0, 0, offset)
		for _, target := range targets {
			if candidate.Weekday() == target {
				if interval > 1 && startOfWeek(candidate).After(startOfWeek(from)) {
					candidate = candidate.AddDate(0, 0, 7*(interval-1))
				}
				return candidate
			}
		}
	}

	return from.AddDate(0, 0, 7*interval)
}

// startOfWeek returns midnight of the Sunday that begins t's calendar week.
func startOfWeek(t time.Time) time.Time {
	day := t.AddDate(0, 0, -int(t.Weekday()))
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, t.Location())
}

// daysInMonth returns the number of days in the given month.
func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// addMonthsClamped advances `from` by the given number of months without the
// month-overflow that time.AddDate produces (e.g. Jan 31 + 1 month -> Mar 3).
// The target day is dayOfMonth when positive, otherwise from's day, and is
// clamped to the last valid day of the target month.
func addMonthsClamped(from time.Time, months, dayOfMonth int) time.Time {
	total := int(from.Month()) - 1 + months
	year := from.Year() + total/12
	month := time.Month(total%12 + 1)
	if total%12 < 0 {
		year--
		month += 12
	}

	day := dayOfMonth
	if day <= 0 {
		day = from.Day()
	}
	if max := daysInMonth(year, month); day > max {
		day = max
	}

	return time.Date(year, month, day, from.Hour(), from.Minute(), 0, 0, from.Location())
}
