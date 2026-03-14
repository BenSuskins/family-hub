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
	choreRepo    repository.ChoreRepository
	icalFetcher  *services.ICalFetcher
	userRepo     repository.UserRepository
	tokenRepo    repository.APITokenRepository
	mealPlanRepo repository.MealPlanRepository
	baseURL      string
}

func NewCalendarHandler(
	choreRepo repository.ChoreRepository,
	icalFetcher *services.ICalFetcher,
	userRepo repository.UserRepository,
	tokenRepo repository.APITokenRepository,
	mealPlanRepo repository.MealPlanRepository,
	baseURL string,
) *CalendarHandler {
	return &CalendarHandler{
		choreRepo:    choreRepo,
		icalFetcher:  icalFetcher,
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		mealPlanRepo: mealPlanRepo,
		baseURL:      baseURL,
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
		// Find the Monday that starts this week
		offset := (int(date.Weekday()) + 6) % 7
		start = time.Date(date.Year(), date.Month(), date.Day()-offset, 0, 0, 0, 0, time.Local)
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

	events, err := handler.icalFetcher.FetchForRange(ctx, start, end)
	if err != nil {
		slog.Error("fetching ical events for calendar", "error", err)
	}

	chores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		DueAfter:  &start,
		DueBefore: &end,
	})
	if err != nil {
		slog.Error("finding chores for calendar", "error", err)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users for calendar", "error", err)
	}

	meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: start.Format("2006-01-02"),
		DateTo:   end.Format("2006-01-02"),
	})
	if err != nil {
		slog.Error("finding meals for calendar", "error", err)
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
		Meals:         meals,
		UserNameMap:   userNameMap,
		UserAvatarMap: userAvatarMap,
	})
	component.Render(ctx, w)
}

func (handler *CalendarHandler) EventDetail(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	title := q.Get("title")
	location := q.Get("location")
	description := q.Get("description")
	allDay := q.Get("all_day") == "true"

	start, _ := time.Parse(time.RFC3339, q.Get("start"))
	event := models.Event{
		Title:       title,
		Location:    location,
		Description: description,
		StartTime:   start,
		AllDay:      allDay,
	}
	if endStr := q.Get("end"); endStr != "" {
		if end, err := time.Parse(time.RFC3339, endStr); err == nil {
			event.EndTime = &end
		}
	}

	pages.EventDetailFragment(event, "").Render(r.Context(), w)
}

func (handler *CalendarHandler) ShareInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	existing, err := handler.tokenRepo.FindByUserIDAndName(ctx, user.ID, "iCal Feed")
	if err != nil {
		slog.Error("finding existing ical tokens", "error", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	for _, token := range existing {
		if err := handler.tokenRepo.Delete(ctx, token.ID); err != nil {
			slog.Error("deleting old ical token", "error", err, "token_id", token.ID)
		}
	}

	rawToken := uuid.New().String()
	tokenHash := repository.HashToken(rawToken)
	newToken := models.APIToken{
		Name:            "iCal Feed",
		TokenHash:       tokenHash,
		CreatedByUserID: user.ID,
	}
	if _, err := handler.tokenRepo.Create(ctx, newToken); err != nil {
		slog.Error("creating ical token", "error", err)
		http.Error(w, "Error creating share link", http.StatusInternalServerError)
		return
	}

	icalURL := handler.baseURL + "/ical?token=" + rawToken
	component := pages.CalendarShareModal(icalURL)
	component.Render(ctx, w)
}
