package services_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/services"
)

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com/recipe",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://example.com/recipe",
			wantErr: false,
		},
		{
			name:    "localhost blocked",
			url:     "http://localhost:8080/secret",
			wantErr: true,
		},
		{
			name:    "127.0.0.1 blocked",
			url:     "http://127.0.0.1/secret",
			wantErr: true,
		},
		{
			name:    "cloud metadata IP blocked",
			url:     "http://169.254.169.254/latest/meta-data/",
			wantErr: true,
		},
		{
			name:    "private 10.x blocked",
			url:     "http://10.0.0.1/internal",
			wantErr: true,
		},
		{
			name:    "private 192.168.x blocked",
			url:     "http://192.168.1.1/internal",
			wantErr: true,
		},
		{
			name:    "private 172.16.x blocked",
			url:     "http://172.16.0.1/internal",
			wantErr: true,
		},
		{
			name:    "FTP scheme blocked",
			url:     "ftp://example.com/file",
			wantErr: true,
		},
		{
			name:    "file scheme blocked",
			url:     "file:///etc/passwd",
			wantErr: true,
		},
		{
			name:    "empty URL blocked",
			url:     "",
			wantErr: true,
		},
		{
			name:    "metadata.google.internal blocked",
			url:     "http://metadata.google.internal/computeMetadata/v1/",
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := services.ValidateExternalURL(testCase.url)
			if testCase.wantErr && err == nil {
				t.Errorf("expected error for URL %q, got nil", testCase.url)
			}
			if !testCase.wantErr && err != nil {
				t.Errorf("unexpected error for URL %q: %v", testCase.url, err)
			}
		})
	}
}

func TestSafeHTTPClient_BlocksRedirectToPrivateIP(t *testing.T) {
	client := services.NewSafeHTTPClient(5 * time.Second)

	internalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("internal secret data"))
	}))
	defer internalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internalServer.URL, http.StatusFound)
	}))
	defer redirectServer.Close()

	_, err := client.Get(redirectServer.URL)
	if err == nil {
		t.Error("expected error when following redirect to private IP, got nil")
	}
}

func TestSafeHTTPClient_BlocksDirectPrivateIP(t *testing.T) {
	client := services.NewSafeHTTPClient(5 * time.Second)

	_, err := client.Get("http://127.0.0.1:1/should-not-connect")
	if err == nil {
		t.Error("expected error when connecting to private IP, got nil")
	}
}

func TestSafeHTTPClient_BlocksTooManyRedirects(t *testing.T) {
	var redirectCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, fmt.Sprintf("/?n=%d", redirectCount), http.StatusFound)
	}))
	defer server.Close()

	client := services.NewSafeHTTPClient(5 * time.Second)
	_, err := client.Get(server.URL)
	if err == nil {
		t.Error("expected error after too many redirects, got nil")
	}
}
