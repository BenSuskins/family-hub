package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
)

type AuthService struct {
	oauthConfig  *oauth2.Config
	oidcVerifier *oidc.IDTokenVerifier
	secureCookie *securecookie.SecureCookie
	userRepo     repository.UserRepository
}

type SessionData struct {
	UserID string `json:"user_id"`
}

func NewAuthService(ctx context.Context, cfg config.Config, userRepo repository.UserRepository) (*AuthService, error) {
	if cfg.OIDCIssuer == "" {
		slog.Warn("OIDC not configured, auth will be disabled")
		return &AuthService{
			secureCookie: securecookie.New([]byte(cfg.SessionSecret), nil),
			userRepo:     userRepo,
		}, nil
	}

	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("creating OIDC provider: %w", err)
	}

	oauthConfig := &oauth2.Config{
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURL:  cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})

	return &AuthService{
		oauthConfig:  oauthConfig,
		oidcVerifier: verifier,
		secureCookie: securecookie.New([]byte(cfg.SessionSecret), nil),
		userRepo:     userRepo,
	}, nil
}

func (service *AuthService) OIDCConfigured() bool {
	return service.oauthConfig != nil
}

func (service *AuthService) LoginURL(state string) string {
	if service.oauthConfig == nil {
		return ""
	}
	return service.oauthConfig.AuthCodeURL(state)
}

func (service *AuthService) GenerateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (service *AuthService) HandleCallback(ctx context.Context, code string) (models.User, error) {
	if service.oauthConfig == nil {
		return models.User{}, errors.New("OIDC not configured")
	}

	token, err := service.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return models.User{}, fmt.Errorf("exchanging code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return models.User{}, errors.New("no id_token in response")
	}

	idToken, err := service.oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return models.User{}, fmt.Errorf("verifying id token: %w", err)
	}

	var claims struct {
		Subject          string `json:"sub"`
		Email            string `json:"email"`
		Name             string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture          string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return models.User{}, fmt.Errorf("parsing claims: %w", err)
	}

	displayName := claims.Name
	if displayName == "" {
		displayName = claims.PreferredUsername
	}
	if displayName == "" {
		displayName = claims.Email
	}

	return service.provisionUser(ctx, claims.Subject, claims.Email, displayName, claims.Picture)
}

func (service *AuthService) provisionUser(ctx context.Context, subject, email, name, avatarURL string) (models.User, error) {
	existingUser, err := service.userRepo.FindByOIDCSubject(ctx, subject)
	if err == nil {
		if err := service.userRepo.UpdateProfile(ctx, existingUser.ID, name, email, avatarURL); err != nil {
			slog.Warn("failed to update user profile on login", "error", err)
		}
		existingUser.Name = name
		existingUser.Email = email
		existingUser.AvatarURL = avatarURL
		return existingUser, nil
	}
	if !errors.Is(err, sql.ErrNoRows) && !isNotFound(err) {
		return models.User{}, fmt.Errorf("looking up user: %w", err)
	}

	userCount, err := service.userRepo.Count(ctx)
	if err != nil {
		return models.User{}, fmt.Errorf("counting users: %w", err)
	}

	role := models.RoleMember
	if userCount == 0 {
		role = models.RoleAdmin
	}

	newUser := models.User{
		OIDCSubject: subject,
		Email:       email,
		Name:        name,
		AvatarURL:   avatarURL,
		Role:        role,
	}

	created, err := service.userRepo.Create(ctx, newUser)
	if err != nil {
		return models.User{}, fmt.Errorf("creating user: %w", err)
	}

	slog.Info("provisioned new user", "id", created.ID, "name", created.Name, "role", created.Role)
	return created, nil
}

func (service *AuthService) SetSession(w http.ResponseWriter, userID string) error {
	data := SessionData{UserID: userID}
	encoded, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	value, err := service.secureCookie.Encode("session", string(encoded))
	if err != nil {
		return fmt.Errorf("encoding session cookie: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 30,
	})
	return nil
}

func (service *AuthService) GetSession(r *http.Request) (SessionData, error) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return SessionData{}, fmt.Errorf("no session cookie: %w", err)
	}

	var decoded string
	if err := service.secureCookie.Decode("session", cookie.Value, &decoded); err != nil {
		return SessionData{}, fmt.Errorf("decoding session cookie: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal([]byte(decoded), &session); err != nil {
		return SessionData{}, fmt.Errorf("unmarshaling session: %w", err)
	}
	return session, nil
}

func (service *AuthService) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func (service *AuthService) GetCurrentUser(r *http.Request) (models.User, error) {
	session, err := service.GetSession(r)
	if err != nil {
		return models.User{}, err
	}

	user, err := service.userRepo.FindByID(r.Context(), session.UserID)
	if err != nil {
		return models.User{}, fmt.Errorf("finding user: %w", err)
	}
	return user, nil
}

func isNotFound(err error) bool {
	return err != nil && (errors.Is(err, sql.ErrNoRows) ||
		fmt.Sprintf("%v", err) == "finding user by oidc subject: sql: no rows in result set")
}
