package contracttests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type broadcastStoreFake struct {
	mutex          sync.Mutex
	owner, message string
	created        time.Time
	delivery       app.BroadcastDelivery
}

func (f *broadcastStoreFake) Prepare(_ context.Context, _, owner, message string, now time.Time) (time.Time, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.owner == "" {
		f.owner = owner
		f.message = message
		f.created = now
		f.delivery = app.BroadcastDelivery{ID: "delivery", Device: models.NotificationDevice{ID: "device"}, State: app.DeliveryPending}
	}
	if f.owner != owner || f.message != message {
		return time.Time{}, models.ErrBroadcastConflict
	}
	return f.created.Add(5 * time.Minute), nil
}
func (f *broadcastStoreFake) Deliveries(context.Context, string) ([]app.BroadcastDelivery, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return []app.BroadcastDelivery{f.delivery}, nil
}
func (f *broadcastStoreFake) Claim(_ context.Context, _ string, _ time.Time) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if f.delivery.State != app.DeliveryPending && f.delivery.State != app.DeliveryRetry {
		return false, nil
	}
	f.delivery.State = app.DeliverySending
	return true, nil
}
func (f *broadcastStoreFake) Finish(_ context.Context, _ string, state app.DeliveryState) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.delivery.State = state
	return nil
}
func TestBroadcastStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.BroadcastStore, string){
		"fake": func(*testing.T) (app.BroadcastStore, string) { return &broadcastStoreFake{}, "user" },
		"sqlite": func(t *testing.T) (app.BroadcastStore, string) {
			database := testutil.NewTestDatabase(t)
			ctx := context.Background()
			user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "broadcast-contract", Email: "broadcast@example.com", Name: "Admin", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			if err := repository.NewNotificationRepository(database).Register(ctx, models.NotificationDevice{ID: "device", UserID: user.ID, Token: "token", Environment: "sandbox"}); err != nil {
				t.Fatal(err)
			}
			return repository.NewBroadcastRepository(database), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, user := factory(t)
			ctx := context.Background()
			now := time.Now().UTC().Truncate(time.Second)
			expires, err := store.Prepare(ctx, "broadcast", user, "hello", now)
			if err != nil || !expires.Equal(now.Add(5*time.Minute)) {
				t.Fatalf("prepare expires=%s err=%v", expires, err)
			}
			later, err := store.Prepare(ctx, "broadcast", user, "hello", now.Add(time.Minute))
			if err != nil || !later.Equal(expires) {
				t.Fatalf("retry expires=%s err=%v", later, err)
			}
			if _, err := store.Prepare(ctx, "broadcast", user, "different", now); !errors.Is(err, models.ErrBroadcastConflict) {
				t.Fatalf("conflict=%v", err)
			}
			deliveries, err := store.Deliveries(ctx, "broadcast")
			if err != nil || len(deliveries) != 1 || deliveries[0].Device.ID != "device" {
				t.Fatalf("deliveries=%+v err=%v", deliveries, err)
			}
			const count = 12
			results := make(chan bool, count)
			errs := make(chan error, count)
			start := make(chan struct{})
			for range count {
				go func() {
					<-start
					claimed, err := store.Claim(ctx, deliveries[0].ID, now)
					results <- claimed
					errs <- err
				}()
			}
			close(start)
			wins := 0
			for range count {
				if <-results {
					wins++
				}
				if err := <-errs; err != nil {
					t.Fatal(err)
				}
			}
			if wins != 1 {
				t.Fatalf("claims=%d", wins)
			}
			if err := store.Finish(ctx, deliveries[0].ID, app.DeliveryRetry); err != nil {
				t.Fatal(err)
			}
			claimed, err := store.Claim(ctx, deliveries[0].ID, now)
			if err != nil || !claimed {
				t.Fatalf("retry claimed=%v err=%v", claimed, err)
			}
		})
	}
}
