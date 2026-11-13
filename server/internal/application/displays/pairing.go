package displays

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrPairingUnavailable = errors.New("pairing challenge is invalid, expired, or already used")
	ErrSecretGeneration   = errors.New("could not generate pairing secret")
)

type PairingStore interface {
	CreateChallenge(context.Context, string, string, time.Time) error
	ApproveChallenge(context.Context, string, string, string, string) error
	ClaimChallenge(context.Context, string, string, models.APIToken) error
}
type Challenge struct {
	ID        string
	Code      string
	ExpiresAt time.Time
}
type Pairing struct {
	store  PairingStore
	secret func() (string, error)
	hash   func(string) string
	id     func() string
	now    func() time.Time
}

func NewPairing(store PairingStore, secret func() (string, error), hash func(string) string, id func() string, now func() time.Time) *Pairing {
	return &Pairing{store: store, secret: secret, hash: hash, id: id, now: now}
}
func (service *Pairing) Create(ctx context.Context) (Challenge, error) {
	secret, err := service.secret()
	if err != nil {
		return Challenge{}, fmt.Errorf("%w: %w", ErrSecretGeneration, err)
	}
	challenge := Challenge{ID: service.id(), Code: secret, ExpiresAt: service.now().UTC().Add(5 * time.Minute)}
	if err := service.store.CreateChallenge(ctx, challenge.ID, service.hash(secret), challenge.ExpiresAt); err != nil {
		return Challenge{}, err
	}
	return challenge, nil
}
func (service *Pairing) Approve(ctx context.Context, id, code, name, userID string) error {
	return service.store.ApproveChallenge(ctx, id, service.hash(code), name, userID)
}
func (service *Pairing) Claim(ctx context.Context, id, code string) (string, error) {
	secret, err := service.secret()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrSecretGeneration, err)
	}
	token := models.APIToken{Name: "Wall display", TokenHash: service.hash(secret), Scope: models.TokenScopeDisplay}
	if err := service.store.ClaimChallenge(ctx, id, service.hash(code), token); err != nil {
		return "", err
	}
	return secret, nil
}
