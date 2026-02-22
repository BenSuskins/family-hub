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

func TestProvisionUser_PreservesCustomAvatarOnLogin(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	service, err := NewAuthService(context.Background(), config.Config{SessionSecret: "test-secret"}, userRepo)
	if err != nil {
		t.Fatalf("creating auth service: %v", err)
	}
	ctx := context.Background()

	// First login creates the user with an OIDC avatar
	user, err := service.provisionUser(ctx, "subject-123", "alice@test.com", "Alice", "https://oidc.example.com/pic.png")
	if err != nil {
		t.Fatalf("first provisionUser: %v", err)
	}

	// User uploads a custom avatar
	if err := userRepo.UpdateAvatar(ctx, user.ID, "data:image/png;base64,abc="); err != nil {
		t.Fatalf("UpdateAvatar: %v", err)
	}

	// Second login with a different OIDC avatar URL
	returned, err := service.provisionUser(ctx, "subject-123", "alice@test.com", "Alice", "https://oidc.example.com/new-pic.png")
	if err != nil {
		t.Fatalf("second provisionUser: %v", err)
	}

	// Custom avatar URL must be preserved
	if returned.AvatarURL != "/avatar/"+user.ID {
		t.Errorf("expected custom avatar URL '/avatar/%s', got %q", user.ID, returned.AvatarURL)
	}
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
	if user.OIDCSubject != devUserOIDCSubject {
		t.Errorf("expected OIDCSubject %q, got %q", devUserOIDCSubject, user.OIDCSubject)
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
