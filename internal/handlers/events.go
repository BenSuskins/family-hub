package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

type EventHandler struct {
	eventRepo    repository.EventRepository
	categoryRepo repository.CategoryRepository
}

func NewEventHandler(eventRepo repository.EventRepository, categoryRepo repository.CategoryRepository) *EventHandler {
	return &EventHandler{eventRepo: eventRepo, categoryRepo: categoryRepo}
}

func (handler *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	filter := repository.EventFilter{}
	if afterStr := r.URL.Query().Get("after"); afterStr != "" {
		if after, err := time.Parse("2006-01-02", afterStr); err == nil {
			filter.StartAfter = &after
		}
	}
	if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
		if before, err := time.Parse("2006-01-02", beforeStr); err == nil {
			filter.StartBefore = &before
		}
	}

	events, err := handler.eventRepo.FindAll(ctx, filter)
	if err != nil {
		slog.Error("finding events", "error", err)
		http.Error(w, "Error loading events", http.StatusInternalServerError)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	categoryMap := make(map[string]string, len(categories))
	for _, c := range categories {
		categoryMap[c.ID] = c.Name
	}

	component := pages.EventList(pages.EventListProps{
		User:        user,
		Events:      events,
		Categories:  categories,
		CategoryMap: categoryMap,
	})
	component.Render(ctx, w)
}

func (handler *EventHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.EventForm(pages.EventFormProps{
		User:       user,
		Categories: categories,
		IsEdit:     false,
	})
	component.Render(ctx, w)
}

func (handler *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	event := models.Event{
		Title:           r.FormValue("title"),
		Description:     r.FormValue("description"),
		Location:        r.FormValue("location"),
		AllDay:          r.FormValue("all_day") == "on",
		CreatedByUserID: user.ID,
	}

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		event.CategoryID = &categoryID
	}

	startDate := r.FormValue("start_date")
	if event.AllDay {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			event.StartTime = parsed
		}
	} else {
		startTime := r.FormValue("start_time")
		event.StartTime = parseDateTime(startDate, startTime)
		if endTimeStr := r.FormValue("end_time"); endTimeStr != "" {
			endTime := parseDateTime(startDate, endTimeStr)
			event.EndTime = &endTime
		}
	}

	if _, err := handler.eventRepo.Create(ctx, event); err != nil {
		slog.Error("creating event", "error", err)
		http.Error(w, "Error creating event", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/events", http.StatusFound)
}

func (handler *EventHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	eventID := chi.URLParam(r, "id")

	event, err := handler.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.EventForm(pages.EventFormProps{
		User:       user,
		Event:      &event,
		Categories: categories,
		IsEdit:     true,
	})
	component.Render(ctx, w)
}

func (handler *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	event, err := handler.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	event.Title = r.FormValue("title")
	event.Description = r.FormValue("description")
	event.Location = r.FormValue("location")
	event.AllDay = r.FormValue("all_day") == "on"

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		event.CategoryID = &categoryID
	} else {
		event.CategoryID = nil
	}

	startDate := r.FormValue("start_date")
	if event.AllDay {
		if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
			event.StartTime = parsed
		}
		event.EndTime = nil
	} else {
		startTime := r.FormValue("start_time")
		event.StartTime = parseDateTime(startDate, startTime)
		if endTimeStr := r.FormValue("end_time"); endTimeStr != "" {
			endTime := parseDateTime(startDate, endTimeStr)
			event.EndTime = &endTime
		} else {
			event.EndTime = nil
		}
	}

	if err := handler.eventRepo.Update(ctx, event); err != nil {
		slog.Error("updating event", "error", err)
		http.Error(w, "Error updating event", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/events", http.StatusFound)
}

func (handler *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "id")

	if err := handler.eventRepo.Delete(ctx, eventID); err != nil {
		slog.Error("deleting event", "error", err)
		http.Error(w, "Error deleting event", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/events", http.StatusFound)
}

func (handler *EventHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	eventID := chi.URLParam(r, "id")

	event, err := handler.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var categoryName string
	if event.CategoryID != nil {
		category, err := handler.categoryRepo.FindByID(ctx, *event.CategoryID)
		if err == nil {
			categoryName = category.Name
		}
	}

	component := pages.EventDetailFragment(event, categoryName)
	component.Render(ctx, w)
}

func parseDateTime(dateStr string, timeStr string) time.Time {
	if dateStr == "" {
		return time.Time{}
	}
	if timeStr != "" {
		combined := dateStr + "T" + timeStr
		if parsed, err := time.Parse("2006-01-02T15:04", combined); err == nil {
			return parsed
		}
	}
	if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
		return parsed
	}
	return time.Time{}
}
