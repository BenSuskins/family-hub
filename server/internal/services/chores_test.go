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
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo, repository.NewChoreSeriesRepository(db))
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

func TestChoreService_SeedFutureOccurrences(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 2)

	now := time.Now()
	base := now.AddDate(0, 0, 1) // due tomorrow
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Weekly Cleanup",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceWeekly,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		LastAssignedIndex: -1,
	})
	// Set series_id = chore.ID
	seriesID := chore.ID
	chore.SeriesID = &seriesID
	choreRepo.Update(ctx, chore)

	until := now.AddDate(0, 0, 28) // seed 4 weeks ahead
	if err := service.SeedFutureOccurrences(ctx, chore, until); err != nil {
		t.Fatalf("SeedFutureOccurrences: %v", err)
	}

	all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending},
	})
	// base chore + 3 seeded instances (4 weeks - 1 already exists = 3 new ones)
	if len(all) < 3 {
		t.Errorf("want at least 3 pending chores, got %d", len(all))
	}
	for _, c := range all {
		if c.ID == chore.ID {
			continue // skip base
		}
		if c.Status != models.ChoreStatusPending {
			t.Errorf("seeded chore %s has status %s, want pending", c.ID, c.Status)
		}
		if c.SeriesID == nil || *c.SeriesID != seriesID {
			t.Errorf("seeded chore %s has wrong series_id", c.ID)
		}
		if c.AssignedToUserID == nil {
			t.Errorf("seeded chore %s has no assignee", c.ID)
		}
	}
}

func TestChoreService_SeedFutureOccurrences_RotatesAcrossUsers(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 3)
	_ = users

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceDaily,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		LastAssignedIndex: -1,
	})
	seriesID := chore.ID
	chore.SeriesID = &seriesID
	choreRepo.Update(ctx, chore)
	chore, _ = service.AssignNextUser(ctx, chore)

	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 0, 7)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	pending, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending},
		OrderBy:  repository.OrderByDueDateAsc,
	})

	seen := map[string]bool{}
	prev := ""
	for _, c := range pending {
		if c.AssignedToUserID == nil {
			t.Fatalf("occurrence %s left unassigned", c.ID)
		}
		seen[*c.AssignedToUserID] = true
		if prev != "" && prev == *c.AssignedToUserID {
			t.Errorf("consecutive occurrences both assigned to %s", prev)
		}
		prev = *c.AssignedToUserID
	}
	if len(seen) != 3 {
		t.Errorf("expected rotation to cover all 3 users, got %d distinct", len(seen))
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

func TestChoreService_TopUpAllSeries_RefillsExhaustedSeries(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	createUsers(t, userRepo, 2)

	// A recurring series whose only row is already in the past: the future
	// window is exhausted.
	past := time.Now().AddDate(0, 0, -3)
	seriesID := "exhausted-series"
	user := func() string { u, _ := userRepo.FindAll(ctx); return u[0].ID }()
	choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   user,
		RecurrenceType:    models.RecurrenceDaily,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &past,
		Status:            models.ChoreStatusPending,
		SeriesID:          &seriesID,
		LastAssignedIndex: -1,
	})

	if err := service.TopUpAllSeries(ctx, time.Now().AddDate(0, 0, 10)); err != nil {
		t.Fatalf("TopUpAllSeries: %v", err)
	}

	now := time.Now()
	pending, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending},
	})
	future := 0
	for _, c := range pending {
		if c.DueDate != nil && c.DueDate.After(now) {
			future++
		}
	}
	if future == 0 {
		t.Error("top-up should refill future occurrences for an exhausted series")
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

	// Widen the pool; the next occurrence must continue rotation from the cursor.
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

func TestChoreService_BackfillSeries(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	choreRepo := repository.NewChoreRepository(db)
	assignmentRepo := repository.NewChoreAssignmentRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	service := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo)
	ctx := context.Background()

	users := createUsers(t, userRepo, 2)

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	seriesID := "legacy-series"
	uid := users[0].ID
	choreRepo.Create(ctx, models.Chore{
		Name:             "Hoover",
		Description:      "Whole house",
		CreatedByUserID:  users[0].ID,
		AssignedToUserID: &uid,
		RecurrenceType:   models.RecurrenceWeekly,
		RecurrenceValue:  `{"interval":1}`,
		DueDate:          &base,
		Status:           models.ChoreStatusPending,
		SeriesID:         &seriesID,
	})

	if err := service.BackfillSeries(ctx); err != nil {
		t.Fatalf("BackfillSeries: %v", err)
	}

	series, err := seriesRepo.FindByID(ctx, seriesID)
	if err != nil {
		t.Fatalf("finding backfilled series: %v", err)
	}
	if series == nil {
		t.Fatal("expected a backfilled series row")
	}
	if series.Name != "Hoover" || series.RecurrenceType != models.RecurrenceWeekly {
		t.Errorf("backfilled series mismatch: %+v", series)
	}
	if series.RotationCursorUserID == nil || *series.RotationCursorUserID != uid {
		t.Errorf("expected rotation cursor %s, got %v", uid, series.RotationCursorUserID)
	}

	// Idempotent: a second pass must not error or duplicate.
	if err := service.BackfillSeries(ctx); err != nil {
		t.Fatalf("second BackfillSeries: %v", err)
	}
}

