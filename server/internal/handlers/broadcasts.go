package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/google/uuid"
)

type BroadcastHandler struct {
	service   *services.BroadcastService
	available bool
}

func NewBroadcastHandler(service *services.BroadcastService, available bool) *BroadcastHandler {
	return &BroadcastHandler{service: service, available: available}
}

func (handler *BroadcastHandler) Send(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RequestID string `json:"requestID"`
		Message   string `json:"message"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	body.Message = strings.TrimSpace(body.Message)
	if _, err := uuid.Parse(body.RequestID); err != nil || body.Message == "" || utf8.RuneCountInString(body.Message) > 500 {
		writeJSONError(w, 400, "a request UUID and a message of 1 to 500 characters are required")
		return
	}
	if !handler.available {
		writeJSONError(w, 503, "push notifications are not configured on the server")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	result, err := handler.service.Send(ctx, body.RequestID, middleware.GetUser(r.Context()).ID, body.Message, time.Now())
	if err != nil {
		if errors.Is(err, models.ErrBroadcastConflict) {
			writeJSONError(w, 409, "request identifier already used for another broadcast")
			return
		}
		writeAPIError(w, r, 500, "failed to send family broadcast", err)
		return
	}
	writeJSON(w, 200, result)
}
