package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/apidocs"
	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/go-chi/chi/v5"
)

type contractFixture struct {
	application   *Server
	specification *openapi3.T
	router        routers.Router
	requests      int
}

func newContractFixture(t *testing.T) *contractFixture {
	t.Helper()
	specification, err := openapi3.NewLoader().LoadFromData(apidocs.Specification())
	if err != nil {
		t.Fatal(err)
	}
	if err := specification.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	contractRouter, err := legacy.NewRouter(specification)
	if err != nil {
		t.Fatal(err)
	}
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	for _, role := range []models.Role{models.RoleMember, models.RoleAdmin} {
		user, err := users.Create(context.Background(), models.User{OIDCSubject: string(role), Email: string(role) + "@example.com", Name: string(role), Role: role})
		if err != nil {
			t.Fatal(err)
		}
		_, err = tokens.Create(context.Background(), models.APIToken{Name: string(role), TokenHash: repository.HashToken(string(role)), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID})
		if err != nil {
			t.Fatal(err)
		}
		if role == models.RoleAdmin {
			_, err = tokens.Create(context.Background(), models.APIToken{Name: "display", TokenHash: repository.HashToken("display"), Scope: models.TokenScopeDisplay, CreatedByUserID: user.ID})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	return &contractFixture{application: New(database, config.Config{APIDocsEnabled: true}), specification: specification, router: contractRouter}
}

func (fixture *contractFixture) request(t *testing.T, method, path, token, body string, validateRequest bool) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	fixture.requests++
	request.RemoteAddr = fmt.Sprintf("198.18.%d.%d:1234", fixture.requests/250, fixture.requests%250+1)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	route, parameters, err := fixture.router.FindRoute(request)
	if err != nil {
		t.Fatal(err)
	}
	input := &openapi3filter.RequestValidationInput{Request: request, PathParams: parameters, Route: route, Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
	if validateRequest {
		if err := openapi3filter.ValidateRequest(context.Background(), input); err != nil {
			t.Fatalf("%s %s request: %v", method, path, err)
		}
	}
	response := httptest.NewRecorder()
	fixture.application.router.ServeHTTP(response, request)
	validation := &openapi3filter.ResponseValidationInput{RequestValidationInput: input, Status: response.Code, Header: response.Header(), Options: &openapi3filter.Options{IncludeResponseStatus: true}}
	validation.SetBodyBytes(response.Body.Bytes())
	if response.Code >= 400 || strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
		if err := openapi3filter.ValidateResponse(context.Background(), validation); err != nil {
			t.Fatalf("%s %s response %d: %v\n%s", method, path, response.Code, err, response.Body.String())
		}
	} else {
		validation.Options.ExcludeResponseBody = true
		if err := openapi3filter.ValidateResponse(context.Background(), validation); err != nil {
			t.Fatal(err)
		}
	}
	return response
}

func TestOpenAPITracksEveryRegisteredOperation(t *testing.T) {
	fixture := newContractFixture(t)
	actual := map[string]bool{}
	if err := chi.Walk(fixture.application.router, func(method, path string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		actual[method+" "+path] = true
		documented := fixture.specification.Paths.Value(path)
		if documented == nil || documented.GetOperation(method) == nil {
			t.Errorf("undocumented operation: %s %s", method, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for path, item := range fixture.specification.Paths.Map() {
		for method := range item.Operations() {
			if !actual[method+" "+path] {
				t.Errorf("documented operation is not registered: %s %s", method, path)
			}
		}
	}
}

func TestOpenAPIAccessRulesAgainstRealRouter(t *testing.T) {
	fixture := newContractFixture(t)
	placeholder := regexp.MustCompile(`\{[^}]+\}`)
	for path, item := range fixture.specification.Paths.Map() {
		for method, operation := range item.Operations() {
			access, _ := operation.Extensions["x-access"].(string)
			if access == "public" || access == "developer" {
				continue
			}
			t.Run(method+" "+path, func(t *testing.T) {
				target := placeholder.ReplaceAllString(path, "missing")
				credentials := []string{"", "invalid"}
				if access == "oidc" {
					credentials = []string{""}
				}
				for _, token := range credentials {
					response := fixture.request(t, method, target, token, "", false)
					if response.Code != http.StatusUnauthorized {
						t.Fatalf("%s credential %q: got %d, want 401", access, token, response.Code)
					}
				}
				if access == "oidc" {
					return
				}
				wrongScope := "display"
				if access == "display" {
					wrongScope = "member"
				}
				if response := fixture.request(t, method, target, wrongScope, "", false); response.Code != http.StatusUnauthorized {
					t.Fatalf("wrong scope: got %d, want 401", response.Code)
				}
				if access == "admin" {
					if response := fixture.request(t, method, target, "member", "", false); response.Code != http.StatusForbidden {
						t.Fatalf("member on admin route: got %d, want 403", response.Code)
					}
				}
			})
		}
	}
}

func TestOpenAPIReadContracts(t *testing.T) {
	fixture := newContractFixture(t)
	for _, path := range []string{
		"/api/client-config", "/api/me", "/api/users", "/api/categories", "/api/settings", "/api/home-info", "/api/weather", "/api/departures", "/api/chores", "/api/chore-definitions", "/api/chore-history", "/api/meals", "/api/recipes", "/api/calendar", "/api/inventory", "/api/shopping", "/api/scratchpad", "/api/chore-reminders", "/api/bin-collections", "/api/dashboard", "/api/tokens", "/api/calendars", "/api/displays",
		"/api/display/home-info", "/api/display/weather", "/api/display/departures", "/api/display/bin-collections", "/api/display/chores", "/api/display/dashboard", "/api/display/people", "/api/display/meals", "/api/display/calendar", "/api/display/shopping", "/api/display/inventory", "/api/display/scratchpad", "/api/display/recipes", "/api/openapi.json",
	} {
		t.Run(path, func(t *testing.T) {
			token := "admin"
			if strings.HasPrefix(path, "/api/display/") {
				token = "display"
			}
			response := fixture.request(t, http.MethodGet, path, token, "", true)
			if response.Code != http.StatusOK {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestOpenAPIChoreLifecycleContract(t *testing.T) {
	fixture := newContractFixture(t)
	created := fixture.request(t, http.MethodPost, "/api/chores", "member", `{"name":"Take out bins","recurrenceType":"none"}`, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var chore models.Chore
	if err := json.Unmarshal(created.Body.Bytes(), &chore); err != nil {
		t.Fatal(err)
	}
	path := "/api/chores/" + chore.ID
	completed := fixture.request(t, http.MethodPost, path+"/complete", "member", "", true)
	if completed.Code != http.StatusNoContent {
		t.Fatalf("complete: %d %s", completed.Code, completed.Body.String())
	}
	repeated := fixture.request(t, http.MethodPost, path+"/complete", "member", "", true)
	if repeated.Code != http.StatusConflict {
		t.Fatalf("repeat: %d %s", repeated.Code, repeated.Body.String())
	}
	saved := fixture.request(t, http.MethodGet, path, "member", "", true)
	if !bytes.Contains(saved.Body.Bytes(), []byte(`"Status":"completed"`)) {
		t.Fatalf("saved state: %s", saved.Body.String())
	}
}

func TestOpenAPIRejectsInvalidResponseShapes(t *testing.T) {
	fixture := newContractFixture(t)
	request := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	route, parameters, err := fixture.router.FindRoute(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{`{"id":"wrong-casing"}`, `{"ID":42}`, `[]`} {
		input := &openapi3filter.ResponseValidationInput{RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request, PathParams: parameters, Route: route}, Status: 200, Header: http.Header{"Content-Type": []string{"application/json"}}}
		input.SetBodyBytes([]byte(payload))
		if err := openapi3filter.ValidateResponse(context.Background(), input); err == nil {
			t.Fatalf("invalid response accepted: %s", payload)
		}
	}
}

func TestOpenAPIChoreDefinitionLifecycleContract(t *testing.T) {
	fixture := newContractFixture(t)
	created := fixture.request(t, http.MethodPost, "/api/chore-definitions", "member", `{"name":"Bins","dueDate":"2026-01-01","recurrenceType":"daily","recurrenceCount":2}`, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("create definition: %d %s", created.Code, created.Body.String())
	}
	var definition struct {
		ID          string
		CurrentTask *models.Chore
	}
	if err := json.Unmarshal(created.Body.Bytes(), &definition); err != nil {
		t.Fatal(err)
	}
	if definition.ID == "" || definition.CurrentTask == nil {
		t.Fatalf("missing current chore: %s", created.Body.String())
	}
	definitionPath := "/api/chore-definitions/" + definition.ID
	updated := fixture.request(t, http.MethodPut, definitionPath, "member", `{"name":"Bins edited","dueDate":"2026-01-01","recurrenceType":"daily","recurrenceCount":2}`, true)
	if updated.Code != http.StatusOK {
		t.Fatalf("update: %d %s", updated.Code, updated.Body.String())
	}
	skipped := fixture.request(t, http.MethodPost, "/api/chores/"+definition.CurrentTask.ID+"/skip", "member", "", true)
	if skipped.Code != http.StatusNoContent {
		t.Fatalf("skip: %d %s", skipped.Code, skipped.Body.String())
	}
	history := fixture.request(t, http.MethodGet, "/api/chore-history?definition_id="+definition.ID+"&limit=1", "member", "", true)
	if history.Code != http.StatusOK {
		t.Fatalf("history: %d %s", history.Code, history.Body.String())
	}
}
