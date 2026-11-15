package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	notifications "github.com/bensuskins/family-hub/internal/application/notifications"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
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

	rootContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	srv, workers, err := compose(rootContext, db, cfg, time.Now)
	if err != nil {
		slog.Error("configuring server", "error", err)
		os.Exit(1)
	}
	if err := serve(rootContext, srv, workers, 30*time.Second); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func compose(ctx context.Context, db *sql.DB, cfg config.Config, now func() time.Time) (*server.Server, []worker, error) {
	userRepo := repository.NewUserRepository(db)

	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	choreService := choreapp.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo, choreapp.WithChoreTimezone(services.ChoreTimezone(repository.NewSettingsRepository(db))))

	var reminderSender notifications.ReminderSender
	var messageSender notifications.MessageSender
	if cfg.PushConfigured() {
		key, err := os.ReadFile(cfg.APNsKeyPath)
		if err != nil {
			return nil, nil, errors.New("reading APNs signing key failed")
		}
		sender, err := services.NewAPNsSender(cfg.APNsKeyID, cfg.APNsTeamID, cfg.APNsTopic, key, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("configuring APNs: %w", err)
		}
		reminderSender = sender
		messageSender = sender
	}
	reminderService := notifications.NewChoreReminderService(services.NewReminderConfig(repository.NewSettingsRepository(db)), choreRepo, repository.NewNotificationRepository(db), reminderSender, userRepo)
	binReminderService := notifications.NewBinReminderService(services.NewReminderConfig(repository.NewSettingsRepository(db)), repository.NewNotificationRepository(db), messageSender)
	srv := server.NewWithContext(ctx, db, cfg, messageSender)
	workers := []worker{
		{name: "sending chore reminders", interval: 30 * time.Second, timeout: 25 * time.Second, run: func(ctx context.Context) error { return reminderService.Check(ctx, now()) }},
		{name: "sending bin reminders", interval: 30 * time.Second, timeout: 25 * time.Second, run: func(ctx context.Context) error { return binReminderService.Check(ctx, now()) }},
		{name: "updating overdue chores", interval: 5 * time.Minute, run: choreService.UpdateOverdueChores},
	}
	return srv, workers, nil
}
