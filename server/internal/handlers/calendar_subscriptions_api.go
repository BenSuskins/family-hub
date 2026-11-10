package handlers

import (
	"errors"
	"net/http"
	"time"

	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
)

type CalendarSubscriptionsAPIHandler struct{ service *calendarapp.Subscriptions }

type CalendarSubscriptionResponse struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Color         string     `json:"color"`
	LastFetchedAt *time.Time `json:"lastFetchedAt"`
	CreatedAt     time.Time  `json:"createdAt"`
}

type calendarSubscriptionRequest struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Color string `json:"color"`
}

func NewCalendarSubscriptionsAPIHandler(service *calendarapp.Subscriptions) *CalendarSubscriptionsAPIHandler {
	return &CalendarSubscriptionsAPIHandler{service: service}
}

func (handler *CalendarSubscriptionsAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := handler.service.List(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load calendars", err)
		return
	}
	responses := make([]CalendarSubscriptionResponse, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		responses = append(responses, calendarSubscriptionResponse(subscription))
	}
	writeJSON(w, http.StatusOK, responses)
}

func (handler *CalendarSubscriptionsAPIHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body calendarSubscriptionRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" || body.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if err := services.ValidateExternalURL(body.URL); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid calendar URL")
		return
	}
	if body.Color == "" {
		body.Color = "indigo"
	}
	if !isValidSubscriptionColor(body.Color) {
		writeJSONError(w, http.StatusBadRequest, "invalid calendar color")
		return
	}
	created, err := handler.service.Create(r.Context(), body.Name, body.URL, body.Color)
	if err != nil {
		if errors.Is(err, calendarapp.ErrReadCreated) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load created calendar", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to create calendar", err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, calendarSubscriptionResponse(created))
}

func (handler *CalendarSubscriptionsAPIHandler) UpdateColor(w http.ResponseWriter, r *http.Request) {
	var body calendarSubscriptionRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if !isValidSubscriptionColor(body.Color) {
		writeJSONError(w, http.StatusBadRequest, "invalid calendar color")
		return
	}
	subscription, err := handler.service.UpdateColor(r.Context(), chi.URLParam(r, "id"), body.Color)
	if err != nil {
		if errors.Is(err, calendarapp.ErrWrite) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to update calendar color", err)
		} else {
			writeCalendarLookupError(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, calendarSubscriptionResponse(subscription))
}

func (handler *CalendarSubscriptionsAPIHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if err := handler.service.Refresh(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeCalendarSubscriptionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *CalendarSubscriptionsAPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := handler.service.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		if errors.Is(err, calendarapp.ErrWrite) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to delete calendar", err)
		} else {
			writeCalendarLookupError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func calendarSubscriptionResponse(subscription models.ICalSubscription) CalendarSubscriptionResponse {
	return CalendarSubscriptionResponse{
		ID: subscription.ID, Name: subscription.Name, URL: subscription.URL,
		Color: subscription.Color, LastFetchedAt: subscription.LastFetchedAt, CreatedAt: subscription.CreatedAt,
	}
}

func isValidSubscriptionColor(color string) bool {
	switch color {
	case "indigo", "violet", "rose", "red", "orange", "amber", "emerald", "teal", "sky", "blue":
		return true
	default:
		return false
	}
}

func writeCalendarSubscriptionError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, calendarapp.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "calendar not found")
		return
	}
	observability.RecordDependencyFailure(r, "calendar_feed")
	writeAPIError(w, r, http.StatusBadGateway, "calendar request failed", err)
}

func writeCalendarLookupError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, calendarapp.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "calendar not found")
		return
	}
	writeAPIError(w, r, http.StatusInternalServerError, "failed to load calendar", err)
}
