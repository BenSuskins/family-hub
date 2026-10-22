package models

import "time"

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
