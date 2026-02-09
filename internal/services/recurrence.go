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

	var config RecurrenceConfig
	if chore.RecurrenceValue != "" {
		if err := json.Unmarshal([]byte(chore.RecurrenceValue), &config); err != nil {
			return nil, fmt.Errorf("parsing recurrence config: %w", err)
		}
	}

	var nextDate time.Time

	switch chore.RecurrenceType {
	case models.RecurrenceDaily:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		nextDate = baseDate.AddDate(0, 0, interval)

	case models.RecurrenceWeekly:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		if len(config.Days) > 0 {
			nextDate = findNextWeekday(baseDate, config.Days)
		} else {
			nextDate = baseDate.AddDate(0, 0, 7*interval)
		}

	case models.RecurrenceMonthly:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		nextDate = baseDate.AddDate(0, interval, 0)
		if config.DayOfMonth > 0 {
			nextDate = time.Date(nextDate.Year(), nextDate.Month(), config.DayOfMonth,
				baseDate.Hour(), baseDate.Minute(), 0, 0, baseDate.Location())
		}

	case models.RecurrenceCustom:
		interval := config.Interval
		if interval <= 0 {
			interval = 1
		}
		switch config.Unit {
		case "days":
			nextDate = baseDate.AddDate(0, 0, interval)
		case "weeks":
			nextDate = baseDate.AddDate(0, 0, 7*interval)
		case "months":
			nextDate = baseDate.AddDate(0, interval, 0)
		default:
			nextDate = baseDate.AddDate(0, 0, interval)
		}

	case models.RecurrenceCalendar:
		nextDate = baseDate.AddDate(0, 0, 1)

	default:
		return nil, nil
	}

	if !chore.RecurOnComplete {
		now := time.Now()
		for nextDate.Before(now) {
			switch chore.RecurrenceType {
			case models.RecurrenceDaily:
				interval := config.Interval
				if interval <= 0 {
					interval = 1
				}
				nextDate = nextDate.AddDate(0, 0, interval)
			case models.RecurrenceWeekly:
				nextDate = nextDate.AddDate(0, 0, 7)
			case models.RecurrenceMonthly:
				nextDate = nextDate.AddDate(0, 1, 0)
			default:
				nextDate = nextDate.AddDate(0, 0, 1)
			}
		}
	}

	return &nextDate, nil
}

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
