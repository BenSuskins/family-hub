package repository_test

import (
	"context"
	"errors"
	"testing"

	domain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestShoppingMissingReturnsDomainError(t *testing.T) {
	repo := repository.NewShoppingRepository(testutil.NewTestDatabase(t))
	if err := repo.Delete(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}
