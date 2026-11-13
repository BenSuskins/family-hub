package notifications_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/notifications"
	"github.com/bensuskins/family-hub/internal/models"
)

type broadcastFake struct {
	events     []string
	delivery   app.BroadcastDelivery
	prepareErr error
}

func (f *broadcastFake) Prepare(_ context.Context, _, _, _ string, now time.Time) (time.Time, error) {
	f.events = append(f.events, "prepare")
	return now.Add(time.Minute), f.prepareErr
}
func (f *broadcastFake) Deliveries(context.Context, string) ([]app.BroadcastDelivery, error) {
	return []app.BroadcastDelivery{f.delivery}, nil
}
func (f *broadcastFake) Claim(context.Context, string, time.Time) (bool, error) {
	f.events = append(f.events, "claim")
	return true, nil
}
func (f *broadcastFake) Finish(_ context.Context, _ string, state app.DeliveryState) error {
	f.events = append(f.events, "finish")
	f.delivery.State = state
	return nil
}
func (f *broadcastFake) Invalidate(context.Context, models.NotificationDevice) error {
	f.events = append(f.events, "invalidate")
	return nil
}

type senderFake struct {
	store *broadcastFake
	err   error
}

func (f senderFake) SendMessage(_ context.Context, _ models.NotificationDevice, _ app.PushMessage, _ string, _ time.Time) error {
	f.store.events = append(f.store.events, "send")
	return f.err
}
func TestBroadcastClaimsBeforeSendingAndFinishesAfter(t *testing.T) {
	store := &broadcastFake{delivery: app.BroadcastDelivery{ID: "delivery", Device: models.NotificationDevice{ID: "device"}}}
	service := app.NewBroadcastService(store, store, senderFake{store: store})
	result, err := service.Send(context.Background(), "request", "user", "hello", time.Now())
	if err != nil || result.Accepted != 1 || len(store.events) != 4 || store.events[0] != "prepare" || store.events[1] != "claim" || store.events[2] != "send" || store.events[3] != "finish" {
		t.Fatalf("result=%+v events=%v err=%v", result, store.events, err)
	}
}
func TestBroadcastDoesNotSendWhenPrepareFails(t *testing.T) {
	store := &broadcastFake{prepareErr: errors.New("commit failed")}
	service := app.NewBroadcastService(store, store, senderFake{store: store})
	_, err := service.Send(context.Background(), "request", "user", "hello", time.Now())
	if err == nil || len(store.events) != 1 {
		t.Fatalf("events=%v error=%v", store.events, err)
	}
}

func TestBroadcastInvalidDeviceIsRemovedBeforeFinishing(t *testing.T) {
	store := &broadcastFake{delivery: app.BroadcastDelivery{ID: "delivery", Device: models.NotificationDevice{ID: "device"}}}
	sender := senderFake{store: store, err: &app.PushError{Reason: "invalid", InvalidDevice: true}}
	result, err := app.NewBroadcastService(store, store, sender).Send(context.Background(), "request", "user", "hello", time.Now())
	if err != nil || result.Failed != 1 || store.delivery.State != app.DeliveryFailed || len(store.events) != 5 || store.events[3] != "invalidate" || store.events[4] != "finish" {
		t.Fatalf("result=%+v state=%q events=%v err=%v", result, store.delivery.State, store.events, err)
	}
}
