package middleware

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/repository"
)

func RequireOnboarding(settingsRepo repository.SettingsRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			complete, err := settingsRepo.Get(r.Context(), "onboarding_complete")
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				slog.Error("loading onboarding_complete setting", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
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
