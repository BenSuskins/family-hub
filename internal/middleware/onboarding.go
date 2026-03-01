package middleware

import (
	"net/http"

	"github.com/bensuskins/family-hub/internal/repository"
)

func RequireOnboarding(settingsRepo repository.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			complete, _ := settingsRepo.Get(r.Context(), "onboarding_complete")
			if complete != "true" {
				http.Redirect(w, r, "/setup", http.StatusFound)
				return
			}

			user := GetUser(r.Context())
			if user.OnboardedAt == nil {
				http.Redirect(w, r, "/welcome", http.StatusFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
