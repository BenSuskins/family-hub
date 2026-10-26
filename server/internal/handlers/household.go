package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/repository"
)

type WeatherProvider interface {
	Forecast(context.Context, models.HomeInfo) (models.Weather, error)
}
type DepartureProvider interface {
	Departures(context.Context, models.HomeInfo) (models.Departures, error)
}

type HouseholdHandler struct {
	settings   repository.SettingsRepository
	weather    WeatherProvider
	departures DepartureProvider
	shopping   repository.ShoppingRepository
}

func NewHouseholdHandler(settings repository.SettingsRepository, weather WeatherProvider, departures DepartureProvider, shopping repository.ShoppingRepository) *HouseholdHandler {
	return &HouseholdHandler{settings: settings, weather: weather, departures: departures, shopping: shopping}
}

func (handler *HouseholdHandler) loadHome(ctx context.Context) (models.HomeInfo, error) {
	home := models.HomeInfo{ArrivalMarginMinutes: 10, Timezone: "Europe/London"}
	value, err := handler.settings.Get(ctx, "home_info")
	if errors.Is(err, sql.ErrNoRows) {
		return home, nil
	}
	if err != nil {
		return home, err
	}
	err = json.Unmarshal([]byte(value), &home)
	return home, err
}

func (handler *HouseholdHandler) HomeInfo(w http.ResponseWriter, r *http.Request) {
	home, err := handler.loadHome(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	familyName, nameErr := handler.settings.Get(r.Context(), repository.SettingsKeyFamilyName)
	if nameErr != nil || strings.TrimSpace(familyName) == "" {
		familyName = "Family"
	}
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
	home, err := handler.loadHome(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load home information", err)
		return
	}
	if !decodeJSONBody(w, r, &home) {
		return
	}
	if home.Latitude == nil || home.Longitude == nil || math.IsNaN(*home.Latitude) || math.IsNaN(*home.Longitude) || math.Abs(*home.Latitude) > 90 || math.Abs(*home.Longitude) > 180 || home.ArrivalMarginMinutes < 0 || home.ArrivalMarginMinutes > 120 {
		writeJSONError(w, http.StatusBadRequest, "valid coordinates and an arrival margin from 0 to 120 are required")
		return
	}
	if _, err := time.LoadLocation(home.Timezone); err != nil || home.Timezone == "" || home.Timezone == "Local" {
		writeJSONError(w, http.StatusBadRequest, "valid timezone required")
		return
	}
	value, _ := json.Marshal(home)
	if err := handler.settings.Set(r.Context(), "home_info", string(value)); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to save home information", err)
		return
	}
	writeJSON(w, http.StatusOK, home)
}

func (handler *HouseholdHandler) Weather(w http.ResponseWriter, r *http.Request) {
	home, err := handler.loadHome(r.Context())
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
	weather, err := handler.weather.Forecast(r.Context(), home)
	if err != nil {
		observability.RecordDependencyFailure(r, "weather")
		writeAPIError(w, r, http.StatusBadGateway, "weather unavailable", err)
		return
	}
	writeJSON(w, http.StatusOK, weather)
}

func (handler *HouseholdHandler) Departures(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	home, err := handler.loadHome(r.Context())
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
	departures, err := handler.departures.Departures(r.Context(), home)
	if err != nil {
		observability.RecordDependencyFailure(r, "departures")
		writeAPIError(w, r, http.StatusBadGateway, "departures unavailable", err)
		return
	}
	writeJSON(w, http.StatusOK, departures)
}
