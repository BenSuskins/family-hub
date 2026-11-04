package repository_test

import (
	"context"
	"errors"
	"testing"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestChoreTransactionCommitAndNestedRollback(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(db)
	user, err := users.Create(ctx, models.User{OIDCSubject: "tx-behavior", Email: "tx@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	store := repository.NewChoreRepository(db)
	task, err := store.Create(ctx, models.Chore{Name: "Original", CreatedByUserID: user.ID, Status: models.ChoreStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	err = store.Transact(ctx, func(chores choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, series choreapp.ChoreSeriesRepository) error {
		task.Name = "Committed"
		return chores.Update(ctx, task)
	})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := store.FindByID(ctx, task.ID)
	if err != nil || saved.Name != "Committed" {
		t.Fatalf("commit: %+v, %v", saved, err)
	}
	failure := errors.New("outer failure")
	err = store.Transact(ctx, func(chores choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, series choreapp.ChoreSeriesRepository) error {
		return chores.Transact(ctx, func(nested choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, series choreapp.ChoreSeriesRepository) error {
			saved.Name = "Rolled back"
			if err := nested.Update(ctx, saved); err != nil {
				return err
			}
			return failure
		})
	})
	if !errors.Is(err, failure) {
		t.Fatalf("nested failure: %v", err)
	}
	saved, err = store.FindByID(ctx, task.ID)
	if err != nil || saved.Name != "Committed" {
		t.Fatalf("nested rollback: %+v, %v", saved, err)
	}
}

func TestChoreTransactionCancelledContextDoesNotRunCallback(t *testing.T) {
	store := repository.NewChoreRepository(testutil.NewTestDatabase(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	err := store.Transact(ctx, func(choreapp.ChoreRepository, choreapp.ChoreAssignmentRepository, choreapp.UserRepository, choreapp.ChoreSeriesRepository) error {
		called = true
		return nil
	})
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("err=%v callback ran=%v", err, called)
	}
}
