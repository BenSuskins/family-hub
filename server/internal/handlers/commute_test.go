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
	"github.com/go-chi/chi/v5"
)

func TestCommuteAPIUsesRouteAndCount(t *testing.T) {
	for _, test := range []struct {
		target, providerPath, origin, destination string
	}{
		{"/api/commute/trains/from/KEL/to/LST", "/departures/KEL/to/LST/5", "KEL", "LST"},
		{"/api/commute/trains/from/lst/to/kel?count=1", "/departures/LST/to/KEL/1", "LST", "KEL"},
		{"/api/commute/trains/from/ABC/to/XYZ?count=10", "/departures/ABC/to/XYZ/10", "ABC", "XYZ"},
	} {
		t.Run(test.target, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.providerPath {
					t.Errorf("unexpected route: %s", request.URL.Path)
				}
				writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":null}`))
			}))
			t.Cleanup(provider.Close)
			client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
			handler := NewCommuteHandler(services.NewCommuteService(client, nil))
			recorder := serveCommute(handler, test.target)
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected 200: %d %s", recorder.Code, recorder.Body.String())
			}
			var board models.CommuteBoard
			if err := json.Unmarshal(recorder.Body.Bytes(), &board); err != nil {
				t.Fatal(err)
			}
			if board.Origin.Code != test.origin || board.Destination.Code != test.destination || board.Trains == nil || board.FetchedAt.IsZero() || board.GeneratedAt.IsZero() {
				t.Fatalf("unexpected board: %s", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), `"direction"`) {
				t.Fatal("the response must not choose a Home/Work direction")
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("HTTP caches must not retain train times")
			}
		})
	}
}

func TestCommuteAPIRejectsInvalidParameters(t *testing.T) {
	service := services.NewCommuteService(services.NewHuxleyClient("", "", nil), nil)
	handler := NewCommuteHandler(service)
	for _, target := range []string{
		"/api/commute/trains/from/AB/to/LST",
		"/api/commute/trains/from/K3L/to/LST",
		"/api/commute/trains/from/KEL/to/LST?count=",
		"/api/commute/trains/from/KEL/to/LST?count=0",
		"/api/commute/trains/from/KEL/to/LST?count=-1",
		"/api/commute/trains/from/KEL/to/LST?count=11",
		"/api/commute/trains/from/KEL/to/LST?count=abc",
		"/api/commute/trains/from/KEL/to/LST?count=1.5",
		"/api/commute/trains/from/KEL/to/LST?count=1&count=2",
	} {
		t.Run(target, func(t *testing.T) {
			recorder := serveCommute(handler, target)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", recorder.Code)
			}
		})
	}
}

func TestCommuteAPICachesByRouteAndCountThenErrorsAfterExpiry(t *testing.T) {
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
	request := func(target string) *httptest.ResponseRecorder { return serveCommute(handler, target) }
	base := "/api/commute/trains/from/KEL/to/LST"
	first := request(base)
	if first.Code != http.StatusOK {
		t.Fatalf("initial request: %s", first.Body.String())
	}
	available.Store(false)
	now = now.Add(29 * time.Second)
	cached := request(base)
	if cached.Code != http.StatusOK || cached.Body.String() != first.Body.String() {
		t.Fatal("fresh cache is not reused")
	}
	if separateCount := request(base + "?count=1"); separateCount.Code != http.StatusBadGateway {
		t.Fatal("cache reused for a different count")
	}
	if separateRoute := request("/api/commute/trains/from/LST/to/KEL"); separateRoute.Code != http.StatusBadGateway {
		t.Fatal("cache reused for a different route")
	}
	now = now.Add(time.Second)
	if expired := request(base); expired.Code != http.StatusBadGateway {
		t.Fatalf("expired cache returned %d", expired.Code)
	}
	available.Store(true)
	if recovered := request(base); recovered.Code != http.StatusOK || recovered.Body.String() == first.Body.String() {
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
			recorder := serveCommute(handler, "/api/commute/trains/from/KEL/to/LST")
			if recorder.Code != test.expectedStatus || strings.Contains(recorder.Body.String(), "private-test-token") {
				t.Fatalf("expected safe error %d, got %d %s", test.expectedStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func serveCommute(handler *CommuteHandler, target string) *httptest.ResponseRecorder {
	router := chi.NewRouter()
	router.Get("/api/commute/trains/from/{origin}/to/{destination}", handler.ListTrains)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	return recorder
}
