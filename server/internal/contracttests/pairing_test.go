package contracttests

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type pairingFake struct {
	mutex        sync.Mutex
	hash, status string
	expires      time.Time
}

func (fake *pairingFake) CreateChallenge(_ context.Context, _ string, hash string, expires time.Time) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	fake.hash = hash
	fake.status = "pending"
	fake.expires = expires
	return nil
}
func (fake *pairingFake) ApproveChallenge(_ context.Context, _ string, hash, _, _ string) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if fake.hash != hash || fake.status != "pending" || !time.Now().Before(fake.expires) {
		return app.ErrPairingUnavailable
	}
	fake.status = "approved"
	return nil
}
func (fake *pairingFake) ClaimChallenge(_ context.Context, _ string, hash string, _ models.APIToken) error {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if fake.hash != hash || fake.status != "approved" || !time.Now().Before(fake.expires) {
		return app.ErrPairingUnavailable
	}
	fake.status = "claimed"
	return nil
}
func TestPairingClaimContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.PairingStore, string){
		"fake": func(*testing.T) (app.PairingStore, string) { return &pairingFake{}, "user" },
		"sqlite": func(t *testing.T) (app.PairingStore, string) {
			database := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(database).Create(context.Background(), models.User{OIDCSubject: "pairing-contract", Email: "pairing@example.com", Name: "Admin", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewDisplayPairingRepository(database), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, user := factory(t)
			ctx := context.Background()
			if err := store.CreateChallenge(ctx, "challenge", "secret-hash", time.Now().Add(5*time.Minute)); err != nil {
				t.Fatal(err)
			}
			if err := store.ClaimChallenge(ctx, "challenge", "secret-hash", models.APIToken{}); !errors.Is(err, app.ErrPairingUnavailable) {
				t.Fatalf("unapproved claim: %v", err)
			}
			if err := store.ApproveChallenge(ctx, "challenge", "wrong", "Kitchen", user); !errors.Is(err, app.ErrPairingUnavailable) {
				t.Fatalf("wrong secret: %v", err)
			}
			if err := store.ApproveChallenge(ctx, "challenge", "secret-hash", "Kitchen", user); err != nil {
				t.Fatal(err)
			}
			const attempts = 12
			results := make(chan error, attempts)
			start := make(chan struct{})
			for i := range attempts {
				go func() {
					<-start
					results <- store.ClaimChallenge(ctx, "challenge", "secret-hash", models.APIToken{TokenHash: fmt.Sprintf("token-%d", i)})
				}()
			}
			close(start)
			successes := 0
			for range attempts {
				err := <-results
				if err == nil {
					successes++
				} else if !errors.Is(err, app.ErrPairingUnavailable) {
					t.Fatalf("claim error: %v", err)
				}
			}
			if successes != 1 {
				t.Fatalf("successful claims=%d", successes)
			}
		})
	}
}
