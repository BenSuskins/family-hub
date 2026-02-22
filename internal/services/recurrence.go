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

func advanceToNextOccurrence(from time.Time, recurrenceType models.RecurrenceType, config RecurrenceConfig) time.Time {
	switch recurrenceType {
	case models.RecurrenceDaily:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		return from.AddDate(0, 0, interval)

	case models.RecurrenceWeekly:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		if len(config.Days) > 0 {
			return findNextWeekday(from, config.Days)
		}
		return from.AddDate(0, 0, 7*interval)

	case models.RecurrenceMonthly:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		next := from.AddDate(0, interval, 0)
		if config.DayOfMonth > 0 {
			next = time.Date(next.Year(), next.Month(), config.DayOfMonth,
				from.Hour(), from.Minute(), 0, 0, from.Location())
		}
		return next

	case models.RecurrenceCustom:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		switch config.Unit {
		case "days":
			return from.AddDate(0, 0, interval)
		case "weeks":
			return from.AddDate(0, 0, 7*interval)
		case "months":
			return from.AddDate(0, interval, 0)
		default:
			return from.AddDate(0, 0, interval)
		}

	case models.RecurrenceCalendar:
		return from.AddDate(0, 0, 1)

	default:
		return from
	}
}

func CalculateNextDueDate(chore models.Chore, completedAt time.Time) (*time.Time, error) {
	if chore.RecurrenceType == models.RecurrenceNone {
		return nil, nil
	}

	var baseDate time.Time
	if chore.RecurOnComplete {
		baseDate = completedAt
	} else {
		if chore.DueDate != nil {
			baseDate = *chore.DueDate
		} else {
			baseDate = completedAt
		}
	}

	config, err := parseConfig(chore.RecurrenceValue)
	if err != nil {
		return nil, err
	}

	nextDate := advanceToNextOccurrence(baseDate, chore.RecurrenceType, config)

	if !chore.RecurOnComplete {
		now := time.Now()
		for nextDate.Before(now) {
			nextDate = advanceToNextOccurrence(nextDate, chore.RecurrenceType, config)
		}
	}

	return &nextDate, nil
}

const maxExpansionIterations = 366

func findNextWeekday(from time.Time, targetDays []string) time.Time {
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
		return from.AddDate(0, 0, 7)
	}

	for offset := 1; offset <= 7; offset++ {
		candidate := from.AddDate(0, 0, offset)
		for _, target := range targets {
			if candidate.Weekday() == target {
				return candidate
			}
		}
	}

	return from.AddDate(0, 0, 7)
}
