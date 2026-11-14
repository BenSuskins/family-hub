package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/identity"
)

const maxOIDCResponseBytes int64 = 1 << 20

type OIDCProvider struct {
	url    string
	client *http.Client
}

func NewOIDCProvider(url string, client *http.Client) *OIDCProvider {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &OIDCProvider{url: url, client: client}
}
func (provider *OIDCProvider) UserInfo(ctx context.Context, token string) (app.UserInfo, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.url, nil)
	if err != nil {
		return app.UserInfo{}, fmt.Errorf("%w: %w", app.ErrProviderRequest, err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := provider.client.Do(request)
	if err != nil {
		return app.UserInfo{}, fmt.Errorf("%w: %w", app.ErrProviderUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		if response.StatusCode >= 500 || response.StatusCode == 429 {
			return app.UserInfo{}, fmt.Errorf("%w: HTTP %d", app.ErrProviderUnavailable, response.StatusCode)
		}
		return app.UserInfo{}, app.ErrInvalidToken
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxOIDCResponseBytes+1))
	if err != nil {
		return app.UserInfo{}, fmt.Errorf("%w: %w", app.ErrProviderResponse, err)
	}
	if int64(len(data)) > maxOIDCResponseBytes {
		return app.UserInfo{}, fmt.Errorf("%w: %w", app.ErrProviderResponse, errors.New("OIDC user-info response exceeds size limit"))
	}
	var body struct {
		Subject           string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		return app.UserInfo{}, fmt.Errorf("%w: %w", app.ErrProviderResponse, err)
	}
	if body.Subject == "" {
		return app.UserInfo{}, fmt.Errorf("%w: missing subject", app.ErrProviderResponse)
	}
	return app.UserInfo{Subject: body.Subject, Email: body.Email, Name: body.Name, PreferredUsername: body.PreferredUsername, Picture: body.Picture}, nil
}
