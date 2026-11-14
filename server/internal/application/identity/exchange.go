package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/bensuskins/family-hub/internal/models"
)

var (
	ErrProviderRequest     = errors.New("invalid identity provider request")
	ErrProviderUnavailable = errors.New("identity provider unavailable")
	ErrProviderResponse    = errors.New("invalid identity provider response")
	ErrInvalidToken        = errors.New("invalid identity token")
	ErrLookup              = errors.New("failed to look up user")
	ErrCount               = errors.New("failed to determine user role")
	ErrCreateUser          = errors.New("failed to create user")
	ErrReplaceToken        = errors.New("failed to replace client token")
	ErrGenerateToken       = errors.New("failed to generate token")
	ErrCreateToken         = errors.New("failed to create token")
)

type UserInfo struct{ Subject, Email, Name, PreferredUsername, Picture string }
type Provider interface {
	UserInfo(context.Context, string) (UserInfo, error)
}
type Users interface {
	Lookup(context.Context, string) (models.User, bool, error)
	Count(context.Context) (int, error)
	Create(context.Context, models.User) (models.User, error)
}
type Tokens interface {
	FindByUserIDAndName(context.Context, string, string) ([]models.APIToken, error)
	Delete(context.Context, string) error
	Create(context.Context, models.APIToken) (models.APIToken, error)
}
type Result struct {
	Token string
	User  models.User
}
type Service struct {
	provider Provider
	users    Users
	tokens   Tokens
	secret   func() (string, error)
	hash     func(string) string
}

func New(provider Provider, users Users, tokens Tokens, secret func() (string, error), hash func(string) string) *Service {
	return &Service{provider: provider, users: users, tokens: tokens, secret: secret, hash: hash}
}
func (service *Service) Exchange(ctx context.Context, oidcToken, client string) (Result, error) {
	info, err := service.provider.UserInfo(ctx, oidcToken)
	if err != nil {
		return Result{}, err
	}
	user, exists, err := service.users.Lookup(ctx, info.Subject)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrLookup, err)
	}
	if !exists {
		name := info.Name
		if name == "" {
			name = info.PreferredUsername
		}
		if name == "" {
			name = info.Email
		}
		count, err := service.users.Count(ctx)
		if err != nil {
			return Result{}, fmt.Errorf("%w: %w", ErrCount, err)
		}
		role := models.RoleMember
		if count == 0 {
			role = models.RoleAdmin
		}
		user, err = service.users.Create(ctx, models.User{OIDCSubject: info.Subject, Email: info.Email, Name: name, AvatarURL: info.Picture, Role: role})
		if err != nil {
			return Result{}, fmt.Errorf("%w: %w", ErrCreateUser, err)
		}
	}
	name := "iOS App"
	if client == "web" {
		name = "Web App"
	}
	existing, err := service.tokens.FindByUserIDAndName(ctx, user.ID, name)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrReplaceToken, err)
	}
	for _, token := range existing {
		if err := service.tokens.Delete(ctx, token.ID); err != nil {
			return Result{}, fmt.Errorf("%w: %w", ErrReplaceToken, err)
		}
	}
	secret, err := service.secret()
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrGenerateToken, err)
	}
	if _, err := service.tokens.Create(ctx, models.APIToken{Name: name, TokenHash: service.hash(secret), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID}); err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrCreateToken, err)
	}
	return Result{Token: secret, User: user}, nil
}
