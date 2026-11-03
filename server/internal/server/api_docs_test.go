package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestDeveloperDocumentationRequiresOptIn(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		application := New(testutil.NewTestDatabase(t), config.Config{APIDocsEnabled: enabled})
		for _, endpoint := range []struct{ path, contentType, contains string }{
			{"/api/openapi.json", "application/json", `"openapi": "3.1.0"`},
			{"/api/docs", "text/html", "SwaggerUIBundle"},
			{"/api/docs/swagger-ui-bundle.js", "text/javascript", "SwaggerUIBundle"},
			{"/api/docs/swagger-ui.css", "text/css", ".swagger-ui"},
		} {
			response := httptest.NewRecorder()
			application.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, endpoint.path, nil))
			if !enabled {
				if response.Code != http.StatusNotFound || response.Header().Get("Content-Type") != "application/json" {
					t.Fatalf("disabled %s: %d %s", endpoint.path, response.Code, response.Body.String())
				}
				continue
			}
			if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), endpoint.contentType) || !strings.Contains(response.Body.String(), endpoint.contains) {
				t.Fatalf("enabled %s: status=%d type=%s", endpoint.path, response.Code, response.Header().Get("Content-Type"))
			}
		}
	}
}
