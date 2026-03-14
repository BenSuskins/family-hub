package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestChoreRepository_SetAndGetEligibleAssignees(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	user2, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-2",
		Email:       "user2@test.com",
		Name:        "User Two",
		Role:        models.RoleMember,
	})

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
	})

	err := choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{user.ID, user2.ID})
	if err != nil {
		t.Fatalf("setting eligible assignees: %v", err)
	}

	assignees, err := choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		t.Fatalf("getting eligible assignees: %v", err)
	}

	if len(assignees) != 2 {
		t.Fatalf("expected 2 eligible assignees, got %d", len(assignees))
	}
}

func TestChoreRepository_SetEligibleAssignees_ReplaceExisting(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	user2, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-2",
		Email:       "user2@test.com",
		Name:        "User Two",
		Role:        models.RoleMember,
	})

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
	})

	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{user.ID, user2.ID})
	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{user2.ID})

	assignees, err := choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		t.Fatalf("getting eligible assignees: %v", err)
	}

	if len(assignees) != 1 {
		t.Fatalf("expected 1 eligible assignee after replacement, got %d", len(assignees))
	}
	if assignees[0] != user2.ID {
		t.Errorf("expected user2 ID, got %s", assignees[0])
	}
}

func TestChoreRepository_GetEligibleAssignees_Empty(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
	})

	assignees, err := choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		t.Fatalf("getting eligible assignees: %v", err)
	}
	if len(assignees) != 0 {
		t.Fatalf("expected 0 eligible assignees, got %d", len(assignees))
	}
}

func TestChoreRepository_SetEligibleAssignees_ClearAll(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: user.ID,
	})

	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{user.ID})
	choreRepo.SetEligibleAssignees(ctx, chore.ID, nil)

	assignees, err := choreRepo.GetEligibleAssignees(ctx, chore.ID)
	if err != nil {
		t.Fatalf("getting eligible assignees: %v", err)
	}
	if len(assignees) != 0 {
		t.Fatalf("expected 0 eligible assignees after clearing, got %d", len(assignees))
	}
}
