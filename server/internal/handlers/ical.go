package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type ICalHandler struct {
	choreRepo    repository.ChoreRepository
	userRepo     repository.UserRepository
	tokenRepo    repository.APITokenRepository
	settingsRepo repository.SettingsRepository
	mealPlanRepo repository.MealPlanRepository
}

func NewICalHandler(
	choreRepo repository.ChoreRepository,
	userRepo repository.UserRepository,
	tokenRepo repository.APITokenRepository,
	settingsRepo repository.SettingsRepository,
	mealPlanRepo repository.MealPlanRepository,
) *ICalHandler {
	return &ICalHandler{
		choreRepo:    choreRepo,
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		settingsRepo: settingsRepo,
		mealPlanRepo: mealPlanRepo,
	}
}

func (handler *ICalHandler) Feed(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if !handler.isAuthorizedICalToken(r, token) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	chores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{})
	if err != nil {
		slog.Error("finding chores for ical", "error", err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users for ical", "error", err)
	}

	userMap := make(map[string]string)
	for _, user := range users {
		userMap[user.ID] = user.Name
	}

	hubName := "Family Hub"
	if familyName, err := handler.settingsRepo.Get(ctx, "family_name"); err == nil && familyName != "" {
		hubName = familyName + " Hub"
	}

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=family-hub.ics")

	var builder strings.Builder
	builder.WriteString("BEGIN:VCALENDAR\r\n")
	builder.WriteString("VERSION:2.0\r\n")
	builder.WriteString(fmt.Sprintf("PRODID:-//%s//%s//EN\r\n", hubName, hubName))
	builder.WriteString("CALSCALE:GREGORIAN\r\n")
	builder.WriteString("METHOD:PUBLISH\r\n")
	builder.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", hubName))

	meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{})
	if err != nil {
		slog.Error("finding meals for ical", "error", err)
	}

	for _, meal := range meals {
		builder.WriteString("BEGIN:VEVENT\r\n")
		builder.WriteString(fmt.Sprintf("UID:meal-%s-%s@family-hub\r\n", meal.Date, string(meal.MealType)))
		mealLabel := capitalizeFirst(string(meal.MealType))
		builder.WriteString(fmt.Sprintf("SUMMARY:[%s] %s\r\n", mealLabel, escapeICalText(meal.Name)))
		if meal.Notes != "" {
			builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(meal.Notes)))
		}
		builder.WriteString(fmt.Sprintf("DTSTART;VALUE=DATE:%s\r\n", strings.ReplaceAll(meal.Date, "-", "")))
		if parsedDate, err := time.Parse("2006-01-02", meal.Date); err == nil {
			builder.WriteString(fmt.Sprintf("DTEND;VALUE=DATE:%s\r\n", parsedDate.AddDate(0, 0, 1).Format("20060102")))
		}
		builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", meal.CreatedAt.UTC().Format("20060102T150405Z")))
		builder.WriteString("END:VEVENT\r\n")
	}

	for _, chore := range chores {
		builder.WriteString("BEGIN:VTODO\r\n")
		builder.WriteString(fmt.Sprintf("UID:%s@family-hub\r\n", chore.ID))
		builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(chore.Name)))

		description := chore.Description
		if chore.AssignedToUserID != nil {
			if userName, ok := userMap[*chore.AssignedToUserID]; ok {
				description += fmt.Sprintf("\nAssigned to: %s", userName)
			}
		}
		if description != "" {
			builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(description)))
		}

		if chore.DueDate != nil {
			builder.WriteString(fmt.Sprintf("DUE:%s\r\n", chore.DueDate.UTC().Format("20060102T150405Z")))
		}

		switch chore.Status {
		case "completed":
			builder.WriteString("STATUS:COMPLETED\r\n")
			if chore.CompletedAt != nil {
				builder.WriteString(fmt.Sprintf("COMPLETED:%s\r\n", chore.CompletedAt.UTC().Format("20060102T150405Z")))
			}
		case "pending":
			builder.WriteString("STATUS:NEEDS-ACTION\r\n")
		case "overdue":
			builder.WriteString("STATUS:NEEDS-ACTION\r\n")
			builder.WriteString("PRIORITY:1\r\n")
		}

		builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", chore.CreatedAt.UTC().Format("20060102T150405Z")))
		builder.WriteString("END:VTODO\r\n")
	}

	builder.WriteString("END:VCALENDAR\r\n")

	w.Write([]byte(builder.String()))
}

func (handler *ICalHandler) isAuthorizedICalToken(r *http.Request, token string) bool {
	tokenHash := repository.HashToken(token)
	found, err := handler.tokenRepo.FindByTokenHash(r.Context(), tokenHash)
	if err != nil {
		return false
	}

	return found.Scope == models.TokenScopeICal && (found.ExpiresAt == nil || found.ExpiresAt.After(time.Now()))
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func escapeICalText(text string) string {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}
