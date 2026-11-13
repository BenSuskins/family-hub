package displays_test

import (
	"context"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/bensuskins/family-hub/internal/models"
)

type pairingStoreFake struct {
	id, hash, name, user string
	expires              time.Time
	token                models.APIToken
}

func (f *pairingStoreFake) CreateChallenge(_ context.Context, id, hash string, expires time.Time) error {
	f.id = id
	f.hash = hash
	f.expires = expires
	return nil
}
func (f *pairingStoreFake) ApproveChallenge(_ context.Context, id, hash, name, user string) error {
	f.id = id
	f.hash = hash
	f.name = name
	f.user = user
	return nil
}
func (f *pairingStoreFake) ClaimChallenge(_ context.Context, id, hash string, token models.APIToken) error {
	f.id = id
	f.hash = hash
	f.token = token
	return nil
}
func TestPairingStoresOnlyHashesAndReturnsCredentialAfterClaim(t *testing.T) {
	store := &pairingStoreFake{}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := app.NewPairing(store, func() (string, error) { return "secret", nil }, func(string) string { return "hash" }, func() string { return "id" }, func() time.Time { return now })
	challenge, err := service.Create(context.Background())
	if err != nil || challenge.Code != "secret" || store.hash != "hash" || !store.expires.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("challenge=%+v store=%+v err=%v", challenge, store, err)
	}
	token, err := service.Claim(context.Background(), "id", "pairing-code")
	if err != nil || token != "secret" || store.token.TokenHash != "hash" || store.token.Scope != models.TokenScopeDisplay {
		t.Fatalf("token=%q stored=%+v err=%v", token, store.token, err)
	}
}

type failedClaimStore struct{ pairingStoreFake }

func (*failedClaimStore) ClaimChallenge(context.Context, string, string, models.APIToken) error {
	return app.ErrPairingUnavailable
}
func TestClaimNeverReturnsCredentialWhenStoreFails(t *testing.T) {
	store := &failedClaimStore{}
	service := app.NewPairing(store, func() (string, error) { return "secret", nil }, func(string) string { return "hash" }, func() string { return "id" }, time.Now)
	token, err := service.Claim(context.Background(), "id", "code")
	if err == nil || token != "" {
		t.Fatalf("token=%q err=%v", token, err)
	}
}
