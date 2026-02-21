package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type APITokenRepository interface {
	Create(ctx context.Context, token models.APIToken) (models.APIToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (models.APIToken, error)
	FindByUserIDAndName(ctx context.Context, userID string, name string) ([]models.APIToken, error)
	FindAll(ctx context.Context) ([]models.APIToken, error)
	Delete(ctx context.Context, id string) error
}

type SQLiteAPITokenRepository struct {
	database *sql.DB
}

func NewAPITokenRepository(database *sql.DB) *SQLiteAPITokenRepository {
	return &SQLiteAPITokenRepository{database: database}
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (repository *SQLiteAPITokenRepository) Create(ctx context.Context, token models.APIToken) (models.APIToken, error) {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	token.CreatedAt = time.Now()

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO api_tokens (id, name, token_hash, scope, created_by_user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		token.ID, token.Name, token.TokenHash, token.Scope, token.CreatedByUserID, token.ExpiresAt, token.CreatedAt,
	)
	if err != nil {
		return models.APIToken{}, fmt.Errorf("creating api token: %w", err)
	}
	return token, nil
}

func (repository *SQLiteAPITokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (models.APIToken, error) {
	var token models.APIToken
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, name, token_hash, scope, created_by_user_id, expires_at, created_at
		FROM api_tokens WHERE token_hash = ?`, tokenHash,
	).Scan(&token.ID, &token.Name, &token.TokenHash, &token.Scope, &token.CreatedByUserID, &token.ExpiresAt, &token.CreatedAt)
	if err != nil {
		return models.APIToken{}, fmt.Errorf("finding token by hash: %w", err)
	}
	return token, nil
}

func (repository *SQLiteAPITokenRepository) FindByUserIDAndName(ctx context.Context, userID string, name string) ([]models.APIToken, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, token_hash, scope, created_by_user_id, expires_at, created_at
		FROM api_tokens WHERE created_by_user_id = ? AND name = ? ORDER BY created_at DESC`,
		userID, name,
	)
	if err != nil {
		return nil, fmt.Errorf("finding tokens by user and name: %w", err)
	}
	defer rows.Close()

	var tokens []models.APIToken
	for rows.Next() {
		var token models.APIToken
		if err := rows.Scan(&token.ID, &token.Name, &token.TokenHash, &token.Scope, &token.CreatedByUserID, &token.ExpiresAt, &token.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (repository *SQLiteAPITokenRepository) FindAll(ctx context.Context) ([]models.APIToken, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, token_hash, scope, created_by_user_id, expires_at, created_at
		FROM api_tokens ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding all tokens: %w", err)
	}
	defer rows.Close()

	var tokens []models.APIToken
	for rows.Next() {
		var token models.APIToken
		if err := rows.Scan(&token.ID, &token.Name, &token.TokenHash, &token.Scope, &token.CreatedByUserID, &token.ExpiresAt, &token.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning token: %w", err)
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (repository *SQLiteAPITokenRepository) Delete(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM api_tokens WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting token: %w", err)
	}
	return nil
}
