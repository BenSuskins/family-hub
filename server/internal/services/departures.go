package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	commuteapp "github.com/bensuskins/family-hub/internal/application/commute"
	"github.com/bensuskins/family-hub/internal/models"
)

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
		return 0, commuteapp.ErrUnresolvedDestination
	}
	if places.Status != "OK" {
		return 0, errors.New("geocoding unavailable")
	}
	if len(places.Results) != 1 || places.Results[0].Partial || places.Results[0].PlaceID == "" {
		return 0, commuteapp.ErrUnresolvedDestination
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
		return 0, commuteapp.ErrUnresolvedDestination
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
		return 0, commuteapp.ErrUnresolvedDestination
	}
	duration, err := time.ParseDuration(response.Routes[0].Duration)
	if err != nil || duration <= 0 || duration > 7*24*time.Hour {
		return 0, errors.New("invalid driving duration")
	}
	seconds := int((duration + time.Second - 1) / time.Second)
	return seconds, nil
}
