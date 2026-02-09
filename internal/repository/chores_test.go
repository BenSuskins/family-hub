package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func createTestUser(t *testing.T, repo *repository.SQLiteUserRepository) models.User {
	t.Helper()
	user, err := repo.Create(context.Background(), models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return user
}

func TestChoreRepository_CreateAndFindByID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	dueDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	chore := models.Chore{
		Name:            "Clean kitchen",
		Description:     "Wipe counters and mop",
		CreatedByUserID: user.ID,
		DueDate:         &dueDate,
		RecurrenceType:  models.RecurrenceWeekly,
		RecurrenceValue: `{"interval": 1}`,
	}

	created, err := choreRepo.Create(ctx, chore)
	if err != nil {
		t.Fatalf("creating chore: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if created.Status != models.ChoreStatusPending {
		t.Errorf("expected pending status, got '%s'", created.Status)
	}

	found, err := choreRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding chore: %v", err)
	}
	if found.Name != "Clean kitchen" {
		t.Errorf("expected name 'Clean kitchen', got '%s'", found.Name)
	}
	if found.RecurrenceType != models.RecurrenceWeekly {
		t.Errorf("expected weekly recurrence, got '%s'", found.RecurrenceType)
	}
}

func TestChoreRepository_FindAll_WithFilters(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	choreRepo.Create(ctx, models.Chore{
		Name: "Pending chore", CreatedByUserID: user.ID,
		Status: models.ChoreStatusPending,
	})
	completedChore := models.Chore{
		Name: "Completed chore", CreatedByUserID: user.ID,
	}
	created, _ := choreRepo.Create(ctx, completedChore)
	now := time.Now()
	created.Status = models.ChoreStatusCompleted
	created.CompletedAt = &now
	choreRepo.Update(ctx, created)

	pending := models.ChoreStatusPending
	chores, err := choreRepo.FindAll(ctx, repository.ChoreFilter{Status: &pending})
	if err != nil {
		t.Fatalf("finding chores: %v", err)
	}
	if len(chores) != 1 {
		t.Fatalf("expected 1 pending chore, got %d", len(chores))
	}
	if chores[0].Name != "Pending chore" {
		t.Errorf("expected 'Pending chore', got '%s'", chores[0].Name)
	}
}

func TestChoreRepository_Update(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := choreRepo.Create(ctx, models.Chore{
		Name: "Original", CreatedByUserID: user.ID,
	})

	created.Name = "Updated"
	if err := choreRepo.Update(ctx, created); err != nil {
		t.Fatalf("updating chore: %v", err)
	}

	found, _ := choreRepo.FindByID(ctx, created.ID)
	if found.Name != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", found.Name)
	}
}

func TestChoreRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := choreRepo.Create(ctx, models.Chore{
		Name: "To Delete", CreatedByUserID: user.ID,
	})

	if err := choreRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("deleting chore: %v", err)
	}

	_, err := choreRepo.FindByID(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error finding deleted chore")
	}
}

func TestChoreRepository_CountByStatusAndUser(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	choreRepo.Create(ctx, models.Chore{
		Name: "Chore 1", CreatedByUserID: user.ID,
		AssignedToUserID: &user.ID, Status: models.ChoreStatusPending,
	})
	choreRepo.Create(ctx, models.Chore{
		Name: "Chore 2", CreatedByUserID: user.ID,
		AssignedToUserID: &user.ID, Status: models.ChoreStatusPending,
	})

	count, err := choreRepo.CountByStatusAndUser(ctx, models.ChoreStatusPending, user.ID)
	if err != nil {
		t.Fatalf("counting chores: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 pending chores, got %d", count)
	}
}
