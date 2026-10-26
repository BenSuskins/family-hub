package models

import "time"

type HomeInfo struct {
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	Timezone             string   `json:"timezone"`
	ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
}

type WeatherHour struct {
	Time            time.Time `json:"time"`
	Temperature     float64   `json:"temperature"`
	Code            int       `json:"code"`
	RainProbability float64   `json:"rainProbability"`
	WindSpeed       float64   `json:"windSpeed"`
}

type Weather struct {
	Status    string        `json:"status"`
	FetchedAt time.Time     `json:"fetchedAt"`
	Current   WeatherHour   `json:"current"`
	High      float64       `json:"high"`
	Low       float64       `json:"low"`
	Hours     []WeatherHour `json:"hours"`
	Days      []WeatherDay  `json:"days"`
	Advice    []string      `json:"advice"`
}

type WeatherDay struct {
	Date            time.Time `json:"date"`
	High            float64   `json:"high"`
	Low             float64   `json:"low"`
	Code            int       `json:"code"`
	RainProbability *float64  `json:"rainProbability,omitempty"`
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

type ShoppingItem struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	InventoryItemID *string `json:"inventoryItemID"`
	Purchased       bool    `json:"purchased"`
	PurchaseID      *string `json:"purchaseID"`
	Restocked       bool    `json:"restocked"`
	Version         int     `json:"version"`
}

type ShoppingChange struct {
	Name      *string `json:"name"`
	Purchased *bool   `json:"purchased"`
	Version   int     `json:"version"`
}

type ShoppingRestock struct {
	PurchaseID string `json:"purchaseID"`
	Quantity   *int   `json:"quantity"`
	Level      *int   `json:"level"`
}
