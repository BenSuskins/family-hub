package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func binUploadContractRequest(t *testing.T, fixture *contractFixture, token, contents string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("schedule", "schedule.ics")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/bin-collections", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	fixture.requests++
	request.RemoteAddr = fmt.Sprintf("198.18.%d.%d:1234", fixture.requests/250, fixture.requests%250+1)
	route, parameters, err := fixture.router.FindRoute(request)
	if err != nil {
		t.Fatal(err)
	}
	input := &openapi3filter.RequestValidationInput{Request: request, PathParams: parameters, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
	if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
		t.Fatalf("upload request: %v", err)
	}
	response := httptest.NewRecorder()
	fixture.application.router.ServeHTTP(response, request)
	validation := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: response.Code, Header: response.Header(), Options: &openapi3filter.Options{IncludeResponseStatus: true}}
	validation.SetBodyBytes(response.Body.Bytes())
	if err := openapi3filter.ValidateResponse(context.Background(), validation); err != nil {
		t.Fatalf("upload response %d: %v: %s", response.Code, err, response.Body.String())
	}
	return response
}

func TestOpenAPIBinCollectionsContract(t *testing.T) {
	fixture := newContractFixture(t)
	path := "/api/bin-collections"
	if response := fixture.request(t, http.MethodGet, path, "", "", false); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous get: %d %s", response.Code, response.Body.String())
	}
	if response := fixture.request(t, http.MethodGet, "/api/display/bin-collections", "member", "", false); response.Code != http.StatusUnauthorized {
		t.Fatalf("member display: %d %s", response.Code, response.Body.String())
	}
	if response := fixture.request(t, http.MethodPatch, path, "member", `{"enabled":true}`, true); response.Code != http.StatusForbidden {
		t.Fatalf("member update: %d %s", response.Code, response.Body.String())
	}
	const calendar = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:food\r\nDTSTART;VALUE=DATE:20990101\r\nSUMMARY:Food Waste\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	if response := binUploadContractRequest(t, fixture, "member", calendar); response.Code != http.StatusForbidden {
		t.Fatalf("member upload: %d %s", response.Code, response.Body.String())
	}
	unavailable := fixture.request(t, http.MethodPatch, path, "admin", `{"enabled":true}`, true)
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("enable without push: %d %s", unavailable.Code, unavailable.Body.String())
	}
	uploaded := binUploadContractRequest(t, fixture, "admin", calendar)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", uploaded.Code, uploaded.Body.String())
	}
	var status struct {
		Enabled       bool   `json:"enabled"`
		Count         int    `json:"collection_count"`
		PushAvailable bool   `json:"push_available"`
		UploadedAt    string `json:"uploaded_at"`
	}
	if err := json.Unmarshal(uploaded.Body.Bytes(), &status); err != nil || status.Count != 1 || status.Enabled || status.PushAvailable || status.UploadedAt == "" {
		t.Fatalf("upload response: %s: %v", uploaded.Body.String(), err)
	}
	if response := binUploadContractRequest(t, fixture, "admin", "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n"); response.Code != http.StatusBadRequest {
		t.Fatalf("invalid upload: %d %s", response.Code, response.Body.String())
	}
	display := fixture.request(t, http.MethodGet, "/api/display/bin-collections", "display", "", true)
	if display.Code != http.StatusOK {
		t.Fatalf("display: %d %s", display.Code, display.Body.String())
	}
	var data struct {
		Collections []struct {
			Date string `json:"date"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"collections"`
	}
	if err := json.Unmarshal(display.Body.Bytes(), &data); err != nil || len(data.Collections) != 1 || data.Collections[0].Name != "Food caddy" || data.Collections[0].Date != "2099-01-01" {
		t.Fatalf("display response: %s: %v", display.Body.String(), err)
	}
	if err := repository.NewSettingsRepository(fixture.database).Set(context.Background(), "bin_schedule", "{"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ path, token string }{{path, "member"}, {"/api/display/bin-collections", "display"}} {
		response := fixture.request(t, http.MethodGet, test.path, test.token, "", true)
		var failure map[string]string
		if err := json.Unmarshal(response.Body.Bytes(), &failure); response.Code != http.StatusInternalServerError || err != nil || failure["error"] != "failed to load bin schedule" {
			t.Fatalf("malformed schedule %s: %d %s (%v)", test.path, response.Code, response.Body.String(), err)
		}
	}
}
