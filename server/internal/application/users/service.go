package users

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type User struct {
	ID, OIDCSubject, Email, Name, AvatarURL string
	Role                                    Role
	CreatedAt, UpdatedAt                    time.Time
	OnboardedAt                             *time.Time
}
type Store interface {
	List(context.Context) ([]User, error)
	Find(context.Context, string) (User, error)
	SetRole(context.Context, string, Role) error
	SetAvatar(context.Context, string, string) error
	ClearAvatar(context.Context, string) error
}

var ErrReload = errors.New("reload user")
var ErrNotFound = errors.New("user not found")

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) List(ctx context.Context) ([]User, error) {
	users, err := service.store.List(ctx)
	if err != nil || users == nil {
		return users, err
	}
	result := make([]User, len(users))
	for index, user := range users {
		result[index] = copyUser(user)
	}
	return result, nil
}
func (service *Service) Find(ctx context.Context, id string) (User, error) {
	user, err := service.store.Find(ctx, id)
	return copyUser(user), err
}
func (service *Service) ChangeRole(ctx context.Context, id string, role Role) (User, error) {
	if err := service.store.SetRole(ctx, id, role); err != nil {
		return User{}, err
	}
	return service.reload(ctx, id)
}
func (service *Service) SaveAvatar(ctx context.Context, id, data string) (User, error) {
	if err := service.store.SetAvatar(ctx, id, data); err != nil {
		return User{}, err
	}
	return service.reload(ctx, id)
}
func (service *Service) RemoveAvatar(ctx context.Context, id string) error {
	return service.store.ClearAvatar(ctx, id)
}
func (service *Service) reload(ctx context.Context, id string) (User, error) {
	user, err := service.store.Find(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("%w: %w", ErrReload, err)
	}
	return copyUser(user), nil
}

func copyUser(user User) User {
	if user.OnboardedAt != nil {
		value := *user.OnboardedAt
		user.OnboardedAt = &value
	}
	return user
}
