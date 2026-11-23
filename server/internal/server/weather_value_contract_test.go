package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestWeatherSetupContract(t *testing.T) {
	fixture := newContractFixture(t)
	for _, endpoint := range []struct{ path, token string }{{"/api/weather", "member"}, {"/api/display/weather", "display"}} {
		response := fixture.request(t, http.MethodGet, endpoint.path, endpoint.token, "", true)
		var value struct {
			Status string            `json:"status"`
			Hours  []json.RawMessage `json:"hours"`
			Days   []json.RawMessage `json:"days"`
			Advice []string          `json:"advice"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil || response.Code != http.StatusOK || value.Status != "setup_required" || value.Hours == nil || value.Days != nil || value.Advice == nil {
			t.Fatalf("%s: %d %s (%v)", endpoint.path, response.Code, response.Body.String(), err)
		}
	}
	unauthorized := fixture.request(t, http.MethodGet, "/api/weather", "invalid", "", false)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized: %d %s", unauthorized.Code, unauthorized.Body.String())
	}
}
