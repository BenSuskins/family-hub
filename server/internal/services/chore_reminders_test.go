package services

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type fakeReminderSender struct {
	counts    []int
	summaries []ChoreReminderSummary
	sendError error
}

func (sender *fakeReminderSender) Send(ctx context.Context, device models.NotificationDevice, summary ChoreReminderSummary, deliveryID string, expires time.Time) error {
	sender.counts = append(sender.counts, summary.Count)
	sender.summaries = append(sender.summaries, summary)
	return sender.sendError
}

func TestChoreReminderSendsFamilyCountOncePerSlotAcrossRestart(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "reminders", Email: "reminders@example.com", Name: "Family", Role: models.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	settings := repository.NewSettingsRepository(database)
	if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:00","16:30"]}`); err != nil {
		t.Fatal(err)
	}
	devices := repository.NewNotificationRepository(database)
	if err := devices.Register(ctx, models.NotificationDevice{ID: "phone", UserID: user.ID, Token: "device-token", Environment: "sandbox"}); err != nil {
		t.Fatal(err)
	}
	chores := repository.NewChoreRepository(database)
	today := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)
	late := "23:00"
	for _, chore := range []models.Chore{
		{Name: "Today", DueDate: &today, DueTime: &late, Status: models.ChoreStatusPending},
		{Name: "Past pending", DueDate: &yesterday, Status: models.ChoreStatusPending},
		{Name: "Overdue", Status: models.ChoreStatusOverdue},
		{Name: "Done", DueDate: &today, Status: models.ChoreStatusCompleted},
		{Name: "Future", DueDate: &tomorrow, Status: models.ChoreStatusPending},
		{Name: "Undated", Status: models.ChoreStatusPending},
	} {
		chore.CreatedByUserID = user.ID
		chore.AssignedToUserID = &user.ID
		if _, err := chores.Create(ctx, chore); err != nil {
			t.Fatal(err)
		}
	}
	sender := &fakeReminderSender{}
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	for range 2 {
		service := NewChoreReminderService(settings, chores, devices, sender, users)
		if err := service.Check(ctx, now); err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.counts) != 1 || sender.counts[0] != 4 {
		t.Fatalf("notifications: %v", sender.counts)
	}
	service := NewChoreReminderService(settings, chores, devices, sender, users)
	if err := service.Check(ctx, now.Add(6*time.Hour+30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if sender.summaries[0].Tasks[0].Assignee != "Family" || sender.summaries[0].Tasks[0].Name == "" {
		t.Fatal("missing owner or chore name")
	}
	if len(sender.counts) != 2 {
		t.Fatalf("evening reminder missing: %v", sender.counts)
	}
}

func TestChoreReminderSkipsCompletedChoresOnRetryAndExpires(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "retry", Email: "retry@example.com", Name: "Retry", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	settings := repository.NewSettingsRepository(database)
	if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:00","16:30"]}`); err != nil {
		t.Fatal(err)
	}
	devices := repository.NewNotificationRepository(database)
	if err := devices.Register(ctx, models.NotificationDevice{ID: "phone", UserID: user.ID, Token: "token", Environment: "production"}); err != nil {
		t.Fatal(err)
	}
	chores := repository.NewChoreRepository(database)
	due := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	chore, err := chores.Create(ctx, models.Chore{Name: "Dishes", CreatedByUserID: user.ID, DueDate: &due, Status: models.ChoreStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	sender := &fakeReminderSender{sendError: &PushError{Reason: "unavailable", Retry: true}}
	service := NewChoreReminderService(settings, chores, devices, sender)
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	if err := service.Check(ctx, now); err == nil {
		t.Fatal("missing delivery error")
	}
	chore.Status = models.ChoreStatusCompleted
	if err := chores.Update(ctx, chore); err != nil {
		t.Fatal(err)
	}
	sender.sendError = nil
	if err := service.Check(ctx, now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(sender.counts) != 1 {
		t.Fatalf("completed chore sent again: %v", sender.counts)
	}
	chore.Status = models.ChoreStatusPending
	if err := chores.Update(ctx, chore); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(ctx, now.Add(5*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(sender.counts) != 1 {
		t.Fatal("expired reminder sent")
	}
}

func TestChoreReminderUsesHouseholdDateAndDaylightSaving(t *testing.T) {
	for _, example := range []struct{ name, timezone, instant string }{
		{"summer", "Europe/London", "2026-07-01T09:00:00Z"},
		{"winter", "Europe/London", "2026-12-01T10:00:00Z"},
		{"local date ahead", "Pacific/Auckland", "2026-10-17T21:00:00Z"},
		{"local date behind", "America/Los_Angeles", "2026-10-18T17:00:00Z"},
	} {
		t.Run(example.name, func(t *testing.T) {
			ctx := context.Background()
			database := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "timezone", Email: "timezone@example.com", Name: "Timezone", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			settings := repository.NewSettingsRepository(database)
			if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:00","16:30"]}`); err != nil {
				t.Fatal(err)
			}
			if err := settings.Set(ctx, "home_info", `{"timezone":"`+example.timezone+`"}`); err != nil {
				t.Fatal(err)
			}
			devices := repository.NewNotificationRepository(database)
			if err := devices.Register(ctx, models.NotificationDevice{ID: "phone", UserID: user.ID, Token: "token", Environment: "production"}); err != nil {
				t.Fatal(err)
			}
			now, _ := time.Parse(time.RFC3339, example.instant)
			location, _ := time.LoadLocation(example.timezone)
			local := now.In(location)
			date := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
			chores := repository.NewChoreRepository(database)
			if _, err := chores.Create(ctx, models.Chore{Name: "Today", CreatedByUserID: user.ID, DueDate: &date, Status: models.ChoreStatusPending}); err != nil {
				t.Fatal(err)
			}
			sender := &fakeReminderSender{}
			service := NewChoreReminderService(settings, chores, devices, sender)
			if err := service.Check(ctx, now.Add(-time.Second)); err != nil {
				t.Fatal(err)
			}
			if len(sender.counts) != 0 {
				t.Fatal("sent before scheduled time")
			}
			if err := service.Check(ctx, now); err != nil {
				t.Fatal(err)
			}
			if len(sender.counts) != 1 || sender.counts[0] != 1 {
				t.Fatalf("missing local-day reminder: %v", sender.counts)
			}
		})
	}
}

func TestChoreReminderSkipsDisabledAndEmptySchedules(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "empty", Email: "empty@example.com", Name: "Empty", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	settings := repository.NewSettingsRepository(database)
	devices := repository.NewNotificationRepository(database)
	if err := devices.Register(ctx, models.NotificationDevice{ID: "phone", UserID: user.ID, Token: "token", Environment: "sandbox"}); err != nil {
		t.Fatal(err)
	}
	chores := repository.NewChoreRepository(database)
	chore, err := chores.Create(ctx, models.Chore{Name: "Overdue", CreatedByUserID: user.ID, Status: models.ChoreStatusOverdue})
	if err != nil {
		t.Fatal(err)
	}
	sender := &fakeReminderSender{}
	service := NewChoreReminderService(settings, chores, devices, sender)
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	if err := service.Check(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(sender.counts) != 0 {
		t.Fatal("disabled schedule sent a notification")
	}
	chore.Status = models.ChoreStatusCompleted
	if err := chores.Update(ctx, chore); err != nil {
		t.Fatal(err)
	}
	if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:00","16:30"]}`); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(sender.counts) != 0 {
		t.Fatal("completed chores sent a notification")
	}
}

func TestChoreReminderRetriesOnlyFailedDevices(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "partial", Email: "partial@example.com", Name: "Partial", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	settings := repository.NewSettingsRepository(database)
	if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:00","16:30"]}`); err != nil {
		t.Fatal(err)
	}
	devices := repository.NewNotificationRepository(database)
	for _, identifier := range []string{"accepted", "retry", "invalid"} {
		if err := devices.Register(ctx, models.NotificationDevice{ID: identifier, UserID: user.ID, Token: identifier, Environment: "production"}); err != nil {
			t.Fatal(err)
		}
	}
	chores := repository.NewChoreRepository(database)
	if _, err := chores.Create(ctx, models.Chore{Name: "Overdue", CreatedByUserID: user.ID, Status: models.ChoreStatusOverdue}); err != nil {
		t.Fatal(err)
	}
	sender := &partialReminderSender{attempts: map[string]int{}}
	service := NewChoreReminderService(settings, chores, devices, sender)
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	if err := service.Check(ctx, now); err == nil {
		t.Fatal("missing failure")
	}
	if err := service.Check(ctx, now.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	if sender.attempts["accepted"] != 1 || sender.attempts["retry"] != 2 || sender.attempts["invalid"] != 1 {
		t.Fatalf("attempts: %v", sender.attempts)
	}
	remaining, err := devices.Devices(ctx)
	if err != nil || len(remaining) != 2 {
		t.Fatalf("invalid device retained: %v %v", remaining, err)
	}
	// Editing a previously sent slot must not repeat it on the same date.
	if err := settings.Set(ctx, "chore_reminders", `{"enabled":true,"times":["10:01","16:30"]}`); err != nil {
		t.Fatal(err)
	}
	if err := service.Check(ctx, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if sender.attempts["accepted"] != 1 || sender.attempts["retry"] != 2 {
		t.Fatalf("schedule edit repeated a reminder: %v", sender.attempts)
	}
}

type partialReminderSender struct{ attempts map[string]int }

func (sender *partialReminderSender) Send(ctx context.Context, device models.NotificationDevice, summary ChoreReminderSummary, identifier string, expires time.Time) error {
	sender.attempts[device.ID]++
	if device.ID == "retry" && sender.attempts[device.ID] == 1 {
		return &PushError{Reason: "unavailable", Retry: true}
	}
	if device.ID == "invalid" {
		return &PushError{Reason: "Unregistered", InvalidDevice: true}
	}
	return nil
}

func TestReminderGroupsAllChoresByAssignee(t *testing.T) {
	summary := ChoreReminderSummary{Count: 7, Tasks: []ChoreReminderTask{
		{Assignee: "Ben", Name: "Empty bins"},
		{Assignee: "Alex", Name: "Dishwasher"},
		{Assignee: "Ben", Name: "Vacuum"},
		{Assignee: "Alex", Name: "Wipe counters"},
		{Assignee: "Ben", Name: "Water plants"},
		{Assignee: "Alex", Name: "Feed cat"},
		{Assignee: "Ben", Name: "Take out rubbish"},
	}}
	message := summary.Message()
	if message.Body != "Ben: Empty bins, Vacuum, Water plants, Take out rubbish\nAlex: Dishwasher, Wipe counters, Feed cat" || message.URL != "familyhub://today" {
		t.Fatalf("unexpected reminder: %+v", message)
	}
}
