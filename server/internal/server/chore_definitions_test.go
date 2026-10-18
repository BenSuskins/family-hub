package server

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestChoreDefinitionsSeparateSetupTasksAndHistory(t *testing.T) {
	connection := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(connection).Create(context.Background(), models.User{OIDCSubject: "chores", Email: "chores@x", Name: "Ben", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.NewAPITokenRepository(connection).Create(context.Background(), models.APIToken{Name: "test", TokenHash: repository.HashToken("chores"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	server := New(connection, config.Config{}, nil)
	call := func(method, path, body string, status int) []byte {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer chores")
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, response.Code, response.Body.String())
		}
		return response.Body.Bytes()
	}
	var definition services.ChoreDefinition
	if err := json.Unmarshal(call("POST", "/api/chore-definitions", `{"name":"Bins","dueDate":"2020-01-01","recurrenceType":"daily","recurrenceCount":2}`, 201), &definition); err != nil {
		t.Fatal(err)
	}
	var tasks []models.Chore
	json.Unmarshal(call("GET", "/api/chores", "", 200), &tasks)
	if len(tasks) != 1 || definition.CurrentTask == nil {
		t.Fatalf("expected one task: %+v", tasks)
	}
	owner := *definition.CurrentTask.AssignedToUserID
	call("PUT", "/api/chore-definitions/"+definition.ID, `{"name":"Empty bins","dueDate":"2030-01-01","recurrenceType":"daily","recurrenceCount":2}`, 200)
	json.Unmarshal(call("GET", "/api/chore-definitions/"+definition.ID, "", 200), &definition)
	if definition.CurrentTask.Name != "Empty bins" || definition.CurrentTask.DueDate.Format("2006-01-02") != "2020-01-01" || *definition.CurrentTask.AssignedToUserID != owner {
		t.Fatalf("edit changed task ownership or date: %+v", definition)
	}
	call("POST", "/api/chores/"+definition.CurrentTask.ID+"/skip", "", 204)
	call("POST", "/api/chores/"+definition.CurrentTask.ID+"/complete", "", 409)
	call("GET", "/api/chores/"+definition.CurrentTask.ID, "", 409)
	json.Unmarshal(call("GET", "/api/chore-definitions/"+definition.ID, "", 200), &definition)
	if definition.CurrentTask == nil {
		t.Fatal("skip did not create successor")
	}
	call("POST", "/api/chores/"+definition.CurrentTask.ID+"/complete", "", 204)
	var active []services.ChoreDefinition
	json.Unmarshal(call("GET", "/api/chore-definitions", "", 200), &active)
	if len(active) != 0 {
		t.Fatal("finished definition remains active")
	}
	var history struct {
		Items      []models.Chore
		NextCursor string
	}
	json.Unmarshal(call("GET", "/api/chore-history?definition_id="+definition.ID+"&limit=1", "", 200), &history)
	if len(history.Items) != 1 || history.NextCursor == "" {
		t.Fatalf("history pagination: %+v", history)
	}
	json.Unmarshal(call("GET", "/api/chore-history?definition_id="+definition.ID+"&cursor="+history.NextCursor, "", 200), &history)
	if len(history.Items) != 1 {
		t.Fatal("missing second history page")
	}
	call("DELETE", "/api/chore-definitions/"+definition.ID, "", 204)
	call("GET", "/api/chore-history?definition_id="+definition.ID, "", 200)
	call("GET", "/api/chore-history?limit=0", "", 400)
	call("POST", "/api/chore-definitions", `{"name":"One off"}`, 201)
}
