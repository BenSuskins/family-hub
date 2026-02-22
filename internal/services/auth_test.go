package services

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func newDevAuthService(t *testing.T) *AuthService {
	t.Helper()
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	service, err := NewAuthService(context.Background(), config.Config{SessionSecret: "test-secret"}, userRepo)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}
	return service
}

func TestDevLogin_CreatesDevAdminUser(t *testing.T) {
	service := newDevAuthService(t)

	user, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("DevLogin: %v", err)
	}

	if user.Name != "Dev Admin" {
		t.Errorf("expected name 'Dev Admin', got %q", user.Name)
	}
	if user.Email != "dev@localhost" {
		t.Errorf("expected email 'dev@localhost', got %q", user.Email)
	}
	if user.Role != models.RoleAdmin {
		t.Errorf("expected role admin, got %q", user.Role)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
}

func TestDevLogin_IdempotentOnSecondCall(t *testing.T) {
	service := newDevAuthService(t)

	first, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("first DevLogin: %v", err)
	}

	second, err := service.DevLogin(context.Background())
	if err != nil {
		t.Fatalf("second DevLogin: %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same user ID, got %q and %q", first.ID, second.ID)
	}
}
