package services

import (
	"context"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
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

type BroadcastService struct {
	broadcasts repository.BroadcastRepository
	devices    repository.NotificationRepository
	sender     MessageSender
}

func NewBroadcastService(broadcasts repository.BroadcastRepository, devices repository.NotificationRepository, sender MessageSender) *BroadcastService {
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
		state := "accepted"
		if err != nil {
			state = "retry"
			var failure *PushError
			if errors.As(err, &failure) {
				if !failure.Retry {
					state = "failed"
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
		if delivery.State == "accepted" {
			result.Accepted++
		} else {
			result.Failed++
		}
	}
	return result, nil
}
