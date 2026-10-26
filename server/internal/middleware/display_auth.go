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

// RequireDisplay accepts only revocable display-scoped bearer credentials.
func RequireDisplay(tokenRepo repository.APITokenRepository, userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, "Bearer ") {
				httpresponse.Error(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			token, err := tokenRepo.FindByTokenHash(r.Context(), repository.HashToken(strings.TrimPrefix(authorization, "Bearer ")))
			if err != nil || token.Scope != models.TokenScopeDisplay || (token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now())) {
				httpresponse.Error(w, http.StatusUnauthorized, "invalid display credential")
				return
			}
			user, err := userRepo.FindByID(r.Context(), token.CreatedByUserID)
			if err != nil {
				httpresponse.Error(w, http.StatusUnauthorized, "invalid display credential")
				return
			}
			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
