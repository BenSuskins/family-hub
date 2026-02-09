package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestAPITokenRepository_CreateAndFindByHash(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewAPITokenRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	rawToken := "test-token-12345"
	tokenHash := repository.HashToken(rawToken)

	token := models.APIToken{
		Name:            "Test Token",
		TokenHash:       tokenHash,
		CreatedByUserID: user.ID,
	}

	created, err := tokenRepo.Create(ctx, token)
	if err != nil {
		t.Fatalf("creating token: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	found, err := tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		t.Fatalf("finding token by hash: %v", err)
	}
	if found.Name != "Test Token" {
		t.Errorf("expected 'Test Token', got '%s'", found.Name)
	}
}

func TestAPITokenRepository_FindAll(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewAPITokenRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	tokenRepo.Create(ctx, models.APIToken{
		Name: "Token 1", TokenHash: "hash1", CreatedByUserID: user.ID,
	})
	tokenRepo.Create(ctx, models.APIToken{
		Name: "Token 2", TokenHash: "hash2", CreatedByUserID: user.ID,
	})

	tokens, err := tokenRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding tokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}
}

func TestAPITokenRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewAPITokenRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := tokenRepo.Create(ctx, models.APIToken{
		Name: "To Delete", TokenHash: "hash-delete", CreatedByUserID: user.ID,
	})

	if err := tokenRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("deleting token: %v", err)
	}

	tokens, _ := tokenRepo.FindAll(ctx)
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after delete, got %d", len(tokens))
	}
}

func TestHashToken(t *testing.T) {
	hash1 := repository.HashToken("token1")
	hash2 := repository.HashToken("token2")

	if hash1 == hash2 {
		t.Error("different tokens should produce different hashes")
	}

	hash1Again := repository.HashToken("token1")
	if hash1 != hash1Again {
		t.Error("same token should produce same hash")
	}
}
