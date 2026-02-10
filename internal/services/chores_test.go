package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func setupChoreService(t *testing.T) (
	*services.ChoreService,
	*repository.SQLiteChoreRepository,
	*repository.SQLiteChoreAssignmentRepository,
	*repository.SQLiteUserRepository,
) {
	t.Helper()
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo)
	return service, choreRepo, assignmentRepo, userRepo
}

func createUsers(t *testing.T, repo *repository.SQLiteUserRepository, count int) []models.User {
	t.Helper()
	ctx := context.Background()
	var users []models.User
	names := []string{"Alice", "Bob", "Charlie", "Diana"}
	for i := 0; i < count; i++ {
		user, err := repo.Create(ctx, models.User{
			OIDCSubject: "sub-" + names[i],
			Email:       names[i] + "@test.com",
			Name:        names[i],
			Role:        models.RoleMember,
		})
		if err != nil {
			t.Fatalf("creating user %s: %v", names[i], err)
		}
		users = append(users, user)
	}
	return users
}

func TestChoreService_AssignNextUser_RoundRobin(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 3)
	_ = users

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore",
		CreatedByUserID: users[0].ID,
		LastAssignedIndex: -1,
	})

	assigned1, err := service.AssignNextUser(ctx, chore)
	if err != nil {
		t.Fatalf("first assignment: %v", err)
	}
	if assigned1.AssignedToUserID == nil {
		t.Fatal("expected assignment")
	}

	chore2, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Test Chore 2",
		CreatedByUserID: users[0].ID,
		LastAssignedIndex: assigned1.LastAssignedIndex,
	})

	assigned2, err := service.AssignNextUser(ctx, chore2)
	if err != nil {
		t.Fatalf("second assignment: %v", err)
	}

	if *assigned1.AssignedToUserID == *assigned2.AssignedToUserID {
		t.Error("round-robin should assign to different users")
	}
}

func TestChoreService_CompleteChore(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Complete Me",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &users[0].ID,
		Status:           models.ChoreStatusPending,
	})

	if err := service.CompleteChore(ctx, chore.ID, users[0].ID); err != nil {
		t.Fatalf("completing chore: %v", err)
	}

	completed, _ := choreRepo.FindByID(ctx, chore.ID)
	if completed.Status != models.ChoreStatusCompleted {
		t.Errorf("expected completed status, got '%s'", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestChoreService_CompleteChore_WrongUser(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Not Yours",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &users[0].ID,
		Status:           models.ChoreStatusPending,
	})

	err := service.CompleteChore(ctx, chore.ID, users[1].ID)
	if err == nil {
		t.Fatal("expected error completing chore assigned to different user")
	}
	if err != services.ErrNotAssignedToUser {
		t.Errorf("expected ErrNotAssignedToUser, got %v", err)
	}
}

func TestChoreService_CompleteChore_AlreadyComplete(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 1)

	now := time.Now()
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Already Done",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &users[0].ID,
		Status:           models.ChoreStatusCompleted,
		CompletedAt:      &now,
	})

	err := service.CompleteChore(ctx, chore.ID, users[0].ID)
	if err == nil {
		t.Fatal("expected error completing already completed chore")
	}
	if err != services.ErrChoreAlreadyComplete {
		t.Errorf("expected ErrChoreAlreadyComplete, got %v", err)
	}
}

func TestChoreService_CompleteChore_CreatesRecurrence(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	dueDate := time.Now()
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Recurring",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &users[0].ID,
		Status:           models.ChoreStatusPending,
		DueDate:          &dueDate,
		RecurrenceType:   models.RecurrenceDaily,
		RecurrenceValue:  `{"interval": 1}`,
		RecurOnComplete:  true,
	})

	if err := service.CompleteChore(ctx, chore.ID, users[0].ID); err != nil {
		t.Fatalf("completing recurring chore: %v", err)
	}

	allChores, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{})
	pendingCount := 0
	for _, c := range allChores {
		if c.Status == models.ChoreStatusPending {
			pendingCount++
		}
	}
	if pendingCount != 1 {
		t.Errorf("expected 1 new pending chore from recurrence, got %d", pendingCount)
	}
}

func TestChoreService_AssignNextUser_ScopedPool(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 3)

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Scoped Chore",
		CreatedByUserID:   users[0].ID,
		LastAssignedIndex: -1,
	})

	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{users[0].ID, users[2].ID})

	assigned, err := service.AssignNextUser(ctx, chore)
	if err != nil {
		t.Fatalf("scoped assignment: %v", err)
	}
	if assigned.AssignedToUserID == nil {
		t.Fatal("expected assignment")
	}

	assignedID := *assigned.AssignedToUserID
	if assignedID != users[0].ID && assignedID != users[2].ID {
		t.Errorf("assigned to user %s who is not in the eligible pool", assignedID)
	}
	if assignedID == users[1].ID {
		t.Error("should not assign to user not in eligible pool")
	}
}

func TestChoreService_AssignNextUser_PoolChange(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 3)

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Pool Change Chore",
		CreatedByUserID:   users[0].ID,
		LastAssignedIndex: -1,
	})

	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{users[0].ID})

	assigned, err := service.AssignNextUser(ctx, chore)
	if err != nil {
		t.Fatalf("first assignment: %v", err)
	}
	if *assigned.AssignedToUserID != users[0].ID {
		t.Errorf("expected assignment to users[0], got %s", *assigned.AssignedToUserID)
	}

	choreRepo.SetEligibleAssignees(ctx, chore.ID, []string{users[1].ID, users[2].ID})

	chore2, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Pool Change Chore 2",
		CreatedByUserID:   users[0].ID,
		LastAssignedIndex: assigned.LastAssignedIndex,
	})
	choreRepo.SetEligibleAssignees(ctx, chore2.ID, []string{users[1].ID, users[2].ID})

	assigned2, err := service.AssignNextUser(ctx, chore2)
	if err != nil {
		t.Fatalf("second assignment after pool change: %v", err)
	}
	if assigned2.AssignedToUserID == nil {
		t.Fatal("expected assignment")
	}
	assignedID := *assigned2.AssignedToUserID
	if assignedID != users[1].ID && assignedID != users[2].ID {
		t.Errorf("assigned to user %s who is not in the new eligible pool", assignedID)
	}
}

func TestChoreService_UpdateOverdueChores(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 1)

	pastDate := time.Now().AddDate(0, 0, -2)
	choreRepo.Create(ctx, models.Chore{
		Name:            "Overdue",
		CreatedByUserID: users[0].ID,
		DueDate:         &pastDate,
		Status:          models.ChoreStatusPending,
	})

	if err := service.UpdateOverdueChores(ctx); err != nil {
		t.Fatalf("updating overdue: %v", err)
	}

	allChores, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{})
	for _, c := range allChores {
		if c.Name == "Overdue" && c.Status != models.ChoreStatusOverdue {
			t.Errorf("expected overdue status, got '%s'", c.Status)
		}
	}
}
