package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/bins"
)

type binHandlerStore struct {
	schedule         bins.Schedule
	loadErr, saveErr error
}

func (store *binHandlerStore) Load(context.Context) (bins.Schedule, error) {
	return store.schedule, store.loadErr
}
func (store *binHandlerStore) Save(_ context.Context, schedule bins.Schedule) error {
	if store.saveErr != nil {
		return store.saveErr
	}
	store.schedule = schedule
	return nil
}

type binHandlerParser struct{ err error }

func (parser binHandlerParser) Parse(string, time.Time) ([]bins.Collection, error) {
	return []bins.Collection{{Date: "2099-01-01", Type: "food_caddy", Name: "Food caddy"}}, parser.err
}
func TestBinUploadFailureResponses(t *testing.T) {
	failure := errors.New("failed")
	tests := []struct {
		name    string
		store   binHandlerStore
		parser  binHandlerParser
		status  int
		message string
	}{
		{name: "parse", parser: binHandlerParser{err: failure}, status: http.StatusBadRequest, message: "calendar has no upcoming bin collections or is not a valid ICS file"},
		{name: "save", store: binHandlerStore{saveErr: failure}, status: http.StatusInternalServerError, message: "failed to save bin schedule"},
		{name: "reload", store: binHandlerStore{loadErr: failure}, status: http.StatusInternalServerError, message: "failed to load saved bin schedule"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewBinCollectionsHandler(bins.New(&test.store, test.parser, true, time.Now))
			request, _ := binScheduleUploadRequest(t, "schedule.ics", testBinScheduleICS)
			response := httptest.NewRecorder()
			handler.Upload(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.message) {
				t.Fatalf("got %d %s", response.Code, response.Body.String())
			}
		})
	}
}
func TestBinUpdateFailureResponses(t *testing.T) {
	failure := errors.New("failed")
	tests := []struct {
		name    string
		store   binHandlerStore
		push    bool
		body    string
		status  int
		message string
	}{
		{name: "push unavailable", body: `{"enabled":true}`, status: http.StatusServiceUnavailable, message: "push notifications are not configured on the server"},
		{name: "missing schedule", push: true, body: `{"enabled":true}`, status: http.StatusBadRequest, message: "upload a bin collection calendar before enabling reminders"},
		{name: "load failure", store: binHandlerStore{loadErr: failure}, push: true, body: `{"enabled":true}`, status: http.StatusInternalServerError, message: "failed to load bin schedule"},
		{name: "update read failure", store: binHandlerStore{loadErr: failure}, body: `{"enabled":false}`, status: http.StatusInternalServerError, message: "failed to update bin reminders"},
		{name: "update write failure", store: binHandlerStore{saveErr: failure}, body: `{"enabled":false}`, status: http.StatusInternalServerError, message: "failed to update bin reminders"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewBinCollectionsHandler(bins.New(&test.store, binHandlerParser{}, test.push, time.Now))
			request := httptest.NewRequest(http.MethodPatch, "/api/bin-collections", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			handler.Update(response, request)
			if response.Code != test.status || !strings.Contains(response.Body.String(), test.message) {
				t.Fatalf("got %d %s", response.Code, response.Body.String())
			}
		})
	}
}
