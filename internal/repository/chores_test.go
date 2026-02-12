package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func stringPtr(s string) *string { return &s }
func timePtr(t time.Time) *time.Time { return &t }

func TestIsOverdue(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	today := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name    string
		chore   models.Chore
		want    bool
	}{
		{
			name:  "due yesterday no DueTime",
			chore: models.Chore{DueDate: timePtr(yesterday)},
			want:  true,
		},
		{
			name:  "due today no DueTime",
			chore: models.Chore{DueDate: timePtr(today)},
			want:  false,
		},
		{
			name:  "due today DueTime in past",
			chore: models.Chore{DueDate: timePtr(today), DueTime: stringPtr("10:00")},
			want:  true,
		},
		{
			name:  "due today DueTime in future",
			chore: models.Chore{DueDate: timePtr(today), DueTime: stringPtr("16:00")},
			want:  false,
		},
		{
			name:  "due tomorrow",
			chore: models.Chore{DueDate: timePtr(tomorrow)},
			want:  false,
		},
		{
			name:  "nil DueDate",
			chore: models.Chore{},
			want:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := repository.IsOverdue(testCase.chore, now)
			if got != testCase.want {
				t.Errorf("IsOverdue() = %v, want %v", got, testCase.want)
			}
		})
	}
}

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

func TestChoreRepository_FindAll_WithStatuses(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	choreRepo.Create(ctx, models.Chore{
		Name: "Pending chore", CreatedByUserID: user.ID,
		Status: models.ChoreStatusPending,
	})
	overdue := models.Chore{
		Name: "Overdue chore", CreatedByUserID: user.ID,
	}
	created, _ := choreRepo.Create(ctx, overdue)
	created.Status = models.ChoreStatusOverdue
	choreRepo.Update(ctx, created)

	completed := models.Chore{
		Name: "Completed chore", CreatedByUserID: user.ID,
	}
	createdCompleted, _ := choreRepo.Create(ctx, completed)
	now := time.Now()
	createdCompleted.Status = models.ChoreStatusCompleted
	createdCompleted.CompletedAt = &now
	choreRepo.Update(ctx, createdCompleted)

	chores, err := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue},
	})
	if err != nil {
		t.Fatalf("finding chores: %v", err)
	}
	if len(chores) != 2 {
		t.Fatalf("expected 2 chores (pending+overdue), got %d", len(chores))
	}
	for _, chore := range chores {
		if chore.Status != models.ChoreStatusPending && chore.Status != models.ChoreStatusOverdue {
			t.Errorf("unexpected status: %s", chore.Status)
		}
	}
}

func TestChoreRepository_FindAll_WithOrderBy(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	earlier := time.Now().Add(-2 * time.Hour)
	later := time.Now().Add(-1 * time.Hour)

	first, _ := choreRepo.Create(ctx, models.Chore{
		Name: "First completed", CreatedByUserID: user.ID,
	})
	first.Status = models.ChoreStatusCompleted
	first.CompletedAt = &earlier
	choreRepo.Update(ctx, first)

	second, _ := choreRepo.Create(ctx, models.Chore{
		Name: "Second completed", CreatedByUserID: user.ID,
	})
	second.Status = models.ChoreStatusCompleted
	second.CompletedAt = &later
	choreRepo.Update(ctx, second)

	chores, err := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusCompleted},
		OrderBy:  repository.OrderByCompletedAtDesc,
	})
	if err != nil {
		t.Fatalf("finding chores: %v", err)
	}
	if len(chores) != 2 {
		t.Fatalf("expected 2 chores, got %d", len(chores))
	}
	if chores[0].Name != "Second completed" {
		t.Errorf("expected 'Second completed' first, got '%s'", chores[0].Name)
	}
	if chores[1].Name != "First completed" {
		t.Errorf("expected 'First completed' second, got '%s'", chores[1].Name)
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
