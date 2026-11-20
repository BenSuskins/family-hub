package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/commute"
)

type displayTestPrefs struct {
	value app.DisplayPreferences
	err   error
}

func (fake displayTestPrefs) Preferences(context.Context, string) (app.DisplayPreferences, error) {
	return fake.value, fake.err
}

type displayTestZone struct{}

func (displayTestZone) Timezone(context.Context) app.DisplayTimezone {
	return app.DisplayTimezone{Value: "UTC"}
}

type displayTestRail struct{}

func (displayTestRail) Fetch(context.Context, string, string, int) (app.DisplayBoard, error) {
	return app.DisplayBoard{}, errors.New("rail down")
}
func TestDisplayCommuteOperationResponses(t *testing.T) {
	cases := []struct {
		name   string
		prefs  displayTestPrefs
		status int
		want   string
	}{{"setup", displayTestPrefs{}, 200, "setup_required"}, {"lookup error", displayTestPrefs{err: errors.New("missing")}, 500, ""}, {"provider failure", displayTestPrefs{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}}}, 200, "unavailable"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := app.NewDisplayCommute(tc.prefs, displayTestZone{}, displayTestRail{}, func() time.Time { return time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC) })
			handler := NewDisplayCommuteHandler(service)
			recorder := httptest.NewRecorder()
			handler.Trains(recorder, httptest.NewRequest(http.MethodGet, "/api/display/trains", nil))
			if recorder.Code != tc.status || recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status=%d headers=%v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
			}
			if tc.want != "" {
				var data map[string]any
				if err := json.Unmarshal(recorder.Body.Bytes(), &data); err != nil {
					t.Fatal(err)
				}
				if data["status"] != tc.want {
					t.Fatalf("body=%s", recorder.Body.String())
				}
				if _, ok := data["timezone"]; ok != (tc.want != "setup_required") {
					t.Fatalf("timezone body=%s", recorder.Body.String())
				}
			}
		})
	}
}

func TestDisplayCommuteRecordsEachRailFailure(t *testing.T) {
	schedule := app.DisplaySchedule{Enabled: true, Weekdays: []int{2}, Start: "07:00", End: "09:00"}
	service := app.NewDisplayCommute(displayTestPrefs{value: app.DisplayPreferences{Home: "KEL", Work: "LST", Count: 3, ToWork: schedule, ToHome: schedule}}, displayTestZone{}, displayTestRail{}, func() time.Time { return time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC) })
	metrics := observability.NewHTTPMetrics()
	handler := metrics.Middleware(http.HandlerFunc(NewDisplayCommuteHandler(service).Trains))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/display/trains", nil))
	recorder := httptest.NewRecorder()
	metrics.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(recorder.Body.String(), `familyhub_dependency_failures_total{dependency="rail",route="unmatched"} 2`) {
		t.Fatalf("metrics %s", recorder.Body.String())
	}
}

func TestDisplayCommuteBoardResponseMatchesExistingJSON(t *testing.T) {
	platform := "4"
	departure := "08:12"
	generated := time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC)
	original := models.CommuteBoard{Origin: models.CommuteStation{Code: "KEL"}, Destination: models.CommuteStation{Code: "LST"}, GeneratedAt: generated, FetchedAt: generated, Timezone: "Europe/London", Trains: []models.CommuteTrain{{ServiceID: "one", Operator: "rail", Platform: &platform, Status: "on_time", ScheduledDeparture: &departure}}}
	converted := app.DisplayBoard{Origin: "KEL", Destination: "LST", GeneratedAt: generated, FetchedAt: generated, Timezone: "Europe/London", Trains: []app.DisplayTrain{{ServiceID: "one", Operator: "rail", Platform: &platform, Status: "on_time", ScheduledDeparture: &departure}}}
	expected, _ := json.Marshal(original)
	actual, _ := json.Marshal(displayBoardResponseFor(converted))
	if string(actual) != string(expected) {
		t.Fatalf("expected %s got %s", expected, actual)
	}
}
