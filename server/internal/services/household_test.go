package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bensuskins/family-hub/internal/models"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWeatherForecastAdviceAndExpiry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests > 1 {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, weatherFixture(time.Unix(1792324800, 0), 24))
	}))
	defer server.Close()
	now := time.Unix(1792324800, 0)
	provider := NewOpenMeteo(server.Client(), server.URL, func() time.Time { return now })
	latitude, longitude := 51.8, 0.7
	home := models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London"}
	forecast, err := provider.Forecast(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	if forecast.Current.Temperature != 9 || len(forecast.Hours) != 24 || len(forecast.Advice) != 3 {
		t.Fatalf("unexpected forecast: %+v", forecast)
	}
	if _, err := provider.Forecast(context.Background(), home); err != nil || requests != 1 {
		t.Fatal("fresh weather must be reused")
	}
	now = now.Add(16 * time.Minute)
	if _, err := provider.Forecast(context.Background(), home); err == nil {
		t.Fatal("expired weather must not hide provider failure")
	}
}

type fakeDepartureCalendar struct{ events []models.Event }

func (calendar fakeDepartureCalendar) FetchForRange(context.Context, time.Time, time.Time) ([]models.Event, error) {
	return calendar.events, nil
}

type fakeDrivingProvider struct{}

func (fakeDrivingProvider) DrivingSeconds(_ context.Context, _ models.HomeInfo, destination string) (int, error) {
	if destination == "School" {
		return 0, ErrUnresolvedDestination
	}
	return 1200, nil
}
func TestDeparturesSelectStartTimeAndSubtractMargin(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	events := []models.Event{
		{ID: "all-day", AllDay: true, Location: "10 Main Street", StartTime: now.Add(time.Hour)},
		{ID: "online", Location: "https://zoom.us/meeting", StartTime: now.Add(time.Hour)},
		{ID: "vague", Location: "School", StartTime: now.Add(time.Hour)},
		{ID: "later", Title: "Later", Location: "20 Main Street", StartTime: now.Add(2 * time.Hour)},
		{ID: "next", Title: "Next", Location: "10 Main Street", StartTime: now.Add(90 * time.Minute)},
		{ID: "started", Location: "10 Main Street", StartTime: now.Add(-time.Minute)},
	}
	latitude, longitude := 51.8, 0.7
	service := NewDepartureService(fakeDepartureCalendar{events}, fakeDrivingProvider{}, func() time.Time { return now })
	result, err := service.Departures(context.Background(), models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London", ArrivalMarginMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.SelectedEventID == nil || *result.SelectedEventID != "next" || len(result.Items) != 2 {
		t.Fatalf("unexpected departures: %+v", result)
	}
	if !result.Items[0].LeaveAt.Equal(time.Date(2026, 10, 18, 13, 0, 0, 0, time.UTC)) {
		t.Fatalf("wrong leave time: %s", result.Items[0].LeaveAt)
	}
}

func TestGoogleDrivingContract(t *testing.T) {
	latitude, longitude := 51.8, 0.7
	home := models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London"}
	for _, example := range []struct {
		name, geocode, route string
		status               int
		seconds              int
		unresolved           bool
	}{
		{"valid", `{"status":"OK","results":[{"place_id":"place-one","address_components":[{"types":["street_number"]}]}]}`, `{"routes":[{"duration":"1200.5s"}]}`, 200, 1201, false},
		{"no match", `{"status":"ZERO_RESULTS","results":[]}`, `{}`, 200, 0, true},
		{"ambiguous", `{"status":"OK","results":[{"place_id":"one"},{"place_id":"two"}]}`, `{}`, 200, 0, true},
		{"partial", `{"status":"OK","results":[{"place_id":"one","partial_match":true}]}`, `{}`, 200, 0, true},
		{"vague", `{"status":"OK","results":[{"place_id":"school"}]}`, `{}`, 200, 0, true},
		{"denied", `{"status":"REQUEST_DENIED","error_message":"do-not-expose-key"}`, `{}`, 200, 0, false},
		{"http failure", `{}`, `{}`, 503, 0, false},
		{"malformed", `invalid`, `{}`, 200, 0, false},
		{"missing route", `{"status":"OK","results":[{"place_id":"one","address_components":[{"types":["street_number"]}]}]}`, `{"routes":[]}`, 200, 0, true},
	} {
		t.Run(example.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(example.status)
				if r.URL.Path == "/geocode" {
					fmt.Fprint(w, example.geocode)
					return
				}
				if r.Header.Get("X-Goog-Api-Key") != "do-not-expose-key" || r.Header.Get("X-Goog-FieldMask") != "routes.duration" {
					t.Error("missing route credentials or field mask")
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body["travelMode"] != "DRIVE" || body["routingPreference"] != "TRAFFIC_AWARE" {
					t.Error("route must use driving traffic")
				}
				fmt.Fprint(w, example.route)
			}))
			defer server.Close()
			provider := NewGoogleDriving(server.Client(), "do-not-expose-key", server.URL+"/geocode", server.URL+"/routes")
			seconds, err := provider.DrivingSeconds(context.Background(), home, "10 High Street")
			if example.seconds > 0 {
				if err != nil || seconds != example.seconds {
					t.Fatalf("got %d %v", seconds, err)
				}
			} else {
				if err == nil {
					t.Fatal("expected error")
				}
				if errors.Is(err, ErrUnresolvedDestination) != example.unresolved {
					t.Fatalf("wrong error: %v", err)
				}
				if strings.Contains(err.Error(), "do-not-expose-key") {
					t.Fatal("credential exposed")
				}
			}
		})
	}
}

