package handlers

import (
	"encoding/json"
	"testing"
	"time"

	weatherapp "github.com/bensuskins/family-hub/internal/application/weather"
)

func TestWeatherResponsePreservesJSON(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	rain := 25.0
	tests := []struct {
		name  string
		value weatherapp.Forecast
		want  string
	}{
		{"setup", weatherapp.Forecast{Status: "setup_required", Hours: []weatherapp.Hour{}, Advice: []string{}}, `{"status":"setup_required","fetchedAt":"0001-01-01T00:00:00Z","current":{"time":"0001-01-01T00:00:00Z","temperature":0,"code":0,"rainProbability":0,"windSpeed":0},"high":0,"low":0,"hours":[],"days":null,"advice":[]}`},
		{"forecast", weatherapp.Forecast{Status: "ok", FetchedAt: now, Current: weatherapp.Hour{Time: now, Temperature: 12, Code: 3, RainProbability: 4, WindSpeed: 5}, High: 20, Low: 4, Hours: []weatherapp.Hour{{Time: now}}, Days: []weatherapp.Day{{Date: now, RainProbability: &rain}, {Date: now}}, Advice: []string{"rain"}}, `{"status":"ok","fetchedAt":"2026-01-01T12:00:00Z","current":{"time":"2026-01-01T12:00:00Z","temperature":12,"code":3,"rainProbability":4,"windSpeed":5},"high":20,"low":4,"hours":[{"time":"2026-01-01T12:00:00Z","temperature":0,"code":0,"rainProbability":0,"windSpeed":0}],"days":[{"date":"2026-01-01T12:00:00Z","high":0,"low":0,"code":0,"rainProbability":25},{"date":"2026-01-01T12:00:00Z","high":0,"low":0,"code":0}],"advice":["rain"]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(weatherResponseFor(test.value))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != test.want {
				t.Fatalf("got %s, want %s", got, test.want)
			}
		})
	}
}
