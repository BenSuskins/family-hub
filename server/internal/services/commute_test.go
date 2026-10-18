package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/services"
)

const commuteBoard = `{
  "generatedAt":"2026-10-18T08:00:00+01:00",
  "locationName":"Kelvedon","crs":"KEL",
  "trainServices":[{
    "serviceID":"outbound-service","operator":"Greater Anglia",
    "std":"08:15","etd":"On time","platform":"1",
    "subsequentCallingPoints":[{"callingPoint":[
      {"crs":"SRA","st":"09:00","et":"On time"},
      {"crs":"LST","st":"09:10","et":"On time"}
    ]}]
  }]
}`

func TestHuxleyClientReturnsCommuteTimes(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/departures/KEL/to/LST/5" {
			t.Errorf("unexpected provider path: %s", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("expand") != "true" || query.Get("accessToken") != "test-token" || query.Get("timeWindow") != "120" {
			t.Error("provider request lacks expanded details, token, or time window")
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Write([]byte(commuteBoard))
	}))
	t.Cleanup(provider.Close)

	client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
	board, err := client.FetchTrains(context.Background(), "to_work", 5)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Direction string `json:"direction"`
		Timezone  string `json:"timezone"`
		Trains    []struct {
			ServiceID          string `json:"serviceId"`
			Status             string `json:"status"`
			Platform           string `json:"platform"`
			ScheduledDeparture string `json:"scheduledDeparture"`
			ExpectedDeparture  string `json:"expectedDeparture"`
			ScheduledArrival   string `json:"scheduledArrival"`
			ExpectedArrival    string `json:"expectedArrival"`
		} `json:"trains"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatal(err)
	}
	if response.Direction != "to_work" || response.Timezone != "Europe/London" || len(response.Trains) != 1 {
		t.Fatalf("unexpected board: %s", data)
	}
	train := response.Trains[0]
	if train.ServiceID != "outbound-service" || train.Status != "on_time" || train.Platform != "1" ||
		train.ScheduledDeparture != "08:15" || train.ExpectedDeparture != "08:15" ||
		train.ScheduledArrival != "09:10" || train.ExpectedArrival != "09:10" {
		t.Fatalf("unexpected train: %s", data)
	}
}

func TestHuxleyClientReturnsTrainStatusAndUnknownValues(t *testing.T) {
	tests := []struct {
		name              string
		train             string
		status            string
		expectedDeparture string
		expectedArrival   string
	}{
		{"late departure", `"std":"08:15","etd":"08:22","delayReason":"Signal failure"`, "delayed", "08:22", ""},
		{"unknown delay", `"std":"08:15","etd":"Delayed"`, "delayed", "", ""},
		{"cancelled train", `"std":"08:15","etd":"On time","isCancelled":true,"cancelReason":"Staff shortage"`, "cancelled", "", ""},
		{"cancelled destination", `"std":"08:15","etd":"On time","filterLocationCancelled":true`, "cancelled", "", ""},
		{"cancelled stop", `"std":"08:15","etd":"On time","subsequentCallingPoints":[{"callingPoint":[{"crs":"LST","st":"09:10","et":"On time","isCancelled":true}]}]`, "cancelled", "", ""},
		{"late arrival", `"std":"08:15","etd":"On time","subsequentCallingPoints":[{"callingPoint":[{"crs":"LST","st":"09:10","et":"09:15"}]}]`, "delayed", "08:15", "09:15"},
		{"unknown arrival delay", `"std":"08:15","etd":"On time","subsequentCallingPoints":[{"callingPoint":[{"crs":"LST","st":"09:10","et":"Delayed"}]}]`, "delayed", "08:15", ""},
		{"early arrival", `"std":"08:15","etd":"On time","subsequentCallingPoints":[{"callingPoint":[{"crs":"LST","st":"09:10","et":"09:08"}]}]`, "on_time", "08:15", "09:08"},
		{"unknown estimate", `"std":"08:15","etd":"No report"`, "unknown", "", ""},
		{"missing times", `"std":null,"etd":null`, "unknown", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":[{"serviceID":"train",` + test.train + `}]}`))
			}))
			t.Cleanup(provider.Close)
			client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
			board, err := client.FetchTrains(context.Background(), "to_work", 5)
			if err != nil {
				t.Fatal(err)
			}
			train := board.Trains[0]
			if train.Status != test.status || train.Platform != nil {
				t.Fatalf("unexpected status or platform: %+v", train)
			}
			checkCommuteTime(t, train.ExpectedDeparture, test.expectedDeparture)
			checkCommuteTime(t, train.ExpectedArrival, test.expectedArrival)
			if test.name == "late departure" && (train.DelayReason == nil || *train.DelayReason != "Signal failure") {
				t.Fatal("missing delay reason")
			}
			if test.name == "cancelled train" && (train.CancellationReason == nil || *train.CancellationReason != "Staff shortage") {
				t.Fatal("missing cancellation reason")
			}
		})
	}
}

func checkCommuteTime(t *testing.T, actual *string, expected string) {
	t.Helper()
	if expected == "" {
		if actual != nil {
			t.Errorf("expected null, got %q", *actual)
		}
	} else if actual == nil || *actual != expected {
		t.Errorf("expected %q, got %v", expected, actual)
	}
}

