package handlers

import (
	"log/slog"
	"net/http"
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
}

func NewDashboardHandler(
	choreRepo repository.ChoreRepository,
	eventRepo repository.EventRepository,
	userRepo repository.UserRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	choreService *services.ChoreService,
) *DashboardHandler {
	return &DashboardHandler{
		choreRepo:      choreRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		assignmentRepo: assignmentRepo,
		choreService:   choreService,
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

	component := pages.Dashboard(pages.DashboardProps{
		User:           user,
		ChoresDueToday: choresDueToday,
		OverdueChores:  overdueChores,
		UpcomingEvents: upcomingEvents,
		AllChores:      allChores,
		Users:          users,
		UserStats:      convertUserStats(userStats),
		UserAvatarMap:  userAvatarMap,
	})
	component.Render(ctx, w)
}

func convertUserStats(stats []UserStat) []pages.UserStatProps {
	var result []pages.UserStatProps
	for _, stat := range stats {
		result = append(result, pages.UserStatProps{
			UserName:        stat.UserName,
			UserAvatarURL:   stat.UserAvatarURL,
			CompletedWeek:   stat.CompletedWeek,
			CompletedMonth:  stat.CompletedMonth,
			AssignedPending: stat.AssignedPending,
		})
	}
	return result
}
