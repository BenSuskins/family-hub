package models

import "time"

type HomeInfo struct {
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	Timezone             string   `json:"timezone"`
	ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
}

type Departure struct {
	EventID        string    `json:"eventID"`
	Title          string    `json:"title"`
	Destination    string    `json:"destination"`
	StartTime      time.Time `json:"startTime"`
	LeaveAt        time.Time `json:"leaveAt"`
	DrivingSeconds int       `json:"drivingSeconds"`
	FetchedAt      time.Time `json:"fetchedAt"`
}

type Departures struct {
	Status          string      `json:"status"`
	Items           []Departure `json:"items"`
	SelectedEventID *string     `json:"selectedEventID"`
}
