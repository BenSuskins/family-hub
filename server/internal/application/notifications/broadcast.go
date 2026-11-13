package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type PushMessage struct {
	Title    string
	Body     string
	URL      string
	ThreadID string
}

type MessageSender interface {
	SendMessage(context.Context, models.NotificationDevice, PushMessage, string, time.Time) error
}

type PushError struct {
	Reason        string
	InvalidDevice bool
	Retry         bool
}

func (failure *PushError) Error() string { return "push rejected: " + failure.Reason }

type DeliveryState string

const (
	DeliveryPending  DeliveryState = "pending"
	DeliverySending  DeliveryState = "sending"
	DeliveryAccepted DeliveryState = "accepted"
	DeliveryRetry    DeliveryState = "retry"
	DeliveryFailed   DeliveryState = "failed"
)

type BroadcastDelivery struct {
	ID     string
	Device models.NotificationDevice
	State  DeliveryState
}
type BroadcastStore interface {
	Prepare(context.Context, string, string, string, time.Time) (time.Time, error)
	Deliveries(context.Context, string) ([]BroadcastDelivery, error)
	Claim(context.Context, string, time.Time) (bool, error)
	Finish(context.Context, string, DeliveryState) error
}
type DeviceInvalidator interface {
	Invalidate(context.Context, models.NotificationDevice) error
}

type BroadcastService struct {
	broadcasts BroadcastStore
	devices    DeviceInvalidator
	sender     MessageSender
}

func NewBroadcastService(broadcasts BroadcastStore, devices DeviceInvalidator, sender MessageSender) *BroadcastService {
	return &BroadcastService{broadcasts: broadcasts, devices: devices, sender: sender}
}

func (service *BroadcastService) Send(ctx context.Context, identifier, userID, message string, now time.Time) (models.BroadcastResult, error) {
	expires, err := service.broadcasts.Prepare(ctx, identifier, userID, message, now)
	if err != nil {
		return models.BroadcastResult{}, err
	}
	deliveries, err := service.broadcasts.Deliveries(ctx, identifier)
	if err != nil {
		return models.BroadcastResult{}, err
	}
	for _, delivery := range deliveries {
		if !now.Before(expires) || delivery.Device.ID == "" {
			continue
		}
		claimed, err := service.broadcasts.Claim(ctx, delivery.ID, now)
		if err != nil {
			return models.BroadcastResult{}, err
		}
		if !claimed {
			continue
		}
		err = service.sender.SendMessage(ctx, delivery.Device, PushMessage{Title: "Family message", Body: message, URL: "familyhub://today", ThreadID: "family-broadcasts"}, delivery.ID, expires)
		state := DeliveryAccepted
		if err != nil {
			state = DeliveryRetry
			var failure *PushError
			if errors.As(err, &failure) {
				if !failure.Retry {
					state = DeliveryFailed
				}
				if failure.InvalidDevice {
					if err := service.devices.Invalidate(ctx, delivery.Device); err != nil {
						return models.BroadcastResult{}, err
					}
				}
			}
		}
		if err := service.broadcasts.Finish(ctx, delivery.ID, state); err != nil {
			return models.BroadcastResult{}, err
		}
	}
	deliveries, err = service.broadcasts.Deliveries(ctx, identifier)
	if err != nil {
		return models.BroadcastResult{}, err
	}
	result := models.BroadcastResult{}
	for _, delivery := range deliveries {
		if delivery.State == DeliveryAccepted {
			result.Accepted++
		} else {
			result.Failed++
		}
	}
	return result, nil
}
