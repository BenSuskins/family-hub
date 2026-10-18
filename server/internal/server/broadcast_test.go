package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

type fakeBroadcastSender struct {
	messages []services.PushMessage
	attempts map[string]int
}

func (sender *fakeBroadcastSender) SendMessage(ctx context.Context, device models.NotificationDevice, message services.PushMessage, id string, expires time.Time) error {
	sender.messages = append(sender.messages, message)
	sender.attempts[device.ID]++
	if device.ID == "member" && sender.attempts[device.ID] == 1 {
		return &services.PushError{Reason: "unavailable", Retry: true}
	}
	return nil
}

func TestAdminBroadcastIsValidatedAndDoesNotRepeatOnRetry(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	for _, role := range []models.Role{models.RoleAdmin, models.RoleMember} {
		user, err := users.Create(ctx, models.User{OIDCSubject: string(role), Email: string(role) + "@example.com", Name: string(role), Role: role})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tokens.Create(ctx, models.APIToken{Name: "test", TokenHash: repository.HashToken(string(role)), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID}); err != nil {
			t.Fatal(err)
		}
		if err := repository.NewNotificationRepository(database).Register(ctx, models.NotificationDevice{ID: string(role), UserID: user.ID, Token: string(role), Environment: "sandbox"}); err != nil {
			t.Fatal(err)
		}
	}
	sender := &fakeBroadcastSender{attempts: map[string]int{}}
	server := New(database, config.Config{}, nil, sender)
	call := func(body, token string, status int) models.BroadcastResult {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/api/notifications/broadcast", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("got %d want %d: %s", response.Code, status, response.Body.String())
		}
		var result models.BroadcastResult
		if status == 200 {
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
		}
		return result
	}
	identifier := uuid.NewString()
	body := `{"requestID":"` + identifier + `","message":"Dinner is ready."}`
	call(body, "member", 403)
	call(`{"requestID":"`+identifier+`","message":"  "}`, "admin", 400)
	for attempt := range 2 {
		result := call(body, "admin", 200)
		if result.Accepted != attempt+1 || result.Failed != 1-attempt {
			t.Fatalf("result: %+v", result)
		}
	}
	if len(sender.messages) != 3 || sender.attempts["admin"] != 1 || sender.attempts["member"] != 2 || sender.messages[0].Body != "Dinner is ready." {
		t.Fatalf("messages: %v", sender.messages)
	}
	call(`{"requestID":"`+identifier+`","message":"Different message"}`, "admin", 409)
}