func TestChoreService_SeedFutureOccurrences_StopsAtRecurrenceUntil(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 2)

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	until := now.AddDate(0, 0, 5) // series ends in 5 days
	seriesID := "until-series"
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceDaily,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		SeriesID:          &seriesID,
		RecurrenceUntil:   &until,
		LastAssignedIndex: -1,
	})

	// Ask to seed 30 days ahead; the end date must win.
	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 0, 30)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
	for _, c := range all {
		if c.DueDate != nil && c.DueDate.After(until) {
			t.Errorf("occurrence due %v is past recurrence_until %v", c.DueDate, until)
		}
	}
}

func TestChoreService_SeedFutureOccurrences_StopsAfterRecurrenceCount(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 2)

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	count := 3
	seriesID := "count-series"
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceDaily,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		SeriesID:          &seriesID,
		RecurrenceCount:   &count,
		LastAssignedIndex: -1,
	})

	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 0, 30)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	total, _ := choreRepo.CountBySeries(ctx, seriesID)
	if total != count {
		t.Errorf("expected exactly %d occurrences in capped series, got %d", count, total)
	}

	// Re-seeding must not exceed the cap.
	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 0, 30)); err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	total, _ = choreRepo.CountBySeries(ctx, seriesID)
	if total != count {
		t.Errorf("cap breached on re-seed: got %d, want %d", total, count)
	}
}

func TestChoreService_SeedFutureOccurrences_SkipsPastNoBackfill(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 1)

	now := time.Now()
	base := now.AddDate(0, 0, -10) // anchor due 10 days ago
	seriesID := "past-series"
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceDaily,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		SeriesID:          &seriesID,
		LastAssignedIndex: -1,
	})

	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 0, 5)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
	for _, c := range all {
		if c.ID == chore.ID {
			continue // the anchor itself is allowed to be in the past
		}
		if c.DueDate != nil && c.DueDate.Before(now) {
			t.Errorf("seeded occurrence due %v is in the past; catch-up must not back-fill", c.DueDate)
		}
	}
}

func TestChoreService_SeedFutureOccurrences_SkipsRecurOnComplete(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 1)

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Ad-hoc",
		CreatedByUserID: users[0].ID,
		RecurrenceType:  models.RecurrenceWeekly,
		RecurrenceValue: `{"interval":1}`,
		RecurOnComplete: true,
		DueDate:         &base,
		Status:          models.ChoreStatusPending,
	})

	if err := service.SeedFutureOccurrences(ctx, chore, now.AddDate(0, 1, 0)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{})
	if len(all) != 1 {
		t.Errorf("RecurOnComplete chores should not be seeded; want 1 chore, got %d", len(all))
	}
}

func TestChoreService_SeedFutureOccurrences_Idempotent(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 1)

	now := time.Now()
	base := now.AddDate(0, 0, 1)
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Daily",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceDaily,
		DueDate:           &base,
		Status:            models.ChoreStatusPending,
		LastAssignedIndex: -1,
	})
	seriesID := chore.ID
	chore.SeriesID = &seriesID
	choreRepo.Update(ctx, chore)

	until := now.AddDate(0, 0, 7)
	service.SeedFutureOccurrences(ctx, chore, until)
	firstCount, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})

	// Seed again — should not create duplicates
	service.SeedFutureOccurrences(ctx, chore, until)
	secondCount, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})

	if len(firstCount) != len(secondCount) {
		t.Errorf("seeding twice should be idempotent: first=%d second=%d", len(firstCount), len(secondCount))
	}
}

func TestChoreService_CompleteChore_SeedsAhead(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 2)

	now := time.Now()
	dueDate := now.AddDate(0, 0, -1) // overdue
	seriesID := "test-series"
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Weekly",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceWeekly,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &dueDate,
		Status:            models.ChoreStatusPending,
		SeriesID:          &seriesID,
		LastAssignedIndex: -1,
	})
	chore, _ = service.AssignNextUser(ctx, chore)

	if err := service.CompleteChore(ctx, chore.ID, users[0].ID); err != nil {
		t.Fatalf("CompleteChore: %v", err)
	}

	pending, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{
		Statuses: []models.ChoreStatus{models.ChoreStatusPending},
	})
	if len(pending) == 0 {
		t.Error("completing a recurring chore should seed future pending instances")
	}
}

func TestChoreService_SeedExistingRecurringChores(t *testing.T) {
	service, choreRepo, _, userRepo := setupChoreService(t)
	ctx := context.Background()
	users := createUsers(t, userRepo, 1)

	now := time.Now()
	dueDate := now.AddDate(0, 0, 1)
	// Create a legacy recurring chore with no series_id
	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:              "Legacy Weekly",
		CreatedByUserID:   users[0].ID,
		RecurrenceType:    models.RecurrenceWeekly,
		RecurrenceValue:   `{"interval":1}`,
		DueDate:           &dueDate,
		Status:            models.ChoreStatusPending,
		LastAssignedIndex: -1,
	})
	if chore.SeriesID != nil {
		t.Fatal("freshly created chore should have nil series_id for this test")
	}

	until := now.AddDate(0, 0, 21) // 3 weeks
	if err := service.SeedExistingRecurringChores(ctx, until); err != nil {
		t.Fatalf("SeedExistingRecurringChores: %v", err)
	}

	all, _ := choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
	if len(all) < 3 {
		t.Errorf("want at least 3 pending instances (1 + 2 seeded), got %d", len(all))
	}

	// Verify series_id was set on the original chore
	updated, _ := choreRepo.FindByID(ctx, chore.ID)
	if updated.SeriesID == nil {
		t.Error("original chore should have series_id set after seeding")
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
