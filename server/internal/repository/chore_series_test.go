package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func createTestUserNamed(t *testing.T, repo *repository.SQLiteUserRepository, name string) models.User {
	t.Helper()
	user, err := repo.Create(context.Background(), models.User{
		OIDCSubject: "sub-" + name,
		Email:       name + "@example.com",
		Name:        name,
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating test user %s: %v", name, err)
	}
	return user
}

func TestChoreSeriesRepository_CreateAndFindByID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	until := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	count := 12
	cursor := user.ID
	created, err := seriesRepo.Create(ctx, models.ChoreSeries{
		Name:                 "Bins",
		Description:          "Take the bins out",
		CreatedByUserID:      user.ID,
		RecurrenceType:       models.RecurrenceWeekly,
		RecurrenceValue:      `{"interval":1}`,
		RecurOnComplete:      false,
		RecurrenceUntil:      &until,
		RecurrenceCount:      &count,
		RotationCursorUserID: &cursor,
	})
	if err != nil {
		t.Fatalf("creating series: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated id")
	}

	found, err := seriesRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding series: %v", err)
	}
	if found == nil {
		t.Fatal("expected series, got nil")
	}
	if found.Name != "Bins" || found.RecurrenceType != models.RecurrenceWeekly {
		t.Errorf("unexpected series: %+v", found)
	}
	if found.RecurrenceUntil == nil || !found.RecurrenceUntil.Equal(until) {
		t.Errorf("recurrence_until round-trip failed: %v", found.RecurrenceUntil)
	}
	if found.RecurrenceCount == nil || *found.RecurrenceCount != count {
		t.Errorf("recurrence_count round-trip failed: %v", found.RecurrenceCount)
	}
	if found.RotationCursorUserID == nil || *found.RotationCursorUserID != cursor {
		t.Errorf("rotation cursor round-trip failed: %v", found.RotationCursorUserID)
	}
}

func TestChoreSeriesRepository_FindByID_NotFound(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	seriesRepo := repository.NewChoreSeriesRepository(db)

	found, err := seriesRepo.FindByID(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("expected nil error for missing series, got %v", err)
	}
	if found != nil {
		t.Errorf("expected nil series for missing id, got %+v", found)
	}
}

func TestChoreSeriesRepository_RotationCursorAndEligible(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	seriesRepo := repository.NewChoreSeriesRepository(db)
	ctx := context.Background()

	u1 := createTestUserNamed(t, userRepo, "Ada")
	u2 := createTestUserNamed(t, userRepo, "Bea")

	series, err := seriesRepo.Create(ctx, models.ChoreSeries{
		Name:            "Dishes",
		CreatedByUserID: u1.ID,
		RecurrenceType:  models.RecurrenceDaily,
	})
	if err != nil {
		t.Fatalf("creating series: %v", err)
	}

	if err := seriesRepo.SetEligibleAssignees(ctx, series.ID, []string{u1.ID, u2.ID}); err != nil {
		t.Fatalf("setting eligible: %v", err)
	}
	if err := seriesRepo.SetRotationCursor(ctx, series.ID, u2.ID); err != nil {
		t.Fatalf("setting cursor: %v", err)
	}

	found, err := seriesRepo.FindByID(ctx, series.ID)
	if err != nil {
		t.Fatalf("finding series: %v", err)
	}
	if len(found.EligibleAssignees) != 2 {
		t.Errorf("expected 2 eligible assignees, got %d", len(found.EligibleAssignees))
	}
	if found.RotationCursorUserID == nil || *found.RotationCursorUserID != u2.ID {
		t.Errorf("expected cursor %s, got %v", u2.ID, found.RotationCursorUserID)
	}

	// Soft delete sets deleted_at.
	if err := seriesRepo.MarkDeleted(ctx, series.ID); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}
	found, _ = seriesRepo.FindByID(ctx, series.ID)
	if found.DeletedAt == nil {
		t.Error("expected deleted_at to be set after MarkDeleted")
	}
}
