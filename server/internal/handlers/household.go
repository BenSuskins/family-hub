package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	householdapp "github.com/bensuskins/family-hub/internal/application/household"
	shoppingapp "github.com/bensuskins/family-hub/internal/application/shopping"
	householddomain "github.com/bensuskins/family-hub/internal/domain/household"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
)

type WeatherProvider interface {
	Forecast(context.Context, models.HomeInfo) (models.Weather, error)
}
type DepartureProvider interface {
	Departures(context.Context, models.HomeInfo) (models.Departures, error)
}

type HouseholdHandler struct {
	home       *householdapp.Service
	weather    WeatherProvider
	departures DepartureProvider
	shopping   *shoppingapp.Service
}

func NewHouseholdHandler(home *householdapp.Service, weather WeatherProvider, departures DepartureProvider, shopping *shoppingapp.Service) *HouseholdHandler {
	return &HouseholdHandler{home: home, weather: weather, departures: departures, shopping: shopping}
}

func (handler *HouseholdHandler) HomeInfo(w http.ResponseWriter, r *http.Request) {
	home, err := handler.home.Home(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	familyName := handler.home.FamilyName(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"latitude":             home.Latitude,
		"longitude":            home.Longitude,
		"timezone":             home.Timezone,
		"arrivalMarginMinutes": home.ArrivalMarginMinutes,
		"familyName":           familyName,
	})
}

func (handler *HouseholdHandler) UpdateHomeInfo(w http.ResponseWriter, r *http.Request) {
	if middleware.GetUser(r.Context()).Role != models.RoleAdmin {
		writeJSONError(w, http.StatusForbidden, "admin access required")
		return
	}
	home, err := handler.home.Home(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	body := homeResponseFor(home)
	if !decodeJSONBody(w, r, &body) {
		return
	}
	updated, err := handler.home.SaveHome(r.Context(), body.home())
	if err != nil {
		if errors.Is(err, householddomain.ErrInvalidLocation) || errors.Is(err, householddomain.ErrInvalidTimezone) {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeAPIError(w, r, http.StatusInternalServerError, "failed to save home information", err)
		return
	}
	writeJSON(w, http.StatusOK, homeResponseFor(updated))
}

func (handler *HouseholdHandler) Weather(w http.ResponseWriter, r *http.Request) {
	home, err := handler.home.Home(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	if home.Latitude == nil || home.Longitude == nil {
		writeJSON(w, http.StatusOK, models.Weather{Status: "setup_required", Hours: []models.WeatherHour{}, Advice: []string{}})
		return
	}
	if handler.weather == nil {
		slog.Error("weather service unavailable", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusServiceUnavailable, "reason", "weather_provider_not_configured")
		writeJSONError(w, http.StatusServiceUnavailable, "weather unavailable")
		return
	}
	weather, err := handler.weather.Forecast(r.Context(), providerHome(home))
	if err != nil {
		observability.RecordDependencyFailure(r, "weather")
		writeAPIError(w, r, http.StatusBadGateway, "weather unavailable", err)
		return
	}
	writeJSON(w, http.StatusOK, weather)
}

func (handler *HouseholdHandler) Departures(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	home, err := handler.home.Home(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	if home.Latitude == nil || home.Longitude == nil {
		slog.Info("departure estimates unavailable", "reason", "home_location_not_configured", "latitude_configured", home.Latitude != nil, "longitude_configured", home.Longitude != nil)
		writeJSON(w, http.StatusOK, models.Departures{Status: "setup_required", Items: []models.Departure{}})
		return
	}
	if handler.departures == nil {
		slog.Error("departure estimates unavailable", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusServiceUnavailable, "reason", "departure_service_not_configured")
		writeJSONError(w, http.StatusServiceUnavailable, "departures unavailable")
		return
	}
	departures, err := handler.departures.Departures(r.Context(), providerHome(home))
	if err != nil {
		observability.RecordDependencyFailure(r, "departures")
		writeAPIError(w, r, http.StatusBadGateway, "departures unavailable", err)
		return
	}
	writeJSON(w, http.StatusOK, departures)
}

type homeResponse struct {
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	Timezone             string   `json:"timezone"`
	ArrivalMarginMinutes int      `json:"arrivalMarginMinutes"`
}

func homeResponseFor(home householddomain.Home) homeResponse {
	return homeResponse{home.Latitude, home.Longitude, home.Timezone, home.ArrivalMarginMinutes}
}
func (response homeResponse) home() householddomain.Home {
	return householddomain.Home{Latitude: response.Latitude, Longitude: response.Longitude, Timezone: response.Timezone, ArrivalMarginMinutes: response.ArrivalMarginMinutes}
}
func providerHome(home householddomain.Home) models.HomeInfo {
	return models.HomeInfo{Latitude: home.Latitude, Longitude: home.Longitude, Timezone: home.Timezone, ArrivalMarginMinutes: home.ArrivalMarginMinutes}
}
