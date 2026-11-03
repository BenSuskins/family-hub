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

func TestChoreTransactionRollsBackAllRepositories(t *testing.T) {
	ctx := context.Background()
	db := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(db)
	user, err := users.Create(ctx, models.User{OIDCSubject: "transaction-user", Email: "transaction@example.com", Name: "Transaction", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	store := repository.NewChoreRepository(db)
	task, err := store.Create(ctx, models.Chore{Name: "Before", CreatedByUserID: user.ID, Status: models.ChoreStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("cancel transaction")
	err = store.Transact(ctx, func(chores choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, series choreapp.ChoreSeriesRepository) error {
		task.Name = "After"
		if err := chores.Update(ctx, task); err != nil {
			return err
		}
		if _, err := assignments.Create(ctx, models.ChoreAssignment{ChoreID: task.ID, UserID: user.ID, Status: models.AssignmentStatusAssigned}); err != nil {
			return err
		}
		if _, err := series.Create(ctx, models.ChoreSeries{ID: task.ID, Name: "Definition", CreatedByUserID: user.ID}); err != nil {
			return err
		}
		if err := users.UpdateRole(ctx, user.ID, models.RoleAdmin); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatalf("expected transaction failure, got %v", err)
	}
	persisted, err := store.FindByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Name != "Before" {
		t.Fatalf("chore update committed: %q", persisted.Name)
	}
	records, err := repository.NewChoreAssignmentRepository(db).FindByChoreID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatal("assignment committed")
	}
	definition, err := repository.NewChoreSeriesRepository(db).FindByID(ctx, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if definition != nil {
		t.Fatal("definition committed")
	}
	persistedUser, err := users.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedUser.Role != models.RoleMember {
		t.Fatal("user role committed")
	}
}
