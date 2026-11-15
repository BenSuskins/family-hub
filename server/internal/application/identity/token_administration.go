package identity

import (
	"context"
	"fmt"
	"time"
)

type TokenRecord struct {
	ID              string
	Name            string
	TokenHash       string
	Scope           string
	CreatedByUserID string
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}

type TokenStore interface {
	CreateManagedToken(context.Context, TokenRecord) (TokenRecord, error)
	ListManagedTokens(context.Context) ([]TokenRecord, error)
	DeleteManagedToken(context.Context, string) error
}

type CreatedToken struct{ ID, Name, Token string }

type TokenAdministration struct {
	store  TokenStore
	secret func() (string, error)
	hash   func(string) string
}

func NewTokenAdministration(store TokenStore, secret func() (string, error), hash func(string) string) *TokenAdministration {
	return &TokenAdministration{store: store, secret: secret, hash: hash}
}

func (service *TokenAdministration) Create(ctx context.Context, name, userID string) (CreatedToken, error) {
	secret, err := service.secret()
	if err != nil {
		return CreatedToken{}, fmt.Errorf("%w: %w", ErrGenerateToken, err)
	}
	record, err := service.store.CreateManagedToken(ctx, TokenRecord{Name: name, TokenHash: service.hash(secret), Scope: "api", CreatedByUserID: userID})
	if err != nil {
		return CreatedToken{}, fmt.Errorf("%w: %w", ErrCreateToken, err)
	}
	return CreatedToken{ID: record.ID, Name: record.Name, Token: secret}, nil
}

func (service *TokenAdministration) List(ctx context.Context) ([]TokenRecord, error) {
	records, err := service.store.ListManagedTokens(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]TokenRecord, len(records))
	for index, record := range records {
		result[index] = record
		if record.ExpiresAt != nil {
			expiry := *record.ExpiresAt
			result[index].ExpiresAt = &expiry
		}
	}
	return result, nil
}

func (service *TokenAdministration) Delete(ctx context.Context, id string) error {
	return service.store.DeleteManagedToken(ctx, id)
}
