package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type UserRepository interface {
	FindByID(ctx context.Context, id string) (models.User, error)
	FindByOIDCSubject(ctx context.Context, subject string) (models.User, error)
	FindAll(ctx context.Context) ([]models.User, error)
	Create(ctx context.Context, user models.User) (models.User, error)
	UpdateRole(ctx context.Context, id string, role models.Role) error
	UpdateProfile(ctx context.Context, id string, name string, email string, avatarURL string) error
	Count(ctx context.Context) (int, error)
}

type SQLiteUserRepository struct {
	database *sql.DB
}

func NewUserRepository(database *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{database: database}
}

func (repository *SQLiteUserRepository) FindByID(ctx context.Context, id string) (models.User, error) {
	var user models.User
	err := repository.database.QueryRowContext(ctx,
		"SELECT id, oidc_subject, email, name, avatar_url, role, created_at, updated_at FROM users WHERE id = ?", id,
	).Scan(&user.ID, &user.OIDCSubject, &user.Email, &user.Name, &user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("finding user by id: %w", err)
	}
	return user, nil
}

func (repository *SQLiteUserRepository) FindByOIDCSubject(ctx context.Context, subject string) (models.User, error) {
	var user models.User
	err := repository.database.QueryRowContext(ctx,
		"SELECT id, oidc_subject, email, name, avatar_url, role, created_at, updated_at FROM users WHERE oidc_subject = ?", subject,
	).Scan(&user.ID, &user.OIDCSubject, &user.Email, &user.Name, &user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("finding user by oidc subject: %w", err)
	}
	return user, nil
}

func (repository *SQLiteUserRepository) FindAll(ctx context.Context) ([]models.User, error) {
	rows, err := repository.database.QueryContext(ctx,
		"SELECT id, oidc_subject, email, name, avatar_url, role, created_at, updated_at FROM users ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("finding all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.OIDCSubject, &user.Email, &user.Name, &user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (repository *SQLiteUserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := repository.database.ExecContext(ctx,
		"INSERT INTO users (id, oidc_subject, email, name, avatar_url, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		user.ID, user.OIDCSubject, user.Email, user.Name, user.AvatarURL, user.Role, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return models.User{}, fmt.Errorf("creating user: %w", err)
	}
	return user, nil
}

func (repository *SQLiteUserRepository) UpdateRole(ctx context.Context, id string, role models.Role) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET role = ?, updated_at = ? WHERE id = ?",
		role, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating user role: %w", err)
	}
	return nil
}

func (repository *SQLiteUserRepository) UpdateProfile(ctx context.Context, id string, name string, email string, avatarURL string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET name = ?, email = ?, avatar_url = ?, updated_at = ? WHERE id = ?",
		name, email, avatarURL, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating user profile: %w", err)
	}
	return nil
}

func (repository *SQLiteUserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := repository.database.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}
