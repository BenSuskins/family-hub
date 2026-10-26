package models

import (
	"strings"
	"time"
)

func NormalizeDisplayStationCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func IsDisplayStationCode(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return false
		}
	}
	return true
}

type DisplayCommuteSchedule struct {
	Enabled  bool   `json:"enabled"`
	Weekdays []int  `json:"weekdays"`
	Start    string `json:"start"`
	End      string `json:"end"`
}

type DisplayCommutePreferences struct {
	ToWork          DisplayCommuteSchedule `json:"toWork"`
	ToHome          DisplayCommuteSchedule `json:"toHome"`
	HomeStationCode string                 `json:"homeStationCode"`
	WorkStationCode string                 `json:"workStationCode"`
	TrainCount      int                    `json:"trainCount"`
}

func DefaultDisplayCommutePreferences() DisplayCommutePreferences {
	return DisplayCommutePreferences{
		ToWork:     DisplayCommuteSchedule{Weekdays: []int{2, 3, 4, 5, 6}, Start: "06:30", End: "09:30"},
		ToHome:     DisplayCommuteSchedule{Weekdays: []int{2, 3, 4, 5, 6}, Start: "16:00", End: "19:30"},
		TrainCount: 3,
	}
}

type CommuteStation struct {
	Code string `json:"code"`
}

type CommuteBoard struct {
	Origin      CommuteStation `json:"origin"`
	Destination CommuteStation `json:"destination"`
	GeneratedAt time.Time      `json:"generatedAt"`
	FetchedAt   time.Time      `json:"fetchedAt"`
	Timezone    string         `json:"timezone"`
	Trains      []CommuteTrain `json:"trains"`
}

type CommuteTrain struct {
	ServiceID          string  `json:"serviceId"`
	Operator           string  `json:"operator"`
	Platform           *string `json:"platform"`
	Status             string  `json:"status"`
	ScheduledDeparture *string `json:"scheduledDeparture"`
	ExpectedDeparture  *string `json:"expectedDeparture"`
	ScheduledArrival   *string `json:"scheduledArrival"`
	ExpectedArrival    *string `json:"expectedArrival"`
	CancellationReason *string `json:"cancellationReason"`
	DelayReason        *string `json:"delayReason"`
}
