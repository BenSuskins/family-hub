package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
)

type DashboardHandler struct {
	choreRepo      repository.ChoreRepository
	eventRepo      repository.EventRepository
	userRepo       repository.UserRepository
	assignmentRepo repository.ChoreAssignmentRepository
	choreService   *services.ChoreService
	mealPlanRepo   repository.MealPlanRepository
	categoryRepo   repository.CategoryRepository
}

func NewDashboardHandler(
	choreRepo repository.ChoreRepository,
	eventRepo repository.EventRepository,
	userRepo repository.UserRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	choreService *services.ChoreService,
	mealPlanRepo repository.MealPlanRepository,
	categoryRepo repository.CategoryRepository,
) *DashboardHandler {
	return &DashboardHandler{
		choreRepo:      choreRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		assignmentRepo: assignmentRepo,
		choreService:   choreService,
		mealPlanRepo:   mealPlanRepo,
		categoryRepo:   categoryRepo,
	}
}

func (handler *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	now := time.Now()

	// Active chores (pending + overdue)
	activeChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
	})
	if err != nil {
		slog.Error("finding active chores", "error", err)
	}

	overdueCount := 0
	for _, chore := range activeChores {
		if chore.Status == models.ChoreStatusOverdue {
			overdueCount++
		}
	}

	// Upcoming events (next 7 days)
	weekFromNow := now.AddDate(0, 0, 7)
	upcomingEvents, err := handler.eventRepo.FindAll(ctx, repository.EventFilter{
		StartAfter:  &now,
		StartBefore: &weekFromNow,
	})
	if err != nil {
		slog.Error("finding upcoming events", "error", err)
	}

	// Meals this week
	weekStart := now.Truncate(24 * time.Hour)
	weekEnd := weekStart.AddDate(0, 0, 7)
	mealsThisWeek, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: weekStart.Format("2006-01-02"),
		DateTo:   weekEnd.Format("2006-01-02"),
	})
	if err != nil {
		slog.Error("finding meals this week", "error", err)
	}

	// Upcoming chores (next 8 due)
	upcomingChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		OrderBy:  repository.OrderByDueDateAsc,
		Limit:    8,
	})
	if err != nil {
		slog.Error("finding upcoming chores", "error", err)
	}

	// My chores (all pending/overdue)
	myChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses:       []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		AssignedToUser: &user.ID,
		OrderBy:        repository.OrderByDueDateAsc,
	})
	if err != nil {
		slog.Error("finding my chores", "error", err)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	userNameMap := make(map[string]string, len(users))
	userAvatarMap := make(map[string]string, len(users))
	for _, u := range users {
		userNameMap[u.ID] = u.Name
		userAvatarMap[u.ID] = u.AvatarURL
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}
	categoryMap := make(map[string]string, len(categories))
	for _, c := range categories {
		categoryMap[c.ID] = c.Name
	}

	component := pages.Dashboard(pages.DashboardProps{
		User:              user,
		ActiveChoreCount:  len(activeChores),
		OverdueCount:      overdueCount,
		UpcomingEventCount: len(upcomingEvents),
		MealsThisWeek:     len(mealsThisWeek),
		UpcomingChores:    upcomingChores,
		MyChores:          myChores,
		Users:             users,
		UserNameMap:       userNameMap,
		UserAvatarMap:     userAvatarMap,
		CategoryMap:       categoryMap,
		ActiveChoreTab:    "all",
	})
	component.Render(ctx, w)
}

func (handler *DashboardHandler) DashboardChoresTable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "all"
	}

	filter := repository.ChoreFilter{
		AssignedToUser: &user.ID,
		OrderBy:        repository.OrderByDueDateAsc,
	}

	switch tab {
	case "pending":
		pending := models.ChoreStatusPending
		filter.Status = &pending
	case "overdue":
		overdue := models.ChoreStatusOverdue
		filter.Status = &overdue
	default:
		filter.Statuses = []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}
	}

	chores, err := handler.choreRepo.FindAll(ctx, filter)
	if err != nil {
		slog.Error("finding dashboard chores", "error", err)
		http.Error(w, "Error loading chores", http.StatusInternalServerError)
		return
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	userNameMap := make(map[string]string, len(users))
	userAvatarMap := make(map[string]string, len(users))
	for _, u := range users {
		userNameMap[u.ID] = u.Name
		userAvatarMap[u.ID] = u.AvatarURL
	}

	component := pages.DashboardChoresTable(pages.DashboardChoresTableProps{
		Chores:         chores,
		User:           user,
		UserNameMap:    userNameMap,
		UserAvatarMap:  userAvatarMap,
		ActiveChoreTab: tab,
	})
	component.Render(ctx, w)
}
