package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	identityapp "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/handlers"
	"github.com/bensuskins/family-hub/internal/repository"
)

func TestTokenAdministrationHTTPContract(t *testing.T) {
	fixture := newContractFixture(t)
	for _, test := range []struct {
		token  string
		status int
	}{{"", http.StatusUnauthorized}, {"member", http.StatusForbidden}} {
		response := fixture.request(t, http.MethodPost, "/api/tokens", test.token, `{"name":"Tablet"}`, true)
		if response.Code != test.status {
			t.Fatalf("create with %q: %d %s", test.token, response.Code, response.Body.String())
		}
	}
	response := fixture.request(t, http.MethodPost, "/api/tokens", "admin", `{"name":"Tablet"}`, true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var created map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if len(created) != 3 || created["id"] == "" || created["name"] != "Tablet" || len(created["token"]) != 64 {
		t.Fatalf("create fields=%v", created)
	}
	listed := fixture.request(t, http.MethodGet, "/api/tokens", "admin", "", true)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(listed.Body.Bytes(), &records); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, record := range records {
		var id string
		if err := json.Unmarshal(record["ID"], &id); err != nil {
			t.Fatalf("missing uppercase ID: %v", err)
		}
		if id != created["id"] {
			continue
		}
		found = true
		if len(record) != 7 {
			t.Fatalf("token fields=%v", record)
		}
		for _, key := range []string{"ID", "Name", "TokenHash", "Scope", "CreatedByUserID", "ExpiresAt", "CreatedAt"} {
			if _, ok := record[key]; !ok {
				t.Fatalf("missing %s: %v", key, record)
			}
		}
		var hash, scope string
		if err := json.Unmarshal(record["TokenHash"], &hash); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(record["Scope"], &scope); err != nil {
			t.Fatal(err)
		}
		if hash != repository.HashToken(created["token"]) || scope != "api" {
			t.Fatalf("hash=%q scope=%q", hash, scope)
		}
	}
	if !found {
		t.Fatalf("created token absent from list: %s", listed.Body.String())
	}
	denied := fixture.request(t, http.MethodGet, "/api/tokens", "member", "", true)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("member list: %d %s", denied.Code, denied.Body.String())
	}
}

type emptyManagedTokenStore struct{}

func (emptyManagedTokenStore) CreateManagedToken(context.Context, identityapp.TokenRecord) (identityapp.TokenRecord, error) {
	return identityapp.TokenRecord{}, nil
}
func (emptyManagedTokenStore) ListManagedTokens(context.Context) ([]identityapp.TokenRecord, error) {
	return nil, nil
}
func (emptyManagedTokenStore) DeleteManagedToken(context.Context, string) error { return nil }

func TestEmptyTokenListJSON(t *testing.T) {
	service := identityapp.NewTokenAdministration(emptyManagedTokenStore{}, nil, nil)
	handler := handlers.NewAPIHandler(nil, nil, nil, service, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "")
	response := httptest.NewRecorder()
	handler.ListTokens(response, httptest.NewRequest(http.MethodGet, "/api/tokens", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %s", response.Code, response.Body.String())
	}
}
