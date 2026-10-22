package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/httpresponse"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireUser authenticates the request using a Bearer API token, populates
// the user into the request context, and delegates to the next handler.
func RequireUser(
	tokenRepo repository.APITokenRepository,
	userRepo repository.UserRepository,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				httpresponse.Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}

			user, ok := authenticateBearer(r.Context(), authHeader, tokenRepo, userRepo)
			if !ok {
				httpresponse.Error(w, http.StatusUnauthorized, "invalid bearer token")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authenticateBearer(
	ctx context.Context,
	authHeader string,
	tokenRepo repository.APITokenRepository,
	userRepo repository.UserRepository,
) (models.User, bool) {
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenHash := repository.HashToken(tokenString)

	token, err := tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return models.User{}, false
	}
	if token.Scope != models.TokenScopeAPI {
		return models.User{}, false
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return models.User{}, false
	}

	user, err := userRepo.FindByID(ctx, token.CreatedByUserID)
	if err != nil {
		return models.User{}, false
	}
	return user, true
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user.Role != models.RoleAdmin {
			httpresponse.Error(w, http.StatusForbidden, "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUser(ctx context.Context) models.User {
	user, _ := ctx.Value(UserContextKey).(models.User)
	return user
}
