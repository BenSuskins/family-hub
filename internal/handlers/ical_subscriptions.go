package handlers

import (
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ICalSubscriptionsHandler struct {
	subRepo repository.ICalSubscriptionRepository
	fetcher *services.ICalFetcher
}

func NewICalSubscriptionsHandler(subRepo repository.ICalSubscriptionRepository, fetcher *services.ICalFetcher) *ICalSubscriptionsHandler {
	return &ICalSubscriptionsHandler{subRepo: subRepo, fetcher: fetcher}
}

func (h *ICalSubscriptionsHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	subs, err := h.subRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding ical subscriptions", "error", err)
	}

	pages.Calendars(pages.CalendarsProps{
		User:          user,
		Subscriptions: subs,
	}).Render(ctx, w)
}

func (h *ICalSubscriptionsHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	url := r.FormValue("url")
	if name == "" || url == "" {
		http.Error(w, "Name and URL are required", http.StatusBadRequest)
		return
	}

	sub := models.ICalSubscription{
		ID:   uuid.New().String(),
		Name: name,
		URL:  url,
	}
	if err := h.subRepo.Create(ctx, sub); err != nil {
		slog.Error("creating ical subscription", "error", err)
		http.Error(w, "Error creating subscription", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/calendars", http.StatusSeeOther)
}

func (h *ICalSubscriptionsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.subRepo.Delete(ctx, id); err != nil {
		slog.Error("deleting ical subscription", "error", err)
		http.Error(w, "Error deleting subscription", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/calendars", http.StatusSeeOther)
}

func (h *ICalSubscriptionsHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := h.fetcher.ForceRefreshByID(ctx, id); err != nil {
		slog.Warn("refreshing ical subscription", "id", id, "error", err)
	}

	http.Redirect(w, r, "/calendars", http.StatusSeeOther)
}
