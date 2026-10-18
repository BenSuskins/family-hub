package handlers

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
)

type ChoreReminderHandler struct {
	settings      repository.SettingsRepository
	devices       repository.NotificationRepository
	pushAvailable bool
}

func NewChoreReminderHandler(settings repository.SettingsRepository, devices repository.NotificationRepository, pushAvailable bool) *ChoreReminderHandler {
	return &ChoreReminderHandler{settings: settings, devices: devices, pushAvailable: pushAvailable}
}

func (handler *ChoreReminderHandler) Settings(w http.ResponseWriter, r *http.Request) {
	settings, err := services.LoadChoreReminders(r.Context(), handler.settings)
	if err != nil {
		writeJSONError(w, 500, "failed to load chore reminders")
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
	value, err := json.Marshal(body)
	if err != nil {
		writeJSONError(w, 500, "failed to encode chore reminders")
		return
	}
	if err := handler.settings.Set(r.Context(), "chore_reminders", string(value)); err != nil {
		writeJSONError(w, 500, "failed to save chore reminders")
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
	if authorization := r.Header.Get("Authorization"); strings.HasPrefix(authorization, "Bearer ") {
		hash := repository.HashToken(strings.TrimPrefix(authorization, "Bearer "))
		device.APITokenHash = &hash
	}
	token, err := hex.DecodeString(device.Token)
	if len(device.ID) == 0 || len(device.ID) > 128 || err != nil || len(token) < 16 || len(token) > 256 || (device.Environment != "sandbox" && device.Environment != "production") {
		writeJSONError(w, 400, "valid device identifier, hexadecimal token, and push environment required")
		return
	}
	device.Token = hex.EncodeToString(token)
	if err := handler.devices.Register(r.Context(), device); err != nil {
		if errors.Is(err, models.ErrDeviceOwnership) {
			writeJSONError(w, 409, "device is registered to another user")
			return
		}
		// Do not log database errors here: constraint messages can contain device tokens.
		slog.Error("registering notification device failed")
		writeJSONError(w, 500, "failed to register notification device")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *ChoreReminderHandler) Remove(w http.ResponseWriter, r *http.Request) {
	if err := handler.devices.Remove(r.Context(), chi.URLParam(r, "id"), middleware.GetUser(r.Context()).ID); err != nil {
		writeJSONError(w, 500, "failed to remove notification device")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
