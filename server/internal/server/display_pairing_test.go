package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestDisplayPairingRoutesEnforceScopeAndRevocation(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	ctx := context.Background()
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	admin, err := users.Create(ctx, models.User{OIDCSubject: "pair-route-admin", Email: "pair-route@example.com", Name: "Admin", Role: models.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tokens.Create(ctx, models.APIToken{Name: "Web App", TokenHash: repository.HashToken("admin-token"), Scope: models.TokenScopeAPI, CreatedByUserID: admin.ID}); err != nil {
		t.Fatal(err)
	}
	member, err := users.Create(ctx, models.User{OIDCSubject: "pair-route-member", Email: "pair-member@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tokens.Create(ctx, models.APIToken{Name: "Web App", TokenHash: repository.HashToken("member-token"), Scope: models.TokenScopeAPI, CreatedByUserID: member.ID}); err != nil {
		t.Fatal(err)
	}
	application := New(database, config.Config{}, nil)
	for _, test := range []struct {
		method string
		path   string
		token  string
		want   int
	}{{http.MethodGet, "/api/display/home-info", "admin-token", http.StatusUnauthorized}, {http.MethodGet, "/api/displays", "member-token", http.StatusForbidden}, {http.MethodPost, "/api/display-pairing/challenges/challenge/approve", "", http.StatusUnauthorized}} {
		request := httptest.NewRequest(test.method, test.path, strings.NewReader(`{"name":"Kitchen","pairingCode":"`+strings.Repeat("a", 64)+`"}`))
		if test.token != "" {
			request.Header.Set("Authorization", "Bearer "+test.token)
		}
		response := httptest.NewRecorder()
		application.router.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("%s %s with token %t returned %d, want %d", test.method, test.path, test.token != "", response.Code, test.want)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/api/display-pairing/challenges", nil)
	response := httptest.NewRecorder()
	application.router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("create challenge: %d %s", response.Code, response.Body.String())
	}
	var challenge struct {
		ID   string `json:"challengeId"`
		Code string `json:"pairingCode"`
	}
	if err := json.NewDecoder(response.Body).Decode(&challenge); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/display-pairing/challenges/"+challenge.ID+"/approve", strings.NewReader(`{"name":"Kitchen","pairingCode":"`+challenge.Code+`"}`))
	request.Header.Set("Authorization", "Bearer admin-token")
	response = httptest.NewRecorder()
	application.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("approve challenge: %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/display-pairing/challenges/"+challenge.ID+"/claim", strings.NewReader(`{"pairingCode":"`+challenge.Code+`"}`))
	response = httptest.NewRecorder()
	application.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("claim challenge: %d %s", response.Code, response.Body.String())
	}
	var claimed struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&claimed); err != nil {
		t.Fatal(err)
	}
	settingsRequest := httptest.NewRequest(http.MethodGet, "/api/display/settings", nil)
	settingsRequest.Header.Set("Authorization", "Bearer "+claimed.Token)
	settingsResponse := httptest.NewRecorder()
	application.router.ServeHTTP(settingsResponse, settingsRequest)
	var settings struct {
		EnabledSections []string `json:"enabledSections"`
	}
	if err := json.NewDecoder(settingsResponse.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if settingsResponse.Code != http.StatusOK || !slices.Contains(settings.EnabledSections, "inventory") {
		t.Fatalf("new display settings = %d %v, expected inventory enabled", settingsResponse.Code, settings.EnabledSections)
	}
	updateSettings := func(sections string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPatch, "/api/displays/"+challenge.ID, strings.NewReader(`{"enabledSections":`+sections+`}`))
		request.Header.Set("Authorization", "Bearer admin-token")
		response := httptest.NewRecorder()
		application.router.ServeHTTP(response, request)
		return response
	}
	if response := updateSettings(`["calendar","inventory"]`); response.Code != http.StatusOK {
		t.Fatalf("save inventory section: %d %s", response.Code, response.Body.String())
	}
	if response := updateSettings(`["calendar"]`); response.Code != http.StatusOK {
		t.Fatalf("save existing section choice: %d %s", response.Code, response.Body.String())
	}
	settingsRequest = httptest.NewRequest(http.MethodGet, "/api/display/settings", nil)
	settingsRequest.Header.Set("Authorization", "Bearer "+claimed.Token)
	settingsResponse = httptest.NewRecorder()
	application.router.ServeHTTP(settingsResponse, settingsRequest)
	if err := json.NewDecoder(settingsResponse.Body).Decode(&settings); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(settings.EnabledSections, []string{"calendar"}) {
		t.Fatalf("saved sections changed unexpectedly: %v", settings.EnabledSections)
	}
	for _, test := range []struct {
		path string
		want int
	}{{"/api/display/home-info", http.StatusOK}, {"/api/me", http.StatusUnauthorized}} {
		request = httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Header.Set("Authorization", "Bearer "+claimed.Token)
		response = httptest.NewRecorder()
		application.router.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf("display token %s returned %d, want %d", test.path, response.Code, test.want)
		}
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/displays/"+challenge.ID, nil)
	request.Header.Set("Authorization", "Bearer admin-token")
	response = httptest.NewRecorder()
	application.router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("revoke display: %d %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/display/home-info", nil)
	request.Header.Set("Authorization", "Bearer "+claimed.Token)
	response = httptest.NewRecorder()
	application.router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("revoked display token returned %d", response.Code)
	}
}

func TestScratchpadRoutesPersistSharedStrokesAndRejectInvalidColors(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(context.Background(), models.User{OIDCSubject: "scratchpad-route", Email: "scratchpad-route@example.com", Name: "Member", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	tokens := repository.NewAPITokenRepository(database)
	if _, err := tokens.Create(context.Background(), models.APIToken{Name: "Web App", TokenHash: repository.HashToken("scratch-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	application := New(database, config.Config{}, nil)
	save := httptest.NewRequest(http.MethodPut, "/api/scratchpad", strings.NewReader(`{"strokes":[{"color":"#3c654c","points":[{"x":0.2,"y":0.7}]}]}`))
	save.Header.Set("Authorization", "Bearer scratch-token")
	saved := httptest.NewRecorder()
	application.router.ServeHTTP(saved, save)
	if saved.Code != http.StatusOK {
		t.Fatalf("save scratchpad: %d %s", saved.Code, saved.Body.String())
	}
	read := httptest.NewRequest(http.MethodGet, "/api/scratchpad", nil)
	read.Header.Set("Authorization", "Bearer scratch-token")
	loaded := httptest.NewRecorder()
	application.router.ServeHTTP(loaded, read)
	if loaded.Code != http.StatusOK || !strings.Contains(loaded.Body.String(), `"x":0.2`) {
		t.Fatalf("read scratchpad: %d %s", loaded.Code, loaded.Body.String())
	}
	invalid := httptest.NewRequest(http.MethodPut, "/api/scratchpad", strings.NewReader(`{"strokes":[{"color":"#ffffff","points":[{"x":0.2,"y":0.7}]}]}`))
	invalid.Header.Set("Authorization", "Bearer scratch-token")
	rejected := httptest.NewRecorder()
	application.router.ServeHTTP(rejected, invalid)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("invalid color returned %d", rejected.Code)
	}
}
