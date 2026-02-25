package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/pages"
)

type UserStat struct {
	UserName         string
	UserAvatarURL    string
	CompletedWeek    int
	CompletedMonth   int
	CompletedAllTime int
	AssignedPending  int
}

type DashboardHandler struct {
	choreRepo      repository.ChoreRepository
	icalFetcher    *services.ICalFetcher
	userRepo       repository.UserRepository
	assignmentRepo repository.ChoreAssignmentRepository
	choreService   *services.ChoreService
	mealPlanRepo   repository.MealPlanRepository
	categoryRepo   repository.CategoryRepository
}

func NewDashboardHandler(
	choreRepo repository.ChoreRepository,
	icalFetcher *services.ICalFetcher,
	userRepo repository.UserRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	choreService *services.ChoreService,
	mealPlanRepo repository.MealPlanRepository,
	categoryRepo repository.CategoryRepository,
) *DashboardHandler {
	return &DashboardHandler{
		choreRepo:      choreRepo,
		icalFetcher:    icalFetcher,
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

	choresDueToday, err := handler.choreRepo.FindDueToday(ctx)
	if err != nil {
		slog.Error("finding chores due today", "error", err)
	}

	overdueChores, err := handler.choreRepo.FindOverdueChores(ctx)
	if err != nil {
		slog.Error("finding overdue chores", "error", err)
	}

	seen := make(map[string]bool, len(choresDueToday))
	for _, chore := range choresDueToday {
		seen[chore.ID] = true
	}
	for _, chore := range overdueChores {
		if !seen[chore.ID] {
			choresDueToday = append(choresDueToday, chore)
		}
	}

	activeChores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses:          []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
		OnlyNextPerSeries: true,
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

	weekFromNow := now.AddDate(0, 0, 7)
	upcomingEvents, err := handler.icalFetcher.FetchForRange(ctx, now, weekFromNow)
	if err != nil {
		slog.Error("fetching upcoming ical events", "error", err)
	}

	weekStart := now.Truncate(24 * time.Hour)
	weekEnd := weekStart.AddDate(0, 0, 7)
	mealsThisWeek, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: weekStart.Format("2006-01-02"),
		DateTo:   weekEnd.Format("2006-01-02"),
	})
	if err != nil {
		slog.Error("finding meals this week", "error", err)
	}

	todayMeals, err := handler.mealPlanRepo.FindByDate(ctx, now.Format("2006-01-02"))
	if err != nil {
		slog.Error("finding today's meals", "error", err)
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

	userStats := handler.collectUserStats(ctx, users)

	component := pages.Dashboard(pages.DashboardProps{
		User:               user,
		ActiveChoreCount:   len(activeChores),
		OverdueCount:       overdueCount,
		UpcomingEventCount: len(upcomingEvents),
		MealsThisWeek:      len(mealsThisWeek),
		ChoresDueToday:     choresDueToday,
		UpcomingEvents:     upcomingEvents,
		TodayMeals:         todayMeals,
		UserStats:          convertUserStats(userStats, "week"),
		Users:              users,
		UserNameMap:        userNameMap,
		UserAvatarMap:      userAvatarMap,
	})
	component.Render(ctx, w)
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

	userStats := handler.collectUserStats(ctx, users)

	component := pages.LeaderboardTable(pages.LeaderboardProps{
		UserStats: convertUserStats(userStats, period),
		Period:    period,
	})
	component.Render(ctx, w)
}

func (handler *DashboardHandler) collectUserStats(ctx context.Context, users []models.User) []UserStat {
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	userStats := make([]UserStat, 0, len(users))
	for _, u := range users {
		completedWeek, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, weekAgo)
		completedMonth, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, monthAgo)
		completedAllTime, _ := handler.assignmentRepo.CompletedCountByUser(ctx, u.ID, time.Time{})
		pendingStatus := models.ChoreStatusPending
		pendingChores, _ := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
			Status:            &pendingStatus,
			AssignedToUser:    &u.ID,
			OnlyNextPerSeries: true,
		})

		userStats = append(userStats, UserStat{
			UserName:         u.Name,
			UserAvatarURL:    u.AvatarURL,
			CompletedWeek:    completedWeek,
			CompletedMonth:   completedMonth,
			CompletedAllTime: completedAllTime,
			AssignedPending:  len(pendingChores),
		})
	}
	return userStats
}

func convertUserStats(stats []UserStat, period string) []pages.UserStatProps {
	sort.Slice(stats, func(i, j int) bool {
		switch period {
		case "month":
			return stats[i].CompletedMonth > stats[j].CompletedMonth
		case "alltime":
			return stats[i].CompletedAllTime > stats[j].CompletedAllTime
		default:
			return stats[i].CompletedWeek > stats[j].CompletedWeek
		}
	})

	var result []pages.UserStatProps
	for index, stat := range stats {
		result = append(result, pages.UserStatProps{
			Rank:             index + 1,
			UserName:         stat.UserName,
			UserAvatarURL:    stat.UserAvatarURL,
			CompletedWeek:    stat.CompletedWeek,
			CompletedMonth:   stat.CompletedMonth,
			CompletedAllTime: stat.CompletedAllTime,
			AssignedPending:  stat.AssignedPending,
		})
	}
	return result
}
