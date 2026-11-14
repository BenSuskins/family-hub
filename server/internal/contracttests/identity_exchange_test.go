package contracttests

import (
	"context"
	"fmt"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type identityProviderFake struct{ subject string }

func (p *identityProviderFake) UserInfo(context.Context, string) (app.UserInfo, error) {
	return app.UserInfo{Subject: p.subject, Email: p.subject + "@example.com", Name: p.subject}, nil
}

type identityUsersFake struct{ users map[string]models.User }

func (s *identityUsersFake) Lookup(_ context.Context, subject string) (models.User, bool, error) {
	user, ok := s.users[subject]
	return user, ok, nil
}
func (s *identityUsersFake) Count(context.Context) (int, error) { return len(s.users), nil }
func (s *identityUsersFake) Create(_ context.Context, user models.User) (models.User, error) {
	user.ID = user.OIDCSubject
	s.users[user.OIDCSubject] = user
	return user, nil
}

type identityTokensFake struct {
	tokens []models.APIToken
	next   int
}

func (s *identityTokensFake) FindByUserIDAndName(_ context.Context, user, name string) ([]models.APIToken, error) {
	var result []models.APIToken
	for _, token := range s.tokens {
		if token.CreatedByUserID == user && token.Name == name {
			result = append(result, token)
		}
	}
	return result, nil
}
func (s *identityTokensFake) Delete(_ context.Context, id string) error {
	for i, token := range s.tokens {
		if token.ID == id {
			s.tokens = append(s.tokens[:i], s.tokens[i+1:]...)
			break
		}
	}
	return nil
}
func (s *identityTokensFake) Create(_ context.Context, token models.APIToken) (models.APIToken, error) {
	s.next++
	token.ID = fmt.Sprint(s.next)
	s.tokens = append(s.tokens, token)
	return token, nil
}
func TestIdentityExchangeContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.Users, app.Tokens){
		"fake": func(*testing.T) (app.Users, app.Tokens) {
			return &identityUsersFake{users: map[string]models.User{}}, &identityTokensFake{}
		},
		"sqlite": func(t *testing.T) (app.Users, app.Tokens) {
			db := testutil.NewTestDatabase(t)
			return repository.NewUserRepository(db), repository.NewAPITokenRepository(db)
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			users, tokens := factory(t)
			provider := &identityProviderFake{subject: "first"}
			next := 0
			service := app.New(provider, users, tokens, func() (string, error) { next++; return fmt.Sprintf("secret-%d", next), nil }, repository.HashToken)
			first, err := service.Exchange(context.Background(), "oidc", "")
			if err != nil || first.User.Role != models.RoleAdmin {
				t.Fatalf("first=%+v err=%v", first, err)
			}
			web, err := service.Exchange(context.Background(), "oidc", "web")
			if err != nil || web.User.ID != first.User.ID {
				t.Fatalf("web=%+v err=%v", web, err)
			}
			_, err = service.Exchange(context.Background(), "oidc", "web")
			if err != nil {
				t.Fatal(err)
			}
			ios, err := tokens.FindByUserIDAndName(context.Background(), first.User.ID, "iOS App")
			if err != nil || len(ios) != 1 || ios[0].TokenHash != repository.HashToken(first.Token) {
				t.Fatalf("ios=%+v err=%v", ios, err)
			}
			webTokens, err := tokens.FindByUserIDAndName(context.Background(), first.User.ID, "Web App")
			if err != nil || len(webTokens) != 1 || webTokens[0].TokenHash == repository.HashToken(web.Token) {
				t.Fatalf("web=%+v err=%v", webTokens, err)
			}
			provider.subject = "second"
			second, err := service.Exchange(context.Background(), "oidc", "")
			if err != nil || second.User.Role != models.RoleMember {
				t.Fatalf("second=%+v err=%v", second, err)
			}
		})
	}
}
