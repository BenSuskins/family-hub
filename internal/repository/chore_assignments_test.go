package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestChoreAssignmentRepository_DeleteCompleted(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	chore, err := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
		Status:          models.ChoreStatusPending,
	})
	if err != nil {
		t.Fatalf("creating chore: %v", err)
	}

	_, err = assignmentRepo.Create(ctx, models.ChoreAssignment{
		ChoreID:    chore.ID,
		UserID:     user.ID,
		AssignedAt: time.Now(),
		Status:     models.AssignmentStatusCompleted,
	})
	if err != nil {
		t.Fatalf("creating completed assignment: %v", err)
	}

	assigned, err := assignmentRepo.Create(ctx, models.ChoreAssignment{
		ChoreID:    chore.ID,
		UserID:     user.ID,
		AssignedAt: time.Now(),
		Status:     models.AssignmentStatusAssigned,
	})
	if err != nil {
		t.Fatalf("creating assigned assignment: %v", err)
	}

	if err := assignmentRepo.DeleteCompleted(ctx); err != nil {
		t.Fatalf("deleting completed assignments: %v", err)
	}

	remaining, err := assignmentRepo.FindByChoreID(ctx, chore.ID)
	if err != nil {
		t.Fatalf("finding assignments: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("expected 1 remaining assignment, got %d", len(remaining))
	}
	if len(remaining) > 0 && remaining[0].ID != assigned.ID {
		t.Errorf("expected assigned assignment to remain, got status %s", remaining[0].Status)
	}
}
