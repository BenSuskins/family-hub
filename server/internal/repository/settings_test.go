package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestSettingsRepository_GetDefault(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewSettingsRepository(db)
	ctx := context.Background()

	value, err := repo.Get(ctx, "family_name")
	if err != nil {
		t.Fatalf("getting default setting: %v", err)
	}
	if value != "Family" {
		t.Errorf("expected default 'Family', got '%s'", value)
	}
}

func TestSettingsRepository_SetAndGet(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewSettingsRepository(db)
	ctx := context.Background()

	if err := repo.Set(ctx, "family_name", "Smith"); err != nil {
		t.Fatalf("setting value: %v", err)
	}

	value, err := repo.Get(ctx, "family_name")
	if err != nil {
		t.Fatalf("getting value: %v", err)
	}
	if value != "Smith" {
		t.Errorf("expected 'Smith', got '%s'", value)
	}
}

func TestSettingsRepository_SetOverwrite(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewSettingsRepository(db)
	ctx := context.Background()

	repo.Set(ctx, "family_name", "Jones")
	repo.Set(ctx, "family_name", "Williams")

	value, err := repo.Get(ctx, "family_name")
	if err != nil {
		t.Fatalf("getting value: %v", err)
	}
	if value != "Williams" {
		t.Errorf("expected 'Williams', got '%s'", value)
	}
}

func TestSettingsRepository_GetMissing(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	repo := repository.NewSettingsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "nonexistent_key")
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}
