package handlers

import (
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
)

type DashboardData struct {
	ChoresDueToday    int
	ChoresOverdue     int
	UpcomingEvents    int
	RecentCompletions int
	UserStats         []UserStat
}

type UserStat struct {
	UserName        string
	UserAvatarURL   string
	CompletedWeek   int
	CompletedMonth  int
	AssignedPending int
}

type DashboardHandler struct {
	choreRepo      repository.ChoreRepository
	eventRepo      repository.EventRepository
	userRepo       repository.UserRepository
	assignmentRepo repository.ChoreAssignmentRepository
	choreService   *services.ChoreService
	mealPlanRepo   repository.MealPlanRepository
}

func NewDashboardHandler(
	choreRepo repository.ChoreRepository,
	eventRepo repository.EventRepository,
	userRepo repository.UserRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	choreService *services.ChoreService,
	mealPlanRepo repository.MealPlanRepository,
) *DashboardHandler {
	return &DashboardHandler{
		choreRepo:      choreRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		assignmentRepo: assignmentRepo,
		choreService:   choreService,
		mealPlanRepo:   mealPlanRepo,
	}
}

func (handler *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	choresDueToday, err := handler.choreRepo.FindDueToday(ctx)
	if err != nil {
		slog.Error("finding chores due today", "error", err)
	}

	overdueChores, err := handler.choreRepo.FindOverdueChores(ctx)
	if err != nil {
		slog.Error("finding overdue chores", "error", err)
	}

	now := time.Now()
	weekFromNow := now.AddDate(0, 0, 7)
	upcomingEvents, err := handler.eventRepo.FindAll(ctx, repository.EventFilter{
		StartAfter:  &now,
		StartBefore: &weekFromNow,
	})
	if err != nil {
		slog.Error("finding upcoming events", "error", err)
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
	}

	userAvatarMap := make(map[string]string, len(users))
	for _, u := range users {
		userAvatarMap[u.ID] = u.AvatarURL
	}

	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	var userStats []UserStat
	for _, u := range users {
		completedWeek, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, weekAgo)
		completedMonth, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, monthAgo)
		assignedPending, _ := handler.choreRepo.CountByStatusAndUser(ctx, "pending", u.ID)

		userStats = append(userStats, UserStat{
			UserName:        u.Name,
			UserAvatarURL:   u.AvatarURL,
			CompletedWeek:   completedWeek,
			CompletedMonth:  completedMonth,
			AssignedPending: assignedPending,
		})
	}

	allChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{})
	if err != nil {
		slog.Error("finding all chores", "error", err)
	}

	todayMeals, err := handler.mealPlanRepo.FindByDate(ctx, now.Format("2006-01-02"))
	if err != nil {
		slog.Error("finding today's meals", "error", err)
	}

	component := pages.Dashboard(pages.DashboardProps{
		User:           user,
		ChoresDueToday: choresDueToday,
		OverdueChores:  overdueChores,
		UpcomingEvents: upcomingEvents,
		AllChores:      allChores,
		Users:          users,
		UserStats:      convertUserStats(userStats, "week"),
		UserAvatarMap:  userAvatarMap,
		TodayMeals:     todayMeals,
	})
	component.Render(ctx, w)
}

func convertUserStats(stats []UserStat, period string) []pages.UserStatProps {
	sort.Slice(stats, func(i, j int) bool {
		if period == "month" {
			return stats[i].CompletedMonth > stats[j].CompletedMonth
		}
		return stats[i].CompletedWeek > stats[j].CompletedWeek
	})

	var result []pages.UserStatProps
	for index, stat := range stats {
		result = append(result, pages.UserStatProps{
			Rank:            index + 1,
			UserName:        stat.UserName,
			UserAvatarURL:   stat.UserAvatarURL,
			CompletedWeek:   stat.CompletedWeek,
			CompletedMonth:  stat.CompletedMonth,
			AssignedPending: stat.AssignedPending,
		})
	}
	return result
}

func (handler *DashboardHandler) Leaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding users", "error", err)
		http.Error(w, "Error loading leaderboard", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	var userStats []UserStat
	for _, u := range users {
		completedWeek, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, weekAgo)
		completedMonth, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, monthAgo)
		assignedPending, _ := handler.choreRepo.CountByStatusAndUser(ctx, "pending", u.ID)

		userStats = append(userStats, UserStat{
			UserName:        u.Name,
			UserAvatarURL:   u.AvatarURL,
			CompletedWeek:   completedWeek,
			CompletedMonth:  completedMonth,
			AssignedPending: assignedPending,
		})
	}

	component := pages.LeaderboardTable(pages.LeaderboardProps{
		UserStats: convertUserStats(userStats, period),
		Period:    period,
	})
	component.Render(ctx, w)
}
