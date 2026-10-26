package middleware

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/httpresponse"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/repository"
)

type contextKey string

const UserContextKey contextKey = "user"
const DisplayTokenIDContextKey contextKey = "display_token_id"

var errInvalidBearerToken = errors.New("invalid bearer token")

func GetDisplayTokenID(ctx context.Context) string {
	id, _ := ctx.Value(DisplayTokenIDContextKey).(string)
	return id
}

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

			user, err := authenticateBearer(r.Context(), authHeader, tokenRepo, userRepo)
			if errors.Is(err, errInvalidBearerToken) {
				httpresponse.Error(w, http.StatusUnauthorized, "invalid bearer token")
				return
			}
			if err != nil {
				slog.Error("API authentication lookup failed", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusInternalServerError, "error", err)
				httpresponse.Error(w, http.StatusInternalServerError, "authentication unavailable")
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
) (models.User, error) {
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	tokenHash := repository.HashToken(tokenString)

	token, err := tokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, errInvalidBearerToken
		}
		return models.User{}, fmt.Errorf("looking up API token: %w", err)
	}
	if token.Scope != models.TokenScopeAPI {
		return models.User{}, errInvalidBearerToken
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return models.User{}, errInvalidBearerToken
	}

	user, err := userRepo.FindByID(ctx, token.CreatedByUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, errInvalidBearerToken
		}
		return models.User{}, fmt.Errorf("looking up API token owner: %w", err)
	}
	return user, nil
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
