package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCategoryRepository_CreateAndFindAll(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	category, err := categoryRepo.Create(ctx, models.Category{
		Name:            "Kitchen",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating category: %v", err)
	}
	if category.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	categories, err := categoryRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding categories: %v", err)
	}
	if len(categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(categories))
	}
	if categories[0].Name != "Kitchen" {
		t.Errorf("expected 'Kitchen', got '%s'", categories[0].Name)
	}
}

func TestCategoryRepository_Update(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := categoryRepo.Create(ctx, models.Category{
		Name: "Old Name", CreatedByUserID: user.ID,
	})

	if err := categoryRepo.Update(ctx, created.ID, "New Name"); err != nil {
		t.Fatalf("updating category: %v", err)
	}

	found, _ := categoryRepo.FindByID(ctx, created.ID)
	if found.Name != "New Name" {
		t.Errorf("expected 'New Name', got '%s'", found.Name)
	}
}

func TestCategoryRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := categoryRepo.Create(ctx, models.Category{
		Name: "To Delete", CreatedByUserID: user.ID,
	})

	if err := categoryRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("deleting category: %v", err)
	}

	categories, _ := categoryRepo.FindAll(ctx)
	if len(categories) != 0 {
		t.Errorf("expected 0 categories after delete, got %d", len(categories))
	}
}
