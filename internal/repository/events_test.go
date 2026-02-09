package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestEventRepository_CreateAndFindByID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	event := models.Event{
		Title:           "Family Dinner",
		Description:     "Weekly family dinner",
		Location:        "Home",
		StartTime:       time.Date(2025, 6, 15, 18, 0, 0, 0, time.UTC),
		AllDay:          false,
		CreatedByUserID: user.ID,
	}

	created, err := eventRepo.Create(ctx, event)
	if err != nil {
		t.Fatalf("creating event: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	found, err := eventRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding event: %v", err)
	}
	if found.Title != "Family Dinner" {
		t.Errorf("expected title 'Family Dinner', got '%s'", found.Title)
	}
	if found.Location != "Home" {
		t.Errorf("expected location 'Home', got '%s'", found.Location)
	}
}

func TestEventRepository_FindAll_WithDateFilter(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	eventRepo.Create(ctx, models.Event{
		Title: "Past Event", StartTime: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
		CreatedByUserID: user.ID,
	})
	eventRepo.Create(ctx, models.Event{
		Title: "Future Event", StartTime: time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC),
		CreatedByUserID: user.ID,
	})

	after := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	events, err := eventRepo.FindAll(ctx, repository.EventFilter{StartAfter: &after})
	if err != nil {
		t.Fatalf("finding events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Title != "Future Event" {
		t.Errorf("expected 'Future Event', got '%s'", events[0].Title)
	}
}

func TestEventRepository_Update(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := eventRepo.Create(ctx, models.Event{
		Title: "Original", StartTime: time.Now(), CreatedByUserID: user.ID,
	})

	created.Title = "Updated"
	if err := eventRepo.Update(ctx, created); err != nil {
		t.Fatalf("updating event: %v", err)
	}

	found, _ := eventRepo.FindByID(ctx, created.ID)
	if found.Title != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", found.Title)
	}
}

func TestEventRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	eventRepo := repository.NewEventRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := eventRepo.Create(ctx, models.Event{
		Title: "To Delete", StartTime: time.Now(), CreatedByUserID: user.ID,
	})

	if err := eventRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("deleting event: %v", err)
	}

	_, err := eventRepo.FindByID(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error finding deleted event")
	}
}
