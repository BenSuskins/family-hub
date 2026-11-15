package handlers

import (
	categoryapp "github.com/bensuskins/family-hub/internal/application/categories"
	identityapp "github.com/bensuskins/family-hub/internal/application/identity"
	userapp "github.com/bensuskins/family-hub/internal/application/users"
	"time"
)

type userResponse struct {
	ID, OIDCSubject, Email, Name, AvatarURL string
	Role                                    userapp.Role
	CreatedAt, UpdatedAt                    time.Time
	OnboardedAt                             *time.Time
}

func mapUser(user userapp.User) userResponse {
	return userResponse{user.ID, user.OIDCSubject, user.Email, user.Name, user.AvatarURL, user.Role, user.CreatedAt, user.UpdatedAt, user.OnboardedAt}
}
func mapUsers(users []userapp.User) []userResponse {
	if users == nil {
		return nil
	}
	result := make([]userResponse, len(users))
	for index, user := range users {
		result[index] = mapUser(user)
	}
	return result
}

type categoryResponse struct {
	ID, Name, CreatedByUserID string
	CreatedAt                 time.Time
}

func mapCategory(category categoryapp.Category) categoryResponse {
	return categoryResponse{category.ID, category.Name, category.CreatedByUserID, category.CreatedAt}
}
func mapCategories(categories []categoryapp.Category) []categoryResponse {
	if categories == nil {
		return nil
	}
	result := make([]categoryResponse, len(categories))
	for index, category := range categories {
		result[index] = mapCategory(category)
	}
	return result
}

type tokenResponse struct {
	ID, Name, TokenHash, Scope, CreatedByUserID string
	ExpiresAt                                   *time.Time
	CreatedAt                                   time.Time
}

func mapTokens(tokens []identityapp.TokenRecord) []tokenResponse {
	result := make([]tokenResponse, len(tokens))
	for index, token := range tokens {
		result[index] = tokenResponse{token.ID, token.Name, token.TokenHash, token.Scope, token.CreatedByUserID, token.ExpiresAt, token.CreatedAt}
	}
	return result
}
