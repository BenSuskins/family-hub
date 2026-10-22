package handlers

import (
	"encoding/json"

	"github.com/bensuskins/family-hub/internal/models"
)

type recurrenceConfigJSON struct {
	Interval   int      `json:"interval,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	Days       []string `json:"days,omitempty"`
	DayOfMonth int      `json:"day_of_month,omitempty"`
}

func buildRecurrenceValueFrom(recurrenceType models.RecurrenceType, interval int, days []string, dayOfMonth int, unit string) string {
	if recurrenceType == models.RecurrenceNone || recurrenceType == models.RecurrenceDaily {
		return ""
	}

	config := recurrenceConfigJSON{Interval: interval}
	if config.Interval == 0 {
		config.Interval = 1
	}

	switch recurrenceType {
	case models.RecurrenceWeekly:
		config.Days = days
	case models.RecurrenceMonthly:
		if dayOfMonth >= 1 && dayOfMonth <= 31 {
			config.DayOfMonth = dayOfMonth
		}
	case models.RecurrenceCustom:
		config.Unit = unit
		if config.Unit == "" {
			config.Unit = "days"
		}
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(encoded)
}
