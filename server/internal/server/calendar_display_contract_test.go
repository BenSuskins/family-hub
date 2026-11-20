package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

func TestOpenAPICalendarReadAndAccessContract(t *testing.T) {
	fixture := newContractFixture(t)
	for _, route := range []struct{ path, token, wrong string }{{"/api/calendar", "member", "display"}, {"/api/display/calendar", "display", "member"}} {
		for _, token := range []string{"", route.wrong} {
			response := fixture.request(t, http.MethodGet, route.path+"?view=day&date=2026-10-25", token, "", true)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("%s wrong scope: %d %s", route.path, response.Code, response.Body.String())
			}
		}
		for _, query := range []string{"?view=day&date=not-a-date", "?view=week&date=not-a-date", "?view=month&month=not-a-month"} {
			response := fixture.request(t, http.MethodGet, route.path+query, route.token, "", true)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("%s%s: %d %s", route.path, query, response.Code, response.Body.String())
			}
		}
		response := fixture.request(t, http.MethodGet, route.path+"?view=day&date=2026-10-25", route.token, "", true)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", route.path, response.Code, response.Body.String())
		}
		var body struct {
			Chores       []json.RawMessage `json:"chores"`
			Events       []json.RawMessage `json:"events"`
			Meals        []json.RawMessage `json:"meals"`
			Availability map[string]string `json:"availability"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Chores == nil || body.Events == nil || body.Meals == nil || !reflect.DeepEqual(body.Availability, map[string]string{"events": "available", "meals": "available"}) {
			t.Fatalf("calendar: %s", response.Body.String())
		}
	}
}

func TestOpenAPIDisplayTrainResponseContract(t *testing.T) {
	fixture := newContractFixture(t)
	for _, token := range []string{"", "member"} {
		response := fixture.request(t, http.MethodGet, "/api/display/trains", token, "", true)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("wrong scope: %d %s", response.Code, response.Body.String())
		}
	}
	ctx := context.Background()
	token, err := repository.NewAPITokenRepository(fixture.database).FindByTokenHash(ctx, repository.HashToken("display"))
	if err != nil {
		t.Fatal(err)
	}
	pairing := repository.NewDisplayPairingRepository(fixture.database)
	if err := pairing.CreateChallenge(ctx, "display-contract", "secret", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	preferences := models.DefaultDisplayCommutePreferences()
	encoded, err := json.Marshal(preferences)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.database.ExecContext(ctx, "INSERT INTO paired_displays(id,name,api_token_id,commute_preferences) VALUES(?,?,?,?)", "display-contract", "Kitchen", token.ID, string(encoded)); err != nil {
		t.Fatal(err)
	}
	response := fixture.request(t, http.MethodGet, "/api/display/trains", "display", "", true)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"boards":[],"status":"setup_required"}` {
		t.Fatalf("setup: %d %s", response.Code, response.Body.String())
	}
	preferences.HomeStationCode = "KEL"
	preferences.WorkStationCode = "LST"
	preferences.ToWork.Enabled = false
	preferences.ToHome.Enabled = false
	encoded, err = json.Marshal(preferences)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.database.ExecContext(ctx, "UPDATE paired_displays SET commute_preferences=? WHERE id=?", string(encoded), "display-contract"); err != nil {
		t.Fatal(err)
	}
	if err = repository.NewSettingsRepository(fixture.database).Set(ctx, "home_info", `{"timezone":"invalid/timezone"}`); err != nil {
		t.Fatal(err)
	}
	response = fixture.request(t, http.MethodGet, "/api/display/trains", "display", "", true)
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"boards":[],"status":"outside_schedule","timezone":"UTC"}` {
		t.Fatalf("outside schedule: %d %s", response.Code, response.Body.String())
	}
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "/LST/to/KEL/") {
			writer.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":[]}`))
	}))
	defer provider.Close()
	fixture.application = New(fixture.database, config.Config{HuxleyBaseURL: provider.URL, HuxleyAccessToken: "test-provider-token"})
	allDay := models.DisplayCommuteSchedule{Enabled: true, Weekdays: []int{1, 2, 3, 4, 5, 6, 7}, Start: "00:00", End: "00:00"}
	preferences.ToWork = allDay
	preferences.ToHome = allDay
	encoded, err = json.Marshal(preferences)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.database.ExecContext(ctx, "UPDATE paired_displays SET commute_preferences=? WHERE id=?", string(encoded), "display-contract"); err != nil {
		t.Fatal(err)
	}
	response = fixture.request(t, http.MethodGet, "/api/display/trains", "display", "", true)
	var trains struct {
		Status string            `json:"status"`
		Boards []json.RawMessage `json:"boards"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &trains); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || trains.Status != "partial" || len(trains.Boards) != 1 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("partial: %d %s", response.Code, response.Body.String())
	}
}
