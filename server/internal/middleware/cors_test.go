package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/middleware"
)

func TestBrowserCORS(t *testing.T) {
	handler := middleware.BrowserCORS("https://family.example.com")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	for _, test := range []struct {
		name, origin string
		method       string
		want         int
		allowOrigin  string
	}{
		{"allowed preflight", "https://family.example.com", http.MethodOptions, http.StatusNoContent, "https://family.example.com"},
		{"unapproved origin", "https://attacker.example", http.MethodOptions, http.StatusForbidden, ""},
		{"no origin remains usable", "", http.MethodGet, http.StatusOK, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, "/api/me", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("want %d, got %d", test.want, response.Code)
			}
			if got := response.Header().Get("Access-Control-Allow-Origin"); got != test.allowOrigin {
				t.Fatalf("want allow origin %q, got %q", test.allowOrigin, got)
			}
		})
	}
}
