package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDecodeJSONBodyWithLimit(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		maximum    int64
		wantStatus int
	}{
		{name: "valid body at limit", body: `{"value":1}`, maximum: 11, wantStatus: http.StatusOK},
		{name: "body above limit", body: `{"value":1} `, maximum: 11, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "trailing JSON value", body: `{"value":1}{}`, maximum: 32, wantStatus: http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			var decoded map[string]int

			if decodeJSONBodyWithLimit(response, request, &decoded, test.maximum) {
				if test.wantStatus != http.StatusOK {
					t.Fatalf("expected rejection with status %d", test.wantStatus)
				}
				return
			}
			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", test.wantStatus, response.Code, response.Body.String())
			}
		})
	}
}

func TestDecodeOIDCUserInfoEnforcesResponseLimit(t *testing.T) {
	valid := `{"sub":"user-1"}`
	if userInfo, err := decodeOIDCUserInfo(strings.NewReader(valid)); err != nil || userInfo.Sub != "user-1" {
		t.Fatalf("expected valid user info, got %#v, %v", userInfo, err)
	}

	oversized := `{"sub":"user-1","padding":"` + strings.Repeat("x", int(maxOIDCResponseBytes)) + `"}`
	if _, err := decodeOIDCUserInfo(strings.NewReader(oversized)); err == nil {
		t.Fatal("expected oversized user-info response to fail")
	}
}

func TestExchangeTokenUsesBoundedHTTPTimeout(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer provider.Close()

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, provider.URL, "", "")
	handler.oidcUserInfoClient = &http.Client{Timeout: 10 * time.Millisecond}
	request := httptest.NewRequest(http.MethodPost, "/api/auth/exchange", nil)
	request.Header.Set("Authorization", "Bearer oidc-token")
	response := httptest.NewRecorder()

	handler.ExchangeToken(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected timeout to reject exchange with 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCopyRestoreDatabaseEnforcesExpandedSizeLimit(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{name: "exact limit", input: "12345678"},
		{name: "over limit", input: "123456789", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var destination bytes.Buffer
			_, err := copyRestoreDatabase(&destination, strings.NewReader(test.input), 8)
			if (err != nil) != test.wantError {
				t.Fatalf("expected error %t, got %v", test.wantError, err)
			}
			if destination.Len() > 8 {
				t.Fatalf("wrote %d bytes beyond the 8-byte limit", destination.Len())
			}
		})
	}
}

func TestCopyRestoreDatabasePropagatesReaderErrors(t *testing.T) {
	expectedError := io.ErrUnexpectedEOF
	reader := errorReader{err: expectedError}
	var destination bytes.Buffer
	if _, err := copyRestoreDatabase(&destination, reader, 8); err != expectedError {
		t.Fatalf("expected %v, got %v", expectedError, err)
	}
}

func TestParseRestoreMultipartFormEnforcesRequestSizeLimit(t *testing.T) {
	body, contentType := buildMultipartFile(t, "backup.db.gz", []byte("0123456789"))
	request := httptest.NewRequest(http.MethodPost, "/", body)
	request.Header.Set("Content-Type", contentType)
	if err := parseRestoreMultipartForm(httptest.NewRecorder(), request, 8); err == nil {
		t.Fatal("expected oversized multipart request to fail")
	} else {
		var maxBytesError *http.MaxBytesError
		if !errors.As(err, &maxBytesError) {
			t.Fatalf("expected MaxBytesError, got %v", err)
		}
	}
}

type errorReader struct{ err error }

func (reader errorReader) Read([]byte) (int, error) { return 0, reader.err }
