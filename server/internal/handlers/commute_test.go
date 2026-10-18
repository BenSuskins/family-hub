package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/services"
)

func TestCommuteAPIUsesDefaultsAndDirection(t *testing.T) {
	for _, test := range []struct{ query, path, direction string }{
		{"", "/departures/KEL/to/LST/5", "to_work"},
		{"?direction=to_home&count=1", "/departures/LST/to/KEL/1", "to_home"},
		{"?direction=to_work&count=10", "/departures/KEL/to/LST/10", "to_work"},
	} {
		t.Run(test.direction+test.query, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.path {
					t.Errorf("unexpected route: %s", request.URL.Path)
				}
				writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":null}`))
			}))
			t.Cleanup(provider.Close)
			client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
			handler := NewCommuteHandler(services.NewCommuteService(client, nil))
			recorder := httptest.NewRecorder()
			handler.ListTrains(recorder, httptest.NewRequest(http.MethodGet, "/api/commute/trains"+test.query, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected 200: %d %s", recorder.Code, recorder.Body.String())
			}
			var board models.CommuteBoard
			if err := json.Unmarshal(recorder.Body.Bytes(), &board); err != nil {
				t.Fatal(err)
			}
			if board.Direction != test.direction || board.Trains == nil || board.FetchedAt.IsZero() || board.GeneratedAt.IsZero() {
				t.Fatalf("unexpected board: %s", recorder.Body.String())
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("HTTP caches must not retain train times")
			}
		})
	}
}

func TestCommuteAPIRejectsInvalidParameters(t *testing.T) {
	handler := NewCommuteHandler(nil)
	for _, query := range []string{"direction=outbound", "direction=", "count=", "count=0", "count=-1", "count=11", "count=abc", "count=1.5", "count=1&count=2", "direction=to_work&direction=to_home"} {
		t.Run(query, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ListTrains(recorder, httptest.NewRequest(http.MethodGet, "/api/commute/trains?"+query, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", recorder.Code)
			}
		})
	}
}

func TestCommuteAPICachesThenReturnsErrorAfterExpiry(t *testing.T) {
	now := time.Date(2026, 10, 18, 7, 0, 0, 0, time.UTC)
	var available atomic.Bool
	available.Store(true)
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !available.Load() {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":[]}`))
	}))
	t.Cleanup(provider.Close)
	client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
	handler := NewCommuteHandler(services.NewCommuteService(client, func() time.Time { return now }))
	request := func(query string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		handler.ListTrains(recorder, httptest.NewRequest(http.MethodGet, "/api/commute/trains"+query, nil))
		return recorder
	}
	first := request("")
	if first.Code != http.StatusOK {
		t.Fatalf("initial request: %s", first.Body.String())
	}
	available.Store(false)
	now = now.Add(29 * time.Second)
	cached := request("")
	if cached.Code != http.StatusOK || cached.Body.String() != first.Body.String() {
		t.Fatal("fresh cache is not reused")
	}
	if separateCount := request("?count=1"); separateCount.Code != http.StatusBadGateway {
		t.Fatal("cache reused for a different count")
	}
	if separateDirection := request("?direction=to_home"); separateDirection.Code != http.StatusBadGateway {
		t.Fatal("cache reused for a different direction")
	}
	now = now.Add(time.Second)
	if expired := request(""); expired.Code != http.StatusBadGateway {
		t.Fatalf("expired cache returned %d", expired.Code)
	}
	available.Store(true)
	if recovered := request(""); recovered.Code != http.StatusOK || recovered.Body.String() == first.Body.String() {
		t.Fatal("provider recovery did not refresh the cache")
	}
}

func TestCommuteAPIReturnsProviderErrors(t *testing.T) {
	for _, test := range []struct {
		name           string
		token          string
		body           string
		providerStatus int
		expectedStatus int
		timeout        bool
	}{
		{"missing configuration", "", "", http.StatusOK, http.StatusServiceUnavailable, false},
		{"provider error", "private-test-token", "private-test-token", http.StatusForbidden, http.StatusBadGateway, false},
		{"malformed response", "private-test-token", "private-test-token", http.StatusOK, http.StatusBadGateway, false},
		{"provider timeout", "private-test-token", "", http.StatusOK, http.StatusGatewayTimeout, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if test.timeout {
					<-request.Context().Done()
					return
				}
				writer.WriteHeader(test.providerStatus)
				writer.Write([]byte(test.body))
			}))
			t.Cleanup(provider.Close)
			httpClient := provider.Client()
			httpClient.Timeout = 20 * time.Millisecond
			client := services.NewHuxleyClient(provider.URL, test.token, httpClient)
			handler := NewCommuteHandler(services.NewCommuteService(client, nil))
			recorder := httptest.NewRecorder()
			handler.ListTrains(recorder, httptest.NewRequest(http.MethodGet, "/api/commute/trains", nil))
			if recorder.Code != test.expectedStatus || strings.Contains(recorder.Body.String(), "private-test-token") {
				t.Fatalf("expected safe error %d, got %d %s", test.expectedStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}
