package services_test

import (
	"context"
	"testing"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type choreUsersFake struct{ users []models.User }

func (fake choreUsersFake) FindAll(context.Context) ([]models.User, error) {
	return append([]models.User(nil), fake.users...), nil
}

func TestChoreAssignmentUsesUserCapability(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(context.Background(), models.User{OIDCSubject: "member", Email: "member@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	chores := repository.NewChoreRepository(database)
	chore, err := chores.Create(context.Background(), models.Chore{Name: "Wash up", CreatedByUserID: user.ID, Status: models.ChoreStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	service := choreapp.NewChoreService(chores, repository.NewChoreAssignmentRepository(database), choreUsersFake{users: []models.User{user}}, nil)
	assigned, err := service.AssignNextUser(context.Background(), chore)
	if err != nil {
		t.Fatal(err)
	}
	if assigned.AssignedToUserID == nil || *assigned.AssignedToUserID != user.ID {
		t.Fatalf("unexpected assignee: %+v", assigned.AssignedToUserID)
	}
}
