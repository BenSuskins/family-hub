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
	FindAvatarData(ctx context.Context, userID string) (string, error)
	UpdateAvatar(ctx context.Context, userID string, dataURI string) error
	ClearAvatar(ctx context.Context, userID string) error
	Count(ctx context.Context) (int, error)
	MarkOnboarded(ctx context.Context, id string) error
}

type SQLiteUserRepository struct {
	database *sql.DB
}

func NewUserRepository(database *sql.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{database: database}
}

const userColumns = "id, oidc_subject, email, name, avatar_url, role, created_at, updated_at, onboarded_at"

func (repository *SQLiteUserRepository) FindByID(ctx context.Context, id string) (models.User, error) {
	var user models.User
	err := repository.database.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM users WHERE id = ?", userColumns), id,
	).Scan(scanUserFields(&user)...)
	if err != nil {
		return models.User{}, fmt.Errorf("finding user by id: %w", err)
	}
	return user, nil
}

func (repository *SQLiteUserRepository) FindByOIDCSubject(ctx context.Context, subject string) (models.User, error) {
	var user models.User
	err := repository.database.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM users WHERE oidc_subject = ?", userColumns), subject,
	).Scan(scanUserFields(&user)...)
	if err != nil {
		return models.User{}, fmt.Errorf("finding user by oidc subject: %w", err)
	}
	return user, nil
}

func (repository *SQLiteUserRepository) FindAll(ctx context.Context) ([]models.User, error) {
	rows, err := repository.database.QueryContext(ctx,
		fmt.Sprintf("SELECT %s FROM users ORDER BY name", userColumns),
	)
	if err != nil {
		return nil, fmt.Errorf("finding all users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(scanUserFields(&user)...); err != nil {
			return nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func scanUserFields(user *models.User) []any {
	return []any{
		&user.ID, &user.OIDCSubject, &user.Email, &user.Name,
		&user.AvatarURL, &user.Role, &user.CreatedAt, &user.UpdatedAt,
		&user.OnboardedAt,
	}
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

func (repository *SQLiteUserRepository) FindAvatarData(ctx context.Context, userID string) (string, error) {
	var avatarData string
	err := repository.database.QueryRowContext(ctx,
		"SELECT avatar_data FROM users WHERE id = ?", userID,
	).Scan(&avatarData)
	if err != nil {
		return "", fmt.Errorf("finding avatar data: %w", err)
	}
	return avatarData, nil
}

func (repository *SQLiteUserRepository) UpdateAvatar(ctx context.Context, userID string, dataURI string) error {
	avatarURL := "/avatar/" + userID
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET avatar_data = ?, avatar_url = ?, updated_at = ? WHERE id = ?",
		dataURI, avatarURL, time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("updating avatar: %w", err)
	}
	return nil
}

func (repository *SQLiteUserRepository) ClearAvatar(ctx context.Context, userID string) error {
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET avatar_data = '', avatar_url = '', updated_at = ? WHERE id = ?",
		time.Now(), userID,
	)
	if err != nil {
		return fmt.Errorf("clearing avatar: %w", err)
	}
	return nil
}

func (repository *SQLiteUserRepository) MarkOnboarded(ctx context.Context, id string) error {
	now := time.Now()
	_, err := repository.database.ExecContext(ctx,
		"UPDATE users SET onboarded_at = COALESCE(onboarded_at, ?), updated_at = ? WHERE id = ?",
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("marking user as onboarded: %w", err)
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
