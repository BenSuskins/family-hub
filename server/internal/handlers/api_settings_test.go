package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestMealWeekSettings(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	handler := &APIHandler{settingsRepo: repository.NewSettingsRepository(database)}
	read := func() map[string]string {
		t.Helper()
		response := httptest.NewRecorder()
		handler.GetSettings(response, httptest.NewRequest(http.MethodGet, "/api/settings", nil))
		var settings map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &settings); err != nil {
			t.Fatal(err)
		}
		return settings
	}
	patch := func(body string, status int) {
		t.Helper()
		response := httptest.NewRecorder()
		handler.PatchSettings(response, httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(body)))
		if response.Code != status {
			t.Fatalf("PATCH %s: got %d: %s", body, response.Code, response.Body.String())
		}
	}
	if got := read()["meal_week_start"]; got != "friday" {
		t.Fatalf("default weekday = %q", got)
	}
	patch(`{"family_name":"Our family"}`, 204)
	for _, day := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
		patch(`{"meal_week_start":"`+day+`"}`, 204)
		settings := read()
		if settings["meal_week_start"] != day || settings["family_name"] != "Our family" {
			t.Fatalf("settings = %v", settings)
		}
	}
	for _, body := range []string{`{}`, `{"meal_week_start":"Friday"}`, `{"meal_week_start":""}`, `{"meal_week_start":null}`, `{"family_name":""}`, `{"family_name":"Changed","meal_week_start":"bad"}`} {
		patch(body, 400)
	}
	if read()["family_name"] != "Our family" {
		t.Fatal("invalid request changed family name")
	}
	patch(`{"family_name":"New name","meal_week_start":"monday"}`, 204)
	if read()["meal_week_start"] != "monday" {
		t.Fatal("weekday did not persist")
	}
}
