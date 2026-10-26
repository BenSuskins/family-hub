package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
)

type CommuteHandler struct {
	service *services.CommuteService
}

func NewCommuteHandler(service *services.CommuteService) *CommuteHandler {
	return &CommuteHandler{service: service}
}

func (handler *CommuteHandler) ListTrains(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		writeJSONError(writer, http.StatusBadRequest, "invalid query parameters")
		return
	}
	count := 5
	if values, exists := query["count"]; exists {
		if len(values) != 1 {
			writeJSONError(writer, http.StatusBadRequest, "count must be an integer from 1 to 10")
			return
		}
		count, err = strconv.Atoi(values[0])
		if err != nil || count < 1 || count > services.MaxCommuteTrains {
			writeJSONError(writer, http.StatusBadRequest, "count must be an integer from 1 to 10")
			return
		}
	}
	board, err := handler.service.FetchTrains(request.Context(), chi.URLParam(request, "origin"), chi.URLParam(request, "destination"), count)
	if err != nil {
		if errors.Is(err, services.ErrCommuteTimeout) || !errors.Is(err, services.ErrInvalidCommuteRequest) && !errors.Is(err, services.ErrCommuteUnavailable) && !errors.Is(err, context.Canceled) {
			observability.RecordDependencyFailure(request, "rail")
		}
		switch {
		case errors.Is(err, services.ErrInvalidCommuteRequest):
			writeJSONError(writer, http.StatusBadRequest, "origin and destination must be three-letter station codes")
		case errors.Is(err, services.ErrCommuteUnavailable):
			observability.RecordDependencyFailure(request, "rail")
			slog.Error("train provider unavailable", "method", request.Method, "route", observability.RoutePattern(request), "status", http.StatusServiceUnavailable, "reason", "provider_token_not_configured")
			writeJSONError(writer, http.StatusServiceUnavailable, "train provider is not configured")
		case errors.Is(err, services.ErrCommuteTimeout):
			writeAPIError(writer, request, http.StatusGatewayTimeout, "train provider timed out", err)
		case errors.Is(err, context.Canceled):
			return
		default:
			writeAPIError(writer, request, http.StatusBadGateway, "train provider is unavailable", err)
		}
		return
	}
	writeJSON(writer, http.StatusOK, board)
}
