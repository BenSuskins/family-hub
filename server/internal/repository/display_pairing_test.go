package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestDisplayPairingRequiresApprovalAndAllowsOneClaim(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	ctx := context.Background()
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "display-pairing", Email: "display@example.com", Name: "Display Admin", Role: models.RoleAdmin})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	pairings := repository.NewDisplayPairingRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	secretHash := repository.HashToken("pairing-secret")
	if err := pairings.CreateChallenge(ctx, "challenge-1", secretHash, time.Now().Add(5*time.Minute)); err != nil {
		t.Fatalf("creating challenge: %v", err)
	}
	if err := pairings.ClaimChallenge(ctx, "challenge-1", secretHash, models.APIToken{TokenHash: repository.HashToken("display-token")}); !errors.Is(err, repository.ErrPairingUnavailable) {
		t.Fatalf("unapproved challenge claim error = %v", err)
	}
	if err := pairings.ApproveChallenge(ctx, "challenge-1", repository.HashToken("wrong-secret"), "Kitchen", user.ID); !errors.Is(err, repository.ErrPairingUnavailable) {
		t.Fatalf("wrong secret approval error = %v", err)
	}
	if err := pairings.ApproveChallenge(ctx, "challenge-1", secretHash, "Kitchen", user.ID); err != nil {
		t.Fatalf("approving challenge: %v", err)
	}
	if err := pairings.ClaimChallenge(ctx, "challenge-1", secretHash, models.APIToken{TokenHash: repository.HashToken("display-token")}); err != nil {
		t.Fatalf("claiming challenge: %v", err)
	}
	if err := pairings.ClaimChallenge(ctx, "challenge-1", secretHash, models.APIToken{TokenHash: repository.HashToken("second-token")}); !errors.Is(err, repository.ErrPairingUnavailable) {
		t.Fatalf("second claim error = %v", err)
	}
	token, err := tokens.FindByTokenHash(ctx, repository.HashToken("display-token"))
	if err != nil || token.Scope != models.TokenScopeDisplay || token.CreatedByUserID != "" {
		t.Fatalf("display token = %#v, %v", token, err)
	}
	displays, err := pairings.List(ctx)
	if err != nil || len(displays) != 1 || displays[0].Name != "Kitchen" {
		t.Fatalf("paired displays = %#v, %v", displays, err)
	}
	if _, err := database.ExecContext(ctx, `DELETE FROM users WHERE id=?`, user.ID); err != nil {
		t.Fatalf("deleting pairing admin: %v", err)
	}
	displays, err = pairings.List(ctx)
	if err != nil || len(displays) != 1 || displays[0].Name != "Kitchen" {
		t.Fatalf("admin deletion removed paired display: %#v, %v", displays, err)
	}
	if err := pairings.Revoke(ctx, "challenge-1"); err != nil {
		t.Fatalf("revoking display: %v", err)
	}
	if _, err := tokens.FindByTokenHash(ctx, repository.HashToken("display-token")); err == nil {
		t.Fatal("revoking display left its token active")
	}
}
