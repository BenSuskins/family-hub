package handlers

import (
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/services"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (handler *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if !handler.authService.OIDCConfigured() {
		http.Error(w, "OIDC not configured", http.StatusServiceUnavailable)
		return
	}

	state, err := handler.authService.GenerateState()
	if err != nil {
		slog.Error("generating state", "error", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   300,
	})

	http.Redirect(w, r, handler.authService.LoginURL(state), http.StatusFound)
}

func (handler *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "Missing state cookie", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "oauth_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code", http.StatusBadRequest)
		return
	}

	user, err := handler.authService.HandleCallback(r.Context(), code)
	if err != nil {
		slog.Error("handling callback", "error", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	if err := handler.authService.SetSession(w, user.ID); err != nil {
		slog.Error("setting session", "error", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (handler *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	handler.authService.ClearSession(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}
