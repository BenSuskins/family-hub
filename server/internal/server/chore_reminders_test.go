package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestReminderRoutesAndDeviceOwnership(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	tokens := repository.NewAPITokenRepository(database)
	for _, role := range []models.Role{models.RoleAdmin, models.RoleMember} {
		user, err := users.Create(context.Background(), models.User{OIDCSubject: string(role), Email: string(role) + "@example.com", Name: string(role), Role: role})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tokens.Create(context.Background(), models.APIToken{Name: "test", TokenHash: repository.HashToken(string(role)), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID}); err != nil {
			t.Fatal(err)
		}
	}
	server := New(database, config.Config{}, nil)
	call := func(method, path, body, token string, status int) []byte {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		response := httptest.NewRecorder()
		server.router.ServeHTTP(response, request)
		if response.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, response.Code, status, response.Body.String())
		}
		return response.Body.Bytes()
	}
	call("GET", "/api/chore-reminders", "", "invalid", http.StatusUnauthorized)
	var settings models.ChoreReminders
	if err := json.Unmarshal(call("GET", "/api/chore-reminders", "", "member", 200), &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Enabled || settings.Times != [2]string{"10:00", "16:30"} || settings.Timezone != "Europe/London" {
		t.Fatalf("defaults: %+v", settings)
	}
	call("PATCH", "/api/chore-reminders", `{"enabled":false,"times":["09:00","18:00"]}`, "member", 403)
	for _, body := range []string{`{"times":["25:00","16:30"]}`, `{"times":["10:00","10:00"]}`, `{"times":["10:00"]}`, `{"times":["10:00","16:30","18:00"]}`} {
		call("PATCH", "/api/chore-reminders", body, "admin", 400)
	}
	call("PATCH", "/api/chore-reminders", `{"enabled":false,"times":["09:00","18:00"]}`, "admin", 200)
	// Enablement reports absent push credentials, rather than pretending alerts work.
	call("PATCH", "/api/chore-reminders", `{"enabled":true,"times":["10:00","16:30"]}`, "admin", 503)
	device := `{"id":"phone","token":"` + strings.Repeat("ab", 32) + `","environment":"sandbox"}`
	call("PUT", "/api/notification-devices/phone", device, "member", 204)
	call("PUT", "/api/notification-devices/phone", device, "admin", 409)
	call("DELETE", "/api/notification-devices/phone", "", "admin", 204)
	registered, err := repository.NewNotificationRepository(database).Devices(context.Background())
	if err != nil || len(registered) != 1 {
		t.Fatalf("another user deleted a device: %v %v", registered, err)
	}
	call("DELETE", "/api/notification-devices/phone", "", "member", 204)
	registered, err = repository.NewNotificationRepository(database).Devices(context.Background())
	if err != nil || len(registered) != 0 {
		t.Fatalf("device remains: %v %v", registered, err)
	}
	call("PUT", "/api/notification-devices/phone", `{"token":"bad","environment":"production"}`, "member", 400)
}

func TestRevokingAPITokenRemovesItsNotificationDevice(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "revoke", Email: "revoke@example.com", Name: "Revoke", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	tokens := repository.NewAPITokenRepository(database)
	token, err := tokens.Create(ctx, models.APIToken{Name: "phone", TokenHash: repository.HashToken("phone-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	server := New(database, config.Config{}, nil)
	request := httptest.NewRequest("PUT", "/api/notification-devices/phone", strings.NewReader(`{"token":"`+strings.Repeat("ab", 32)+`","environment":"sandbox"}`))
	request.Header.Set("Authorization", "Bearer phone-token")
	response := httptest.NewRecorder()
	server.router.ServeHTTP(response, request)
	if response.Code != 204 {
		t.Fatalf("registration: %d %s", response.Code, response.Body.String())
	}
	devices := repository.NewNotificationRepository(database)
	registered, err := devices.Devices(ctx)
	if err != nil || len(registered) != 1 {
		t.Fatalf("registration missing: %v %v", registered, err)
	}
	if err := tokens.Delete(ctx, token.ID); err != nil {
		t.Fatal(err)
	}
	registered, err = devices.Devices(ctx)
	if err != nil || len(registered) != 0 {
		t.Fatalf("revoked token still receives alerts: %v %v", registered, err)
	}
}
