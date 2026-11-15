package handlers

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	notifications "github.com/bensuskins/family-hub/internal/application/notifications"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/go-chi/chi/v5"
)

type ChoreReminderHandler struct {
	service       *notifications.ReminderManagement
	pushAvailable bool
}

func NewChoreReminderHandler(service *notifications.ReminderManagement, pushAvailable bool) *ChoreReminderHandler {
	return &ChoreReminderHandler{service: service, pushAvailable: pushAvailable}
}

func (handler *ChoreReminderHandler) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := handler.service.Settings(r.Context())
	if err != nil {
		writeAPIError(w, r, 500, "failed to load chore reminders", err)
		return
	}
	settings.PushAvailable = handler.pushAvailable
	writeJSON(w, 200, settings)
}

func (handler *ChoreReminderHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool     `json:"enabled"`
		Times   []string `json:"times"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if len(body.Times) != 2 || body.Times[0] == body.Times[1] {
		writeJSONError(w, 400, "two distinct reminder times required")
		return
	}
	for _, clock := range body.Times {
		parsed, err := time.Parse("15:04", clock)
		if err != nil || parsed.Format("15:04") != clock {
			writeJSONError(w, 400, "times must use HH:mm")
			return
		}
	}
	if body.Enabled && !handler.pushAvailable {
		writeJSONError(w, 503, "push notifications are not configured on the server")
		return
	}
	if err := handler.service.SaveSettings(r.Context(), body.Enabled, body.Times); err != nil {
		writeAPIError(w, r, 500, "failed to save chore reminders", err)
		return
	}
	handler.Settings(w, r)
}

func (handler *ChoreReminderHandler) Register(w http.ResponseWriter, r *http.Request) {
	var device models.NotificationDevice
	if !decodeJSONBody(w, r, &device) {
		return
	}
	device.ID = chi.URLParam(r, "id")
	device.UserID = middleware.GetUser(r.Context()).ID
	var bearer *string
	if authorization := r.Header.Get("Authorization"); strings.HasPrefix(authorization, "Bearer ") {
		value := strings.TrimPrefix(authorization, "Bearer ")
		bearer = &value
	}
	token, err := hex.DecodeString(device.Token)
	if len(device.ID) == 0 || len(device.ID) > 128 || err != nil || len(token) < 16 || len(token) > 256 || (device.Environment != "sandbox" && device.Environment != "production") {
		writeJSONError(w, 400, "valid device identifier, hexadecimal token, and push environment required")
		return
	}
	device.Token = hex.EncodeToString(token)
	if err := handler.service.Register(r.Context(), device, bearer); err != nil {
		if errors.Is(err, models.ErrDeviceOwnership) {
			writeJSONError(w, 409, "device is registered to another user")
			return
		}
		// Do not log database errors here: constraint messages can contain device tokens.
		writeAPIError(w, r, 500, "failed to register notification device", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *ChoreReminderHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if err := handler.service.Remove(r.Context(), chi.URLParam(r, "id"), middleware.GetUser(r.Context()).ID); err != nil {
		writeAPIError(w, r, 500, "failed to remove notification device", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
