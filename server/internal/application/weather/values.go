package weather

import "time"

type Location struct {
	Latitude, Longitude *float64
	Timezone            string
}

type Hour struct {
	Time            time.Time
	Temperature     float64
	Code            int
	RainProbability float64
	WindSpeed       float64
}

type Day struct {
	Date            time.Time
	High, Low       float64
	Code            int
	RainProbability *float64
}

type Forecast struct {
	Status    string
	FetchedAt time.Time
	Current   Hour
	High, Low float64
	Hours     []Hour
	Days      []Day
	Advice    []string
}
