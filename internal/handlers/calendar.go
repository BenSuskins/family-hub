package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
)

type CalendarHandler struct {
	choreRepo repository.ChoreRepository
	eventRepo repository.EventRepository
}

func NewCalendarHandler(choreRepo repository.ChoreRepository, eventRepo repository.EventRepository) *CalendarHandler {
	return &CalendarHandler{
		choreRepo: choreRepo,
		eventRepo: eventRepo,
	}
}

func (handler *CalendarHandler) Calendar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil {
			month = m
		}
	}

	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	events, err := handler.eventRepo.FindAll(ctx, repository.EventFilter{
		StartAfter:  &startOfMonth,
		StartBefore: &endOfMonth,
	})
	if err != nil {
		slog.Error("finding events for calendar", "error", err)
	}

	chores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		DueAfter:  &startOfMonth,
		DueBefore: &endOfMonth,
	})
	if err != nil {
		slog.Error("finding chores for calendar", "error", err)
	}

	component := pages.Calendar(pages.CalendarProps{
		User:   user,
		Year:   year,
		Month:  month,
		Events: events,
		Chores: chores,
	})
	component.Render(ctx, w)
}
