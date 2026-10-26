package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
)

type DisplayCommuteHandler struct {
	pairings repository.DisplayPairingRepository
	settings repository.SettingsRepository
	service  *services.CommuteService
}

func NewDisplayCommuteHandler(pairings repository.DisplayPairingRepository, settings repository.SettingsRepository, service *services.CommuteService) *DisplayCommuteHandler {
	return &DisplayCommuteHandler{pairings: pairings, settings: settings, service: service}
}

func (handler *DisplayCommuteHandler) Trains(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	display, err := handler.pairings.FindByTokenID(request.Context(), middleware.GetDisplayTokenID(request.Context()))
	if err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "could not load display settings", err)
		return
	}
	preferences := display.CommutePreferences
	if !models.IsDisplayStationCode(preferences.HomeStationCode) || !models.IsDisplayStationCode(preferences.WorkStationCode) {
		writeJSON(writer, http.StatusOK, map[string]any{"status": "setup_required", "boards": []models.CommuteBoard{}})
		return
	}
	location := displayTimezone(request, handler.settings)
	now := time.Now().In(location)
	type directionRoute struct {
		schedule            models.DisplayCommuteSchedule
		origin, destination string
	}
	routes := []directionRoute{
		{preferences.ToWork, preferences.HomeStationCode, preferences.WorkStationCode},
		{preferences.ToHome, preferences.WorkStationCode, preferences.HomeStationCode},
	}
	boards := make([]models.CommuteBoard, 0, 2)
	failures := 0
	for _, route := range routes {
		if !activeCommuteWindow(route.schedule, now) {
			continue
		}
		board, err := handler.service.FetchTrains(request.Context(), route.origin, route.destination, preferences.TrainCount)
		if err != nil {
			observability.RecordDependencyFailure(request, "rail")
			slog.Error("fetching trains for display commute failed", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", err)
			failures++
			continue
		}
		boards = append(boards, board)
	}
	status := "available"
	if len(boards) == 0 && failures > 0 {
		status = "unavailable"
	}
	if len(boards) > 0 && failures > 0 {
		status = "partial"
	}
	if len(boards) == 0 && failures == 0 {
		status = "outside_schedule"
	}
	writeJSON(writer, http.StatusOK, map[string]any{"status": status, "timezone": location.String(), "boards": boards})
}

func activeCommuteWindow(schedule models.DisplayCommuteSchedule, now time.Time) bool {
	if !schedule.Enabled {
		return false
	}
	start, startErr := time.Parse("15:04", schedule.Start)
	end, endErr := time.Parse("15:04", schedule.End)
	if startErr != nil || endErr != nil || len(schedule.Weekdays) == 0 {
		return false
	}
	weekday := int(now.Weekday()) + 1
	minute := now.Hour()*60 + now.Minute()
	startMinute := start.Hour()*60 + start.Minute()
	endMinute := end.Hour()*60 + end.Minute()
	selected := func(day int) bool {
		for _, value := range schedule.Weekdays {
			if value == day {
				return true
			}
		}
		return false
	}
	if startMinute < endMinute {
		return selected(weekday) && minute >= startMinute && minute < endMinute
	}
	if minute >= startMinute {
		return selected(weekday)
	}
	previous := weekday - 1
	if previous == 0 {
		previous = 7
	}
	return minute < endMinute && selected(previous)
}

func displayTimezone(request *http.Request, settings repository.SettingsRepository) *time.Location {
	var home models.HomeInfo
	value, err := settings.Get(request.Context(), "home_info")
	if err != nil {
		slog.Error("loading household timezone for display commute", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", err)
	} else if err := json.Unmarshal([]byte(value), &home); err != nil {
		slog.Error("decoding household timezone for display commute", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", err)
	}
	if home.Timezone == "" {
		home.Timezone = "Europe/London"
	}
	location, err := time.LoadLocation(home.Timezone)
	if err != nil {
		slog.Error("loading household timezone for display commute", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", err)
		return time.UTC
	}
	return location
}