func TestHuxleyClientReturnsHomeArrivalAtIntermediateStop(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/departures/LST/to/KEL/1" {
			t.Errorf("unexpected return route: %s", request.URL.Path)
		}
		writer.Write([]byte(`{"generatedAt":"2026-10-18T23:30:00+01:00","trainServices":[{
		  "serviceID":"return-service","std":"23:45","etd":"On time","platform":"12",
		  "destination":[{"crs":"IPS","locationName":"Ipswich"}],
		  "subsequentCallingPoints":[{"callingPoint":[{"crs":"KEL","st":"00:35","et":"00:40"},{"crs":"IPS","st":"01:00","et":"01:05"}]}]
		}]}`))
	}))
	t.Cleanup(provider.Close)
	client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
	board, err := client.FetchTrains(context.Background(), "to_home", 1)
	if err != nil {
		t.Fatal(err)
	}
	if board.Origin.Code != "LST" || board.Destination.Code != "KEL" {
		t.Fatalf("unexpected route: %+v", board)
	}
	checkCommuteTime(t, board.Trains[0].ScheduledArrival, "00:35")
	checkCommuteTime(t, board.Trains[0].ExpectedArrival, "00:40")
	if board.Trains[0].Status != "delayed" {
		t.Fatal("arrival delay is absent")
	}
}

func TestHuxleyClientOrdersDeparturesAcrossMidnightAndLimitsCount(t *testing.T) {
	for _, generated := range []string{"2026-10-18T23:30:00+01:00", "2026-10-19T00:01:00+01:00"} {
		t.Run(generated, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write([]byte(`{"generatedAt":"` + generated + `","trainServices":[
				  {"serviceID":"third","std":"00:20","etd":"On time"},
				  {"serviceID":"first","std":"23:55","etd":"00:05"},
				  {"serviceID":"second","std":"00:10","etd":"On time"}
				]}`))
			}))
			t.Cleanup(provider.Close)
			client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
			board, err := client.FetchTrains(context.Background(), "to_work", 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(board.Trains) != 2 || board.Trains[0].ServiceID != "first" || board.Trains[1].ServiceID != "second" {
				t.Fatalf("departures are not limited and ordered: %+v", board.Trains)
			}
		})
	}
}

func TestHuxleyClientReturnsEmptyArray(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"generatedAt":"2026-10-18T08:00:00+01:00","trainServices":null}`))
	}))
	t.Cleanup(provider.Close)
	client := services.NewHuxleyClient(provider.URL, "test-token", provider.Client())
	board, err := client.FetchTrains(context.Background(), "to_work", 5)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"trains":[]`) {
		t.Fatalf("expected empty array: %s", data)
	}
}

func TestHuxleyClientRejectsProviderFailuresWithoutLeakingToken(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"upstream error", http.StatusUnauthorized, "private-test-token"},
		{"invalid JSON", http.StatusOK, "private-test-token"},
		{"missing timestamp", http.StatusOK, `{}`},
		{"invalid timestamp", http.StatusOK, `{"generatedAt":"yesterday"}`},
		{"invalid board", http.StatusOK, `[]`},
		{"trailing data", http.StatusOK, commuteBoard + `{}`},
		{"oversized body", http.StatusOK, commuteBoard + strings.Repeat(" ", 2*1024*1024)},
		{"redirect", http.StatusFound, "private-test-token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if test.status == http.StatusFound {
					writer.Header().Set("Location", "/redirected")
				}
				writer.WriteHeader(test.status)
				writer.Write([]byte(test.body))
			}))
			t.Cleanup(provider.Close)
			client := services.NewHuxleyClient(provider.URL, "private-test-token", provider.Client())
			_, err := client.FetchTrains(context.Background(), "to_work", 5)
			if !errors.Is(err, services.ErrCommuteProvider) || strings.Contains(err.Error(), "private-test-token") {
				t.Fatalf("expected safe provider error, got %v", err)
			}
		})
	}
	t.Run("missing token", func(t *testing.T) {
		client := services.NewHuxleyClient("https://example.invalid", "", nil)
		_, err := client.FetchTrains(context.Background(), "to_work", 5)
		if !errors.Is(err, services.ErrCommuteUnavailable) {
			t.Fatalf("expected unavailable, got %v", err)
		}
	})
	t.Run("invalid URL", func(t *testing.T) {
		client := services.NewHuxleyClient("https://example.invalid/%private-test-token", "private-test-token", nil)
		_, err := client.FetchTrains(context.Background(), "to_work", 5)
		if !errors.Is(err, services.ErrCommuteProvider) || strings.Contains(err.Error(), "private-test-token") {
			t.Fatalf("expected safe URL error, got %v", err)
		}
	})
	t.Run("transport error", func(t *testing.T) {
		provider := httptest.NewServer(http.NotFoundHandler())
		provider.Close()
		client := services.NewHuxleyClient(provider.URL, "private-test-token", provider.Client())
		_, err := client.FetchTrains(context.Background(), "to_work", 5)
		if !errors.Is(err, services.ErrCommuteProvider) || strings.Contains(err.Error(), "private-test-token") {
			t.Fatalf("expected safe transport error, got %v", err)
		}
	})
}

func TestHuxleyClientHandlesTimeoutAndCancellation(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	t.Cleanup(provider.Close)
	httpClient := provider.Client()
	httpClient.Timeout = 20 * time.Millisecond
	client := services.NewHuxleyClient(provider.URL, "private-test-token", httpClient)
	_, err := client.FetchTrains(context.Background(), "to_work", 5)
	if !errors.Is(err, services.ErrCommuteTimeout) || strings.Contains(err.Error(), "private-test-token") {
		t.Fatalf("expected safe timeout, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.FetchTrains(ctx, "to_work", 5)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected request cancellation, got %v", err)
	}
}
