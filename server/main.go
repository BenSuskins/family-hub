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

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/database"
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
	srv, composedWorkers := server.Compose(ctx, db, cfg, server.Dependencies{Now: now, ReminderSender: reminderSender, MessageSender: messageSender})
	workers := make([]worker, len(composedWorkers))
	for index, job := range composedWorkers {
		workers[index] = worker{name: job.Name, interval: job.Interval, timeout: job.Timeout, run: job.Run}
	}

	return srv, workers, nil
}
