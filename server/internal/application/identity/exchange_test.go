package identity_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/models"
)

type providerFake struct{ info app.UserInfo }

func (p providerFake) UserInfo(context.Context, string) (app.UserInfo, error) { return p.info, nil }

type usersFake struct {
	user   models.User
	exists bool
}

func (u *usersFake) Lookup(context.Context, string) (models.User, bool, error) {
	return u.user, u.exists, nil
}
func (u *usersFake) Count(context.Context) (int, error) {
	if u.exists {
		return 1, nil
	}
	return 0, nil
}
func (u *usersFake) Create(_ context.Context, user models.User) (models.User, error) {
	user.ID = "user"
	u.user = user
	u.exists = true
	return user, nil
}

type tokensFake struct {
	tokens    []models.APIToken
	createErr error
}

func (s *tokensFake) FindByUserIDAndName(_ context.Context, user, name string) ([]models.APIToken, error) {
	var result []models.APIToken
	for _, token := range s.tokens {
		if token.CreatedByUserID == user && token.Name == name {
			result = append(result, token)
		}
	}
	return result, nil
}
func (s *tokensFake) Delete(_ context.Context, id string) error {
	for i, token := range s.tokens {
		if token.ID == id {
			s.tokens = append(s.tokens[:i], s.tokens[i+1:]...)
			break
		}
	}
	return nil
}
func (s *tokensFake) Create(_ context.Context, token models.APIToken) (models.APIToken, error) {
	if s.createErr != nil {
		return models.APIToken{}, s.createErr
	}
	token.ID = "new"
	s.tokens = append(s.tokens, token)
	return token, nil
}
func TestFirstExchangeCreatesAdminAndReplacesOnlyWebToken(t *testing.T) {
	users := &usersFake{}
	tokens := &tokensFake{tokens: []models.APIToken{{ID: "ios", Name: "iOS App", CreatedByUserID: "user"}, {ID: "web", Name: "Web App", CreatedByUserID: "user"}}}
	service := app.New(providerFake{info: app.UserInfo{Subject: "subject", Email: "family@example.com", PreferredUsername: "Family"}}, users, tokens, func() (string, error) { return "secret", nil }, func(string) string { return "hash" })
	result, err := service.Exchange(context.Background(), "oidc", "web")
	if err != nil || result.User.Role != models.RoleAdmin || result.User.Name != "Family" || result.Token != "secret" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(tokens.tokens) != 2 || tokens.tokens[0].ID != "ios" || tokens.tokens[1].Name != "Web App" || tokens.tokens[1].TokenHash != "hash" {
		t.Fatalf("tokens=%+v", tokens.tokens)
	}
}

func TestFailedReplacementReturnsNoCredential(t *testing.T) {
	users := &usersFake{exists: true, user: models.User{ID: "user"}}
	tokens := &tokensFake{tokens: []models.APIToken{{ID: "old", Name: "Web App", CreatedByUserID: "user"}}, createErr: errors.New("storage failed")}
	service := app.New(providerFake{}, users, tokens, func() (string, error) { return "secret", nil }, func(string) string { return "hash" })
	result, err := service.Exchange(context.Background(), "oidc", "web")
	if !errors.Is(err, app.ErrCreateToken) || result.Token != "" || len(tokens.tokens) != 0 {
		t.Fatalf("result=%+v tokens=%+v err=%v", result, tokens.tokens, err)
	}
}
