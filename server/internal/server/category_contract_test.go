package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestOpenAPICategoryAdministrationContract(t *testing.T) {
	fixture := newContractFixture(t)
	unauthenticated := fixture.request(t, http.MethodGet, "/api/categories", "", "", false)
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous list: %d %s", unauthenticated.Code, unauthenticated.Body.String())
	}
	forbidden := fixture.request(t, http.MethodPost, "/api/categories", "member", `{"name":"Kitchen"}`, true)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("member create: %d %s", forbidden.Code, forbidden.Body.String())
	}
	created := fixture.request(t, http.MethodPost, "/api/categories", "admin", `{"name":"Kitchen"}`, true)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(created.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if len(data) != 4 {
		t.Fatalf("unexpected category fields: %s", created.Body.String())
	}
	for _, key := range []string{"ID", "Name", "CreatedByUserID", "CreatedAt"} {
		if _, ok := data[key]; !ok {
			t.Fatalf("missing %s: %s", key, created.Body.String())
		}
	}
	var id string
	if err := json.Unmarshal(data["ID"], &id); err != nil || id == "" {
		t.Fatalf("id: %s, %v", created.Body.String(), err)
	}
	path := "/api/categories/" + id
	listed := fixture.request(t, http.MethodGet, "/api/categories", "member", "", true)
	if listed.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listed.Code, listed.Body.String())
	}
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list: %s, %v", listed.Body.String(), err)
	}
	updated := fixture.request(t, http.MethodPut, path, "admin", `{"name":"Garden"}`, true)
	if updated.Code != http.StatusOK {
		t.Fatalf("update: %d %s", updated.Code, updated.Body.String())
	}
	if err := json.Unmarshal(updated.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := json.Unmarshal(data["Name"], &name); err != nil || name != "Garden" {
		t.Fatalf("updated name: %s, %v", updated.Body.String(), err)
	}
	missing := fixture.request(t, http.MethodPut, "/api/categories/missing", "admin", `{"name":"Missing"}`, true)
	if missing.Code != http.StatusInternalServerError {
		t.Fatalf("missing update: %d %s", missing.Code, missing.Body.String())
	}
	var failure map[string]string
	if err := json.Unmarshal(missing.Body.Bytes(), &failure); err != nil || failure["error"] != "failed to reload category" {
		t.Fatalf("missing update error: %s, %v", missing.Body.String(), err)
	}
	deleted := fixture.request(t, http.MethodDelete, path, "admin", "", true)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", deleted.Code, deleted.Body.String())
	}
	listed = fixture.request(t, http.MethodGet, "/api/categories", "member", "", true)
	if listed.Code != http.StatusOK || listed.Body.String() != "null\n" {
		t.Fatalf("empty list: %d %s", listed.Code, listed.Body.String())
	}
}
