package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/google/uuid"
)

type CalendarHandler struct {
	choreRepo repository.ChoreRepository
	eventRepo repository.EventRepository
	userRepo  repository.UserRepository
	tokenRepo repository.APITokenRepository
	baseURL   string
}

func NewCalendarHandler(
	choreRepo repository.ChoreRepository,
	eventRepo repository.EventRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.APITokenRepository,
	baseURL string,
) *CalendarHandler {
	return &CalendarHandler{
		choreRepo: choreRepo,
		eventRepo: eventRepo,
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		baseURL:   baseURL,
	}
}

func (handler *CalendarHandler) Calendar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	now := time.Now()
	view := r.URL.Query().Get("view")
	if view == "" {
		view = "month"
	}

	var start, end time.Time
	var year, month int
	var date time.Time

	switch view {
	case "year":
		year = now.Year()
		if yearStr := r.URL.Query().Get("year"); yearStr != "" {
			if y, err := strconv.Atoi(yearStr); err == nil {
				year = y
			}
		}
		month = int(now.Month())
		start = time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		end = time.Date(year+1, 1, 1, 0, 0, 0, 0, time.Local)
		date = start

	case "week":
		date = now
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				date = d
			}
		}
		// Find the Sunday that starts this week
		weekday := int(date.Weekday())
		start = time.Date(date.Year(), date.Month(), date.Day()-weekday, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 7)
		year = date.Year()
		month = int(date.Month())

	case "day":
		date = now
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				date = d
			}
		}
		start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 1)
		year = date.Year()
		month = int(date.Month())

	default: // "month"
		view = "month"
		year = now.Year()
		month = int(now.Month())
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
		start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 1, 0)
		date = start
	}

	events, err := handler.eventRepo.FindAll(ctx, repository.EventFilter{
		StartAfter:  &start,
		StartBefore: &end,
	})
	if err != nil {
		slog.Error("finding events for calendar", "error", err)
	}

	chores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		DueAfter:  &start,
		DueBefore: &end,
	})
	if err != nil {
		slog.Error("finding chores for calendar", "error", err)
	}

	recurringChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		RecurrenceTypes: []models.RecurrenceType{
			models.RecurrenceDaily,
			models.RecurrenceWeekly,
			models.RecurrenceMonthly,
			models.RecurrenceCustom,
			models.RecurrenceCalendar,
		},
	})
	if err != nil {
		slog.Error("finding recurring chores for calendar", "error", err)
	}

	for _, chore := range recurringChores {
		expanded, err := services.ExpandChoreOccurrences(chore, start, end)
		if err != nil {
			slog.Error("expanding chore occurrences", "error", err, "chore_id", chore.ID)
			continue
		}
		chores = append(chores, expanded...)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users for calendar", "error", err)
	}

	userNameMap := make(map[string]string, len(users))
	userAvatarMap := make(map[string]string, len(users))
	for _, u := range users {
		userNameMap[u.ID] = u.Name
		userAvatarMap[u.ID] = u.AvatarURL
	}

	component := pages.Calendar(pages.CalendarProps{
		User:          user,
		Year:          year,
		Month:         month,
		View:          view,
		Date:          date,
		Events:        events,
		Chores:        chores,
		UserNameMap:   userNameMap,
		UserAvatarMap: userAvatarMap,
	})
	component.Render(ctx, w)
}

func (handler *CalendarHandler) ShareInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	tokens, err := handler.tokenRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding tokens", "error", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var icalToken *models.APIToken
	for _, token := range tokens {
		if token.Name == "iCal Feed" && token.CreatedByUserID == user.ID {
			icalToken = &token
			break
		}
	}

	if icalToken == nil {
		rawToken := uuid.New().String()
		tokenHash := repository.HashToken(rawToken)
		newToken := models.APIToken{
			Name:            "iCal Feed",
			TokenHash:       tokenHash,
			CreatedByUserID: user.ID,
		}
		created, err := handler.tokenRepo.Create(ctx, newToken)
		if err != nil {
			slog.Error("creating ical token", "error", err)
			http.Error(w, "Error creating share link", http.StatusInternalServerError)
			return
		}
		_ = created

		icalURL := handler.baseURL + "/ical?token=" + rawToken
		component := pages.CalendarShareModal(icalURL)
		component.Render(ctx, w)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<div class="p-4 text-sm text-gray-600">A share link was already created. Check your Admin panel for existing API tokens, or use the iCal URL you saved previously.</div>`))
}
