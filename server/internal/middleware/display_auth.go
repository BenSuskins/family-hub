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
func RequireDisplay(tokenRepo repository.APITokenRepository) func(http.Handler) http.Handler {
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
			// Display credentials belong to the paired household device. They must
			// remain valid when the member who approved pairing leaves the household.
			ctx := context.WithValue(r.Context(), DisplayTokenIDContextKey, token.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
