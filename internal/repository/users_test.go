package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestUserRepository_CreateAndFindByID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := models.User{
		OIDCSubject: "sub-123",
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleAdmin,
	}

	created, err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding user: %v", err)
	}
	if found.Name != "Test User" {
		t.Errorf("expected name 'Test User', got '%s'", found.Name)
	}
	if found.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got '%s'", found.Role)
	}
}

func TestUserRepository_FindByOIDCSubject(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	user := models.User{
		OIDCSubject: "unique-subject",
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleMember,
	}
	repo.Create(ctx, user)

	found, err := repo.FindByOIDCSubject(ctx, "unique-subject")
	if err != nil {
		t.Fatalf("finding user by subject: %v", err)
	}
	if found.Email != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got '%s'", found.Email)
	}
}

func TestUserRepository_FindAll(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, models.User{OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleAdmin})
	repo.Create(ctx, models.User{OIDCSubject: "s2", Email: "b@test.com", Name: "Bob", Role: models.RoleMember})

	users, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestUserRepository_UpdateRole(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, models.User{
		OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleMember,
	})

	if err := repo.UpdateRole(ctx, created.ID, models.RoleAdmin); err != nil {
		t.Fatalf("updating role: %v", err)
	}

	found, _ := repo.FindByID(ctx, created.ID)
	if found.Role != models.RoleAdmin {
		t.Errorf("expected admin role, got '%s'", found.Role)
	}
}

func TestUserRepository_Count(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("counting users: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	repo.Create(ctx, models.User{OIDCSubject: "s1", Email: "a@test.com", Name: "Alice", Role: models.RoleAdmin})

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("counting users: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}
