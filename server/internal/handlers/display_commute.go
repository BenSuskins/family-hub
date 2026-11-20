package handlers

import (
	"log/slog"
	"net/http"
	"time"

	commuteapp "github.com/bensuskins/family-hub/internal/application/commute"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/observability"
)

type DisplayCommuteHandler struct{ service *commuteapp.DisplayCommute }

func NewDisplayCommuteHandler(service *commuteapp.DisplayCommute) *DisplayCommuteHandler {
	return &DisplayCommuteHandler{service}
}
func (handler *DisplayCommuteHandler) Trains(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")

	logTimezone := func(message string, err error) {
		slog.Error(message, "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", err)
	}
	observer := commuteapp.DisplayObserver{
		RailFailure: func(failure error) {
			observability.RecordDependencyFailure(request, "rail")
			slog.Error("fetching trains for display commute failed", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusOK, "error", failure)
		},
		TimezoneReadError:     func(err error) { logTimezone("loading household timezone for display commute", err) },
		TimezoneDecodeError:   func(err error) { logTimezone("decoding household timezone for display commute", err) },
		TimezoneLocationError: func(err error) { logTimezone("loading household timezone for display commute", err) },
	}
	result, err := handler.service.Trains(request.Context(), middleware.GetDisplayTokenID(request.Context()), observer)
	if err != nil {
		writeAPIError(writer, request, http.StatusInternalServerError, "could not load display settings", err)
		return
	}

	boards := make([]displayBoardResponse, len(result.Boards))
	for i, board := range result.Boards {
		boards[i] = displayBoardResponseFor(board)
	}
	response := map[string]any{"status": result.Status, "boards": boards}
	if result.Status != "setup_required" {
		response["timezone"] = result.Timezone
	}
	writeJSON(writer, http.StatusOK, response)
}

type displayStationResponse struct {
	Code string `json:"code"`
}
type displayTrainResponse struct {
	ServiceID          string  `json:"serviceId"`
	Operator           string  `json:"operator"`
	Platform           *string `json:"platform"`
	Status             string  `json:"status"`
	ScheduledDeparture *string `json:"scheduledDeparture"`
	ExpectedDeparture  *string `json:"expectedDeparture"`
	ScheduledArrival   *string `json:"scheduledArrival"`
	ExpectedArrival    *string `json:"expectedArrival"`
	CancellationReason *string `json:"cancellationReason"`
	DelayReason        *string `json:"delayReason"`
}
type displayBoardResponse struct {
	Origin      displayStationResponse `json:"origin"`
	Destination displayStationResponse `json:"destination"`
	GeneratedAt time.Time              `json:"generatedAt"`
	FetchedAt   time.Time              `json:"fetchedAt"`
	Timezone    string                 `json:"timezone"`
	Trains      []displayTrainResponse `json:"trains"`
}

func displayBoardResponseFor(board commuteapp.DisplayBoard) displayBoardResponse {
	response := displayBoardResponse{Origin: displayStationResponse{board.Origin}, Destination: displayStationResponse{board.Destination}, GeneratedAt: board.GeneratedAt, FetchedAt: board.FetchedAt, Timezone: board.Timezone}
	if board.Trains != nil {
		response.Trains = make([]displayTrainResponse, len(board.Trains))
		for i, train := range board.Trains {
			response.Trains[i] = displayTrainResponse{train.ServiceID, train.Operator, train.Platform, train.Status, train.ScheduledDeparture, train.ExpectedDeparture, train.ScheduledArrival, train.ExpectedArrival, train.CancellationReason, train.DelayReason}
		}
	}
	return response
}
