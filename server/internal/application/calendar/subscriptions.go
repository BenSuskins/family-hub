package calendar

import (
	"context"
	"errors"
	"fmt"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

var (
	ErrNotFound    = errors.New("calendar not found")
	ErrReadCreated = errors.New("failed to read created calendar")
	ErrWrite       = errors.New("failed to write calendar")
)

type SubscriptionStore interface {
	FindAll(context.Context) ([]models.ICalSubscription, error)
	FindByID(context.Context, string) (models.ICalSubscription, error)
	Create(context.Context, models.ICalSubscription) error
	UpdateColor(context.Context, string, string) error
	Delete(context.Context, string) error
}
type Refresher interface {
	ForceRefreshByID(context.Context, string) error
}
type Subscriptions struct {
	store     SubscriptionStore
	refresher Refresher
	newID     func() string
}

func NewSubscriptions(store SubscriptionStore, refresher Refresher, newID func() string) *Subscriptions {
	if newID == nil {
		newID = uuid.NewString
	}
	return &Subscriptions{store: store, refresher: refresher, newID: newID}
}
func (service *Subscriptions) List(ctx context.Context) ([]models.ICalSubscription, error) {
	items, err := service.store.FindAll(ctx)
	if items != nil {
		copied := make([]models.ICalSubscription, len(items))
		for i, item := range items {
			copied[i] = copySubscription(item)
		}
		items = copied
	}
	return items, err
}
func (service *Subscriptions) Create(ctx context.Context, name, url, color string) (models.ICalSubscription, error) {
	item := models.ICalSubscription{ID: service.newID(), Name: name, URL: url, Color: color}
	if err := service.store.Create(ctx, item); err != nil {
		return models.ICalSubscription{}, fmt.Errorf("%w: %w", ErrWrite, err)
	}
	result, err := service.store.FindByID(ctx, item.ID)
	if err != nil {
		return models.ICalSubscription{}, fmt.Errorf("%w: %w", ErrReadCreated, err)
	}
	return copySubscription(result), nil
}
func (service *Subscriptions) UpdateColor(ctx context.Context, id, color string) (models.ICalSubscription, error) {
	item, err := service.store.FindByID(ctx, id)
	if err != nil {
		return models.ICalSubscription{}, err
	}
	if err = service.store.UpdateColor(ctx, item.ID, color); err != nil {
		return models.ICalSubscription{}, fmt.Errorf("%w: %w", ErrWrite, err)
	}
	item.Color = color
	return copySubscription(item), nil
}
func (service *Subscriptions) Delete(ctx context.Context, id string) error {
	if _, err := service.store.FindByID(ctx, id); err != nil {
		return err
	}
	if err := service.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %w", ErrWrite, err)
	}
	return nil
}
func (service *Subscriptions) Refresh(ctx context.Context, id string) error {
	return service.refresher.ForceRefreshByID(ctx, id)
}

func copySubscription(item models.ICalSubscription) models.ICalSubscription {
	if item.CachedData != nil {
		value := *item.CachedData
		item.CachedData = &value
	}
	if item.LastFetchedAt != nil {
		value := *item.LastFetchedAt
		item.LastFetchedAt = &value
	}
	return item
}
