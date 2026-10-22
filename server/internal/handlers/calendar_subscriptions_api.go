package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type calendarSubscriptionRefresher interface {
	ForceRefreshByID(context.Context, string) error
}

type CalendarSubscriptionsAPIHandler struct {
	repository repository.ICalSubscriptionRepository
	refresher  calendarSubscriptionRefresher
}

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

func NewCalendarSubscriptionsAPIHandler(repository repository.ICalSubscriptionRepository, refresher calendarSubscriptionRefresher) *CalendarSubscriptionsAPIHandler {
	return &CalendarSubscriptionsAPIHandler{repository: repository, refresher: refresher}
}

func (handler *CalendarSubscriptionsAPIHandler) List(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := handler.repository.FindAll(r.Context())
	if err != nil {
		slog.Error("listing calendar subscriptions via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load calendars")
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
	subscription := models.ICalSubscription{ID: uuid.NewString(), Name: body.Name, URL: body.URL, Color: body.Color}
	if err := handler.repository.Create(r.Context(), subscription); err != nil {
		slog.Error("creating calendar subscription via API", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to create calendar")
		return
	}
	created, err := handler.repository.FindByID(r.Context(), subscription.ID)
	if err != nil {
		slog.Error("loading created calendar subscription", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to load created calendar")
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
	subscription, err := handler.repository.FindByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeCalendarLookupError(w, err)
		return
	}
	if err := handler.repository.UpdateColor(r.Context(), subscription.ID, body.Color); err != nil {
		slog.Error("updating calendar color via API", "id", subscription.ID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to update calendar color")
		return
	}
	subscription.Color = body.Color
	writeJSON(w, http.StatusOK, calendarSubscriptionResponse(subscription))
}

func (handler *CalendarSubscriptionsAPIHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if err := handler.refresher.ForceRefreshByID(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeCalendarSubscriptionError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *CalendarSubscriptionsAPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := handler.repository.FindByID(r.Context(), id); err != nil {
		writeCalendarLookupError(w, err)
		return
	}
	if err := handler.repository.Delete(r.Context(), id); err != nil {
		slog.Error("deleting calendar subscription via API", "id", id, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "failed to delete calendar")
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

func writeCalendarSubscriptionError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "calendar not found")
		return
	}
	slog.Error("calendar subscription API request failed", "error", err)
	writeJSONError(w, http.StatusBadGateway, "calendar request failed")
}

func writeCalendarLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "calendar not found")
		return
	}
	slog.Error("looking up calendar subscription via API", "error", err)
	writeJSONError(w, http.StatusInternalServerError, "failed to load calendar")
}
