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

func setupChoreServiceWithSeries(t *testing.T) (
	*services.ChoreService,
	*repository.SQLiteChoreRepository,
	*repository.SQLiteChoreAssignmentRepository,
	*repository.SQLiteUserRepository,
	*repository.SQLiteChoreSeriesRepository,
) {
	t.Helper()
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo)
	return service, choreRepo, assignmentRepo, userRepo, seriesRepo
}

func setupChoreService(t *testing.T) (
	*services.ChoreService,
	*repository.SQLiteChoreRepository,
	*repository.SQLiteChoreAssignmentRepository,
	*repository.SQLiteUserRepository,
) {
	t.Helper()
	service, choreRepo, assignmentRepo, userRepo, _ := setupChoreServiceWithSeries(t)
	return service, choreRepo, assignmentRepo, userRepo
}

// newRecurringChore creates an anchor occurrence and a chore_series definition
// wired together (series.ID = occurrence.ID), as production code does. The
// recurrence rule comes from `series`; occurrence state (due date, assignee,
// status) from `occ`. Required now that the rule and the series_id foreign key
// live in chore_series.
func newRecurringChore(t *testing.T, choreRepo *repository.SQLiteChoreRepository, seriesRepo *repository.SQLiteChoreSeriesRepository, series models.ChoreSeries, occ models.Chore) models.Chore {
	t.Helper()
	ctx := context.Background()
	occ.SeriesID = nil
	created, err := choreRepo.Create(ctx, occ)
	if err != nil {
		t.Fatalf("creating occurrence: %v", err)
	}
	series.ID = created.ID
	if series.CreatedByUserID == "" {
		series.CreatedByUserID = occ.CreatedByUserID
	}
	if _, err := seriesRepo.Create(ctx, series); err != nil {
		t.Fatalf("creating series: %v", err)
	}
	created.SeriesID = &created.ID
	if err := choreRepo.Update(ctx, created); err != nil {
		t.Fatalf("linking series: %v", err)
	}
	return created
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
		Name:              "Test Chore",
		CreatedByUserID:   users[0].ID,
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
		Name:              "Test Chore 2",
		CreatedByUserID:   users[0].ID,
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

func TestChoreService_CompleteChore_AnyUser(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Not Yours",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &users[0].ID,
		Status:           models.ChoreStatusPending,
	})

	if err := service.CompleteChore(ctx, chore.ID, users[1].ID); err != nil {
		t.Fatalf("any user should be able to complete a chore: %v", err)
	}

	completed, _ := choreRepo.FindByID(ctx, chore.ID)
	if completed.Status != models.ChoreStatusCompleted {
		t.Errorf("expected completed status, got '%s'", completed.Status)
	}
	if completed.CompletedByUserID == nil || *completed.CompletedByUserID != users[1].ID {
		t.Error("expected CompletedByUserID to be users[1]")
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
	service, choreRepo, _, userRepo, seriesRepo := setupChoreServiceWithSeries(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	dueDate := time.Now()
	chore := newRecurringChore(t, choreRepo, seriesRepo,
		models.ChoreSeries{RecurrenceType: models.RecurrenceDaily, RecurrenceValue: `{"interval": 1}`, RecurOnComplete: true},
		models.Chore{
			Name:             "Recurring",
			CreatedByUserID:  users[0].ID,
			AssignedToUserID: &users[0].ID,
			Status:           models.ChoreStatusPending,
			DueDate:          &dueDate,
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

func TestChoreService_AssignNextUser_AllOverdueStillRotates(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 2)

	// Every user already has an overdue chore, so the zero-overdue preference
	// can never be satisfied.
	past := time.Now().AddDate(0, 0, -1)
	for _, u := range users {
		uid := u.ID
		choreRepo.Create(ctx, models.Chore{
			Name:             "Old",
			CreatedByUserID:  u.ID,
			AssignedToUserID: &uid,
			DueDate:          &past,
			Status:           models.ChoreStatusOverdue,
		})
	}

	c1, _ := choreRepo.Create(ctx, models.Chore{Name: "A", CreatedByUserID: users[0].ID, LastAssignedIndex: -1})
	a1, err := service.AssignNextUser(ctx, c1)
	if err != nil {
		t.Fatalf("first assign: %v", err)
	}

	c2, _ := choreRepo.Create(ctx, models.Chore{Name: "B", CreatedByUserID: users[0].ID, LastAssignedIndex: a1.LastAssignedIndex})
	a2, err := service.AssignNextUser(ctx, c2)
	if err != nil {
		t.Fatalf("second assign: %v", err)
	}

	if *a1.AssignedToUserID == *a2.AssignedToUserID {
		t.Error("rotation should still advance when every candidate is overdue")
	}
}

func TestChoreService_CompleteChore_UsesSeriesRule(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	seriesID := "s-rule"
	seriesRepo.Create(ctx, models.ChoreSeries{
		ID:              seriesID,
		Name:            "Daily",
		CreatedByUserID: users[0].ID,
		RecurrenceType:  models.RecurrenceDaily,
		RecurrenceValue: `{"interval":1}`,
		RecurOnComplete: true,
	})

	now := time.Now()
	uid := users[0].ID
	// The occurrence's own rule columns are stale (look non-recurring); only the
	// series says it recurs. Completion must honor the series rule.
	occ, _ := choreRepo.Create(ctx, models.Chore{
		Name:             "Daily",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &uid,
		SeriesID:         &seriesID,
		DueDate:          &now,
		Status:           models.ChoreStatusPending,
		RecurrenceType:   models.RecurrenceNone,
	})

	if err := service.CompleteChore(ctx, occ.ID, users[0].ID); err != nil {
		t.Fatalf("complete: %v", err)
	}

	pending, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending},
	})
	if len(pending) == 0 {
		t.Error("series rule should have created a next occurrence despite stale occurrence rule columns")
	}
}

