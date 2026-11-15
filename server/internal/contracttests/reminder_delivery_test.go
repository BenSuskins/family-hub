package contracttests

import (
	"context"
	"sync"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type reminderDeliveryFake struct {
	mutex           sync.Mutex
	state, id, date string
	lease           time.Time
}

func (f *reminderDeliveryFake) Devices(context.Context) ([]models.NotificationDevice, error) {
	return []models.NotificationDevice{{ID: "device"}}, nil
}
func (f *reminderDeliveryFake) Claim(_ context.Context, _, date string, _ int, now time.Time) (string, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.state == "finished" || f.state == "sending" && now.Before(f.lease) {
		return "", nil
	}
	if f.id == "" {
		f.id = "delivery"
	}
	f.date = date
	f.state = "sending"
	f.lease = now.Add(time.Minute)
	return f.id, nil
}
func (f *reminderDeliveryFake) Finish(_ context.Context, _ string, retry bool) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.state = "finished"
	if retry {
		f.state = "retry"
	}
	return nil
}
func (f *reminderDeliveryFake) Invalidate(context.Context, models.NotificationDevice) error {
	return nil
}
func (f *reminderDeliveryFake) Prune(_ context.Context, before string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.date < before {
		f.state = ""
		f.id = ""
	}
	return nil
}
func TestReminderDeliveryStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) app.ReminderDevices{
		"fake": func(*testing.T) app.ReminderDevices { return &reminderDeliveryFake{} },
		"sqlite": func(t *testing.T) app.ReminderDevices {
			db := testutil.NewTestDatabase(t)
			ctx := context.Background()
			user, err := repository.NewUserRepository(db).Create(ctx, models.User{OIDCSubject: "reminder-contract", Email: "reminder@example.com", Name: "Admin", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			store := repository.NewNotificationRepository(db)
			if err := store.Register(ctx, models.NotificationDevice{ID: "device", UserID: user.ID, Token: "token", Environment: "sandbox"}); err != nil {
				t.Fatal(err)
			}
			return store
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store := factory(t)
			ctx := context.Background()
			now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
			const attempts = 12
			results := make(chan string, attempts)
			errs := make(chan error, attempts)
			start := make(chan struct{})
			for range attempts {
				go func() {
					<-start
					id, err := store.Claim(ctx, "device", "2026-01-01", 0, now)
					results <- id
					errs <- err
				}()
			}
			close(start)
			wins := 0
			identifier := ""
			for range attempts {
				if id := <-results; id != "" {
					wins++
					identifier = id
				}
				if err := <-errs; err != nil {
					t.Fatal(err)
				}
			}
			if wins != 1 {
				t.Fatalf("claims=%d", wins)
			}
			if err := store.Finish(ctx, identifier, true); err != nil {
				t.Fatal(err)
			}
			retry, err := store.Claim(ctx, "device", "2026-01-01", 0, now)
			if err != nil || retry != identifier {
				t.Fatalf("retry=%q want %q err=%v", retry, identifier, err)
			}
			if err := store.Finish(ctx, identifier, false); err != nil {
				t.Fatal(err)
			}
			finished, err := store.Claim(ctx, "device", "2026-01-01", 0, now)
			if err != nil || finished != "" {
				t.Fatalf("finished=%q err=%v", finished, err)
			}
			if err := store.Prune(ctx, "2026-01-02"); err != nil {
				t.Fatal(err)
			}
			fresh, err := store.Claim(ctx, "device", "2026-01-01", 0, now)
			if err != nil || fresh == "" {
				t.Fatalf("after prune=%q err=%v", fresh, err)
			}
		})
	}
}
