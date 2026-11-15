package repository

import (
	"context"

	app "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/models"
)

func (repository *SQLiteAPITokenRepository) CreateManagedToken(ctx context.Context, token app.TokenRecord) (app.TokenRecord, error) {
	record, err := repository.Create(ctx, models.APIToken{ID: token.ID, Name: token.Name, TokenHash: token.TokenHash, Scope: models.TokenScope(token.Scope), CreatedByUserID: token.CreatedByUserID, ExpiresAt: token.ExpiresAt, CreatedAt: token.CreatedAt})
	if err != nil {
		return app.TokenRecord{}, err
	}
	return managedTokenRecord(record), nil
}
func (repository *SQLiteAPITokenRepository) ListManagedTokens(ctx context.Context) ([]app.TokenRecord, error) {
	records, err := repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	var result []app.TokenRecord
	for _, record := range records {
		result = append(result, managedTokenRecord(record))
	}
	return result, nil
}
func (repository *SQLiteAPITokenRepository) DeleteManagedToken(ctx context.Context, id string) error {
	return repository.Delete(ctx, id)
}
func managedTokenRecord(record models.APIToken) app.TokenRecord {
	var expiresAt = record.ExpiresAt
	if expiresAt != nil {
		copy := *expiresAt
		expiresAt = &copy
	}
	return app.TokenRecord{ID: record.ID, Name: record.Name, TokenHash: record.TokenHash, Scope: string(record.Scope), CreatedByUserID: record.CreatedByUserID, ExpiresAt: expiresAt, CreatedAt: record.CreatedAt}
}
