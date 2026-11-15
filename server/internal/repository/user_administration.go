package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/application/users"
	"github.com/bensuskins/family-hub/internal/models"
)

func (repository *SQLiteUserRepository) List(ctx context.Context) ([]users.User, error) {
	records, err := repository.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	var result []users.User
	for _, record := range records {
		result = append(result, applicationUser(record))
	}
	return result, nil
}
func (repository *SQLiteUserRepository) Find(ctx context.Context, id string) (users.User, error) {
	record, err := repository.FindByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return users.User{}, users.ErrNotFound
	}
	return applicationUser(record), err
}
func (repository *SQLiteUserRepository) SetRole(ctx context.Context, id string, role users.Role) error {
	return repository.UpdateRole(ctx, id, models.Role(role))
}
func (repository *SQLiteUserRepository) SetAvatar(ctx context.Context, id, data string) error {
	return repository.UpdateAvatar(ctx, id, data)
}
func applicationUser(record models.User) users.User {
	var onboarded *time.Time
	if record.OnboardedAt != nil {
		value := *record.OnboardedAt
		onboarded = &value
	}
	return users.User{ID: record.ID, OIDCSubject: record.OIDCSubject, Email: record.Email, Name: record.Name, AvatarURL: record.AvatarURL, Role: users.Role(record.Role), CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt, OnboardedAt: onboarded}
}
