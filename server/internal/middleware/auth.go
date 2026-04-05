package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/templates/layouts"
)

type contextKey string

const UserContextKey contextKey = "user"

// RequireUser authenticates the request using either a Bearer API token or a
// session cookie, populates the user into the request context, and delegates
// to the next handler. On failure:
//   - if the request carried a Bearer header, responds 401 Unauthorized
//   - otherwise redirects to /login (browser UX)
func RequireUser(
	authService *services.AuthService,
	tokenRepo repository.APITokenRepository,
	userRepo repository.UserRepository,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				user, ok := authenticateBearer(r.Context(), authHeader, tokenRepo, userRepo)
				if !ok {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), UserContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			user, err := authService.GetCurrentUser(r)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusFound)
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
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUser(ctx context.Context) models.User {
	user, _ := ctx.Value(UserContextKey).(models.User)
	return user
}

func InjectFamilyName(settingsRepo repository.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			familyName, err := settingsRepo.Get(r.Context(), repository.SettingsKeyFamilyName)
			if err != nil {
				slog.Debug("loading family name setting", "error", err)
			}
			ctx := layouts.WithFamilyName(r.Context(), familyName)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