func TestDeparturesRespectLocalDayAcrossClockChanges(t *testing.T) {
	latitude, longitude := 51.8, 0.7
	for _, date := range []string{"2026-03-29T22:30:00Z", "2026-10-25T23:30:00Z"} {
		now, err := time.Parse(time.RFC3339, date)
		if err != nil {
			t.Fatal(err)
		}
		local, _ := time.LoadLocation("Europe/London")
		day := now.In(local)
		end := time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, local)
		events := []models.Event{{ID: "today", Location: "10 High Street", StartTime: now.Add(10 * time.Minute)}, {ID: "tomorrow", Location: "10 High Street", StartTime: end.Add(time.Minute)}}
		service := NewDepartureService(fakeDepartureCalendar{events}, fakeDrivingProvider{}, func() time.Time { return now })
		result, err := service.Departures(context.Background(), models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London"})
		if err != nil || len(result.Items) != 1 || result.Items[0].EventID != "today" {
			t.Fatalf("%s: %+v %v", date, result, err)
		}
	}
}

func weatherFixture(now time.Time, count int) string {
	times := []int64{}
	temperatures := []int{}
	codes := []int{}
	rain := []int{}
	wind := []int{}
	for index := 0; index < count; index++ {
		times = append(times, now.Add(time.Duration(index)*time.Hour).Unix())
		temperatures = append(temperatures, 9)
		codes = append(codes, 3)
		rain = append(rain, 80)
		wind = append(wind, 42)
	}
	data, _ := json.Marshal(map[string]any{"current": map[string]any{"time": now.Unix(), "temperature_2m": 9, "weather_code": 3, "wind_speed_10m": 12}, "hourly": map[string]any{"time": times, "temperature_2m": temperatures, "weather_code": codes, "precipitation_probability": rain, "wind_speed_10m": wind}, "daily": map[string]any{"temperature_2m_max": []int{17}, "temperature_2m_min": []int{8}}})
	return string(data)
}
func TestWeatherRejectsShortAndOldForecasts(t *testing.T) {
	now := time.Unix(1792324800, 0)
	for _, body := range []string{weatherFixture(now, 2), weatherFixture(now.Add(-48*time.Hour), 24), `{"current":null}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		latitude, longitude := 51.8, 0.7
		provider := NewOpenMeteo(server.Client(), server.URL, func() time.Time { return now })
		if _, err := provider.Forecast(context.Background(), models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London"}); err == nil {
			t.Error("incomplete forecast must be unavailable")
		}
		server.Close()
	}
}

func TestHouseholdProvidersLive(t *testing.T) {
	if os.Getenv("RUN_HOUSEHOLD_PROVIDER_CHECK") != "1" {
		t.Skip("set RUN_HOUSEHOLD_PROVIDER_CHECK=1 for the public-location provider check")
	}
	latitude, longitude := 51.5074, -0.1278
	home := models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "Europe/London", ArrivalMarginMinutes: 10}
	client := NewSafeHTTPClient(10 * time.Second)
	weather := NewOpenMeteo(client, "https://api.open-meteo.com/v1/forecast", time.Now)
	forecast, err := weather.Forecast(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	if len(forecast.Hours) != 24 {
		t.Fatal("live weather must contain 24 hours")
	}
	if key := os.Getenv("GOOGLE_MAPS_API_KEY"); key != "" {
		provider := NewGoogleDriving(client, key, "https://maps.googleapis.com/maps/api/geocode/json", "https://routes.googleapis.com/directions/v2:computeRoutes")
		seconds, err := provider.DrivingSeconds(context.Background(), home, "10 Downing Street, London SW1A 2AA, UK")
		if err != nil || seconds <= 0 {
			t.Fatalf("live route unavailable: %v", err)
		}
	} else {
		t.Log("Google check skipped: no runtime key")
	}
}
