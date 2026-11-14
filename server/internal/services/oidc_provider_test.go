package services_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/services"
)

func TestOIDCProviderClassifiesResponsesAndLimitsBody(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"valid", 200, `{"sub":"subject","email":"a@example.com"}`, nil},
		{"unauthorized", 401, "", app.ErrInvalidToken},
		{"rate limited", 429, "", app.ErrProviderUnavailable},
		{"server error", 500, "", app.ErrProviderUnavailable},
		{"missing subject", 200, `{}`, app.ErrProviderResponse},
		{"oversized", 200, `{"sub":"subject","padding":"` + strings.Repeat("x", 1<<20) + `"}`, app.ErrProviderResponse},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer oidc" {
					t.Error("missing bearer token")
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			provider := services.NewOIDCProvider(server.URL, server.Client())
			info, err := provider.UserInfo(context.Background(), "oidc")
			if !errors.Is(err, test.want) || test.want == nil && info.Subject != "subject" {
				t.Fatalf("info=%+v err=%v", info, err)
			}
		})
	}
}
func TestOIDCProviderHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	provider := services.NewOIDCProvider(server.URL, &http.Client{Timeout: 10 * time.Millisecond})
	if _, err := provider.UserInfo(context.Background(), "oidc"); !errors.Is(err, app.ErrProviderUnavailable) {
		t.Fatalf("error=%v", err)
	}
}
