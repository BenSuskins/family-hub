package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bensuskins/family-hub/internal/services"
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
	direction := "to_work"
	if values, exists := query["direction"]; exists {
		if len(values) != 1 || (values[0] != "to_work" && values[0] != "to_home") {
			writeJSONError(writer, http.StatusBadRequest, "direction must be to_work or to_home")
			return
		}
		direction = values[0]
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
	board, err := handler.service.FetchTrains(request.Context(), direction, count)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCommuteUnavailable):
			writeJSONError(writer, http.StatusServiceUnavailable, "train provider is not configured")
		case errors.Is(err, services.ErrCommuteTimeout):
			writeJSONError(writer, http.StatusGatewayTimeout, "train provider timed out")
		case errors.Is(err, context.Canceled):
			return
		default:
			writeJSONError(writer, http.StatusBadGateway, "train provider is unavailable")
		}
		return
	}
	writeJSON(writer, http.StatusOK, board)
}
