package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var ErrUnresolvedDestination = errors.New("destination is unresolved")

type DepartureCalendar interface {
	FetchForRange(context.Context, time.Time, time.Time) ([]models.Event, error)
}
type DrivingProvider interface {
	DrivingSeconds(context.Context, models.HomeInfo, string) (int, error)
}
type DepartureService struct {
	calendar DepartureCalendar
	driving  DrivingProvider
	now      func() time.Time
}

func NewDepartureService(calendar DepartureCalendar, driving DrivingProvider, now func() time.Time) *DepartureService {
	return &DepartureService{calendar: calendar, driving: driving, now: now}
}
func (service *DepartureService) Departures(ctx context.Context, home models.HomeInfo) (models.Departures, error) {
	result := models.Departures{Status: "ok", Items: []models.Departure{}}
	location, err := time.LoadLocation(home.Timezone)
	if err != nil {
		return result, errors.New("invalid household timezone")
	}
	now := service.now()
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	end := start.AddDate(0, 0, 1)
	events, err := service.calendar.FetchForRange(ctx, start, end)
	if err != nil {
		return result, errors.New("calendar unavailable")
	}
	sort.SliceStable(events, func(first, second int) bool { return events[first].StartTime.Before(events[second].StartTime) })
	for _, event := range events {
		if event.AllDay || !event.StartTime.After(now) || !event.StartTime.Before(end) || !physicalDestination(event.Location) {
			continue
		}
		seconds, err := service.driving.DrivingSeconds(ctx, home, event.Location)
		if errors.Is(err, ErrUnresolvedDestination) {
			continue
		}
		if err != nil {
			return models.Departures{}, err
		}
		departure := models.Departure{EventID: event.ID, Title: event.Title, Destination: event.Location, StartTime: event.StartTime, DrivingSeconds: seconds, FetchedAt: now, LeaveAt: event.StartTime.Add(-time.Duration(seconds)*time.Second - time.Duration(home.ArrivalMarginMinutes)*time.Minute)}
		result.Items = append(result.Items, departure)
		if result.SelectedEventID == nil {
			identifier := event.ID
			result.SelectedEventID = &identifier
		}
	}
	return result, nil
}
func physicalDestination(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	for _, online := range []string{"http://", "https://", "zoom", "teams.microsoft", "meet.google", "online", "virtual", "microsoft teams", "telephone", "phone call"} {
		if strings.Contains(value, online) {
			return false
		}
	}
	return true
}

type GoogleDriving struct {
	client       *http.Client
	key          string
	geocodingURL string
	routesURL    string
}

func NewGoogleDriving(client *http.Client, key, geocodingURL, routesURL string) *GoogleDriving {
	return &GoogleDriving{client: client, key: key, geocodingURL: geocodingURL, routesURL: routesURL}
}
func (provider *GoogleDriving) DrivingSeconds(ctx context.Context, home models.HomeInfo, destination string) (int, error) {
	if provider.key == "" {
		return 0, errors.New("route provider is not configured")
	}
	if home.Latitude == nil || home.Longitude == nil {
		return 0, errors.New("home location missing")
	}
	parameters := url.Values{"address": {destination}, "key": {provider.key}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.geocodingURL+"?"+parameters.Encode(), nil)
	if err != nil {
		return 0, errors.New("invalid geocoding endpoint")
	}
	var places struct {
		Status  string `json:"status"`
		Results []struct {
			Partial    bool     `json:"partial_match"`
			PlaceID    string   `json:"place_id"`
			Types      []string `json:"types"`
			Components []struct {
				Types []string `json:"types"`
			} `json:"address_components"`
		} `json:"results"`
	}
	if err := readProviderJSON(provider.client, request, &places); err != nil {
		return 0, err
	}
	if places.Status == "ZERO_RESULTS" {
		return 0, ErrUnresolvedDestination
	}
	if places.Status != "OK" {
		return 0, errors.New("geocoding unavailable")
	}
	if len(places.Results) != 1 || places.Results[0].Partial || places.Results[0].PlaceID == "" {
		return 0, ErrUnresolvedDestination
	}
	place := places.Results[0]
	complete := false
	for _, component := range place.Components {
		for _, kind := range component.Types {
			if kind == "street_number" {
				complete = true
			}
		}
	}
	if !complete {
		return 0, ErrUnresolvedDestination
	}
	body, _ := json.Marshal(map[string]any{
		"origin":      map[string]any{"location": map[string]any{"latLng": map[string]float64{"latitude": *home.Latitude, "longitude": *home.Longitude}}},
		"destination": map[string]string{"placeId": place.PlaceID},
		"travelMode":  "DRIVE", "routingPreference": "TRAFFIC_AWARE", "computeAlternativeRoutes": false,
	})
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, provider.routesURL, bytes.NewReader(body))
	if err != nil {
		return 0, errors.New("invalid routes endpoint")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Goog-Api-Key", provider.key)
	request.Header.Set("X-Goog-FieldMask", "routes.duration")
	var response struct {
		Routes []struct {
			Duration string `json:"duration"`
		} `json:"routes"`
	}
	if err := readProviderJSON(provider.client, request, &response); err != nil {
		return 0, err
	}
	if len(response.Routes) == 0 {
		return 0, ErrUnresolvedDestination
	}
	duration, err := time.ParseDuration(response.Routes[0].Duration)
	if err != nil || duration <= 0 || duration > 7*24*time.Hour {
		return 0, errors.New("invalid driving duration")
	}
	seconds := int((duration + time.Second - 1) / time.Second)
	return seconds, nil
}
