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
	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)

	go runOverdueChecker(choreService)

	srv := server.New(db, cfg, authService)
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
