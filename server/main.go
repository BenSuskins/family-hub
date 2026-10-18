package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/database"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/server"
	"github.com/bensuskins/family-hub/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		slog.Error("opening database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		slog.Error("running migrations", "error", err)
		os.Exit(1)
	}

	userRepo := repository.NewUserRepository(db)

	ctx := context.Background()
	authService, err := services.NewAuthService(ctx, cfg, userRepo)
	if err != nil {
		slog.Error("creating auth service", "error", err)
		os.Exit(1)
	}

	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo, services.WithChoreSettings(repository.NewSettingsRepository(db)))

	var reminderSender services.ReminderSender
	var messageSender services.MessageSender
	if cfg.PushConfigured() {
		key, err := os.ReadFile(cfg.APNsKeyPath)
		if err != nil {
			slog.Error("reading APNs signing key failed")
			os.Exit(1)
		}
		sender, err := services.NewAPNsSender(cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsTopic, key, nil)
		if err != nil {
			slog.Error("configuring APNs", "error", err)
			os.Exit(1)
		}
		reminderSender = sender
		messageSender = sender
	}
	reminderService := services.NewChoreReminderService(repository.NewSettingsRepository(db), choreRepo, repository.NewNotificationRepository(db), reminderSender,userRepo)
	go runChoreReminders(reminderService)
	go runOverdueChecker(choreService)

	srv := server.New(db, cfg, authService, messageSender)
	if err := srv.Start(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func runOverdueChecker(choreService *services.ChoreService) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		ctx := context.Background()
		if err := choreService.UpdateOverdueChores(ctx); err != nil {
			slog.Error("updating overdue chores", "error", err)
		}
		<-ticker.C
	}
}

func runChoreReminders(service *services.ChoreReminderService) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		if err := service.Check(ctx, time.Now()); err != nil {
			slog.Error("sending chore reminders", "error", err)
		}
		cancel()
		<-ticker.C
	}
}
