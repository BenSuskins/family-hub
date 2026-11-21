package services_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/services"
)

type huxleyClockTransport func(*http.Request) (*http.Response, error)

func (transport huxleyClockTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestHuxleyClientUsesInjectedFetchClock(t *testing.T) {
	client := &http.Client{Transport: huxleyClockTransport(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"generatedAt":"2026-11-21T10:00:00Z","trainServices":[]}`)), Header: make(http.Header), Request: request}, nil
	})}
	instant := time.Date(2026, 11, 21, 12, 34, 0, 0, time.FixedZone("offset", 2*60*60))
	provider := services.NewHuxleyClient("https://rail.example", "token", client, func() time.Time { return instant })
	board, err := provider.FetchTrains(context.Background(), "ABC", "DEF", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !board.FetchedAt.Equal(instant) || board.FetchedAt.Location() != time.UTC {
		t.Fatalf("fetched at %v, want %v UTC", board.FetchedAt, instant.UTC())
	}
}