func TestChoreService_Assignment_UsesSeriesPoolAndCursor(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo)
	ctx := context.Background()

	users := createUsers(t, userRepo, 3)

	seriesID := "s1"
	seriesRepo.Create(ctx, models.ChoreSeries{
		ID:              seriesID,
		Name:            "X",
		CreatedByUserID: users[0].ID,
		RecurrenceType:  models.RecurrenceDaily,
	})
	// Series pool restricts to a single user.
	seriesRepo.SetEligibleAssignees(ctx, seriesID, []string{users[1].ID})

	c1, _ := choreRepo.Create(ctx, models.Chore{
		Name: "occ1", CreatedByUserID: users[0].ID, SeriesID: &seriesID, LastAssignedIndex: -1,
	})
	// Give the occurrence a DIFFERENT per-chore pool to prove the series wins.
	choreRepo.SetEligibleAssignees(ctx, c1.ID, []string{users[0].ID, users[2].ID})

	a1, err := service.AssignNextUser(ctx, c1)
	if err != nil {
		t.Fatalf("assign occ1: %v", err)
	}
	if *a1.AssignedToUserID != users[1].ID {
		t.Errorf("series pool must be authoritative; expected users[1], got %s", *a1.AssignedToUserID)
	}

	series, _ := seriesRepo.FindByID(ctx, seriesID)
	if series.RotationCursorUserID == nil || *series.RotationCursorUserID != users[1].ID {
		t.Errorf("rotation cursor should advance to users[1], got %v", series.RotationCursorUserID)
	}

	// Resolve the first task before creating another outstanding task.
	a1.Status = models.ChoreStatusCompleted
	if err := choreRepo.Update(ctx, a1); err != nil {
		t.Fatal(err)
	}
	// Widen the pool; the next occurrence continues from the cursor.
	seriesRepo.SetEligibleAssignees(ctx, seriesID, []string{users[0].ID, users[1].ID, users[2].ID})
	c2, _ := choreRepo.Create(ctx, models.Chore{
		Name: "occ2", CreatedByUserID: users[0].ID, SeriesID: &seriesID, LastAssignedIndex: -1,
	})
	a2, err := service.AssignNextUser(ctx, c2)
	if err != nil {
		t.Fatalf("assign occ2: %v", err)
	}
	if *a2.AssignedToUserID == users[1].ID {
		t.Errorf("rotation should advance past the cursor user, but stayed on users[1]")
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

func TestChoreService_UpdateOverdueChores_DoesNotRewriteAlreadyOverdue(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()

	users := createUsers(t, userRepo, 1)

	pastDate := time.Now().AddDate(0, 0, -2)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Overdue",
		CreatedByUserID: users[0].ID,
		DueDate:         &pastDate,
		Status:          models.ChoreStatusPending,
	})

	// First pass flips pending -> overdue and stamps updated_at.
	if err := service.UpdateOverdueChores(ctx); err != nil {
		t.Fatalf("first pass: %v", err)
	}
	afterFirst, _ := choreRepo.FindByID(ctx, chore.ID)
	if afterFirst.Status != models.ChoreStatusOverdue {
		t.Fatalf("expected overdue after first pass, got %s", afterFirst.Status)
	}

	// Second pass must be a no-op: the row is already overdue, so updated_at
	// must not move (no write amplification).
	if err := service.UpdateOverdueChores(ctx); err != nil {
		t.Fatalf("second pass: %v", err)
	}
	afterSecond, _ := choreRepo.FindByID(ctx, chore.ID)
	if !afterSecond.UpdatedAt.Equal(afterFirst.UpdatedAt) {
		t.Errorf("already-overdue chore was rewritten: updated_at moved from %v to %v",
			afterFirst.UpdatedAt, afterSecond.UpdatedAt)
	}
}
