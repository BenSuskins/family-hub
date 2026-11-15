package displays

import (
	"context"
	"errors"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

var ErrDisplayNotFound = errors.New("display not found")

type Display struct {
	ID, Name, PairedByUserID string
	CreatedAt                time.Time
	EnabledSections          []string
	CommutePreferences       models.DisplayCommutePreferences
}
type SettingsStore interface {
	List(context.Context) ([]Display, error)
	FindByID(context.Context, string) (Display, error)
	FindByTokenID(context.Context, string) (Display, error)
	UpdateSections(context.Context, string, []string) error
	UpdateDetails(context.Context, string, string) error
	UpdateCommutePreferences(context.Context, string, models.DisplayCommutePreferences) error
	Revoke(context.Context, string) error
}
type Settings struct{ store SettingsStore }

func NewSettings(store SettingsStore) *Settings { return &Settings{store: store} }
func (service *Settings) List(ctx context.Context) ([]Display, error) {
	items, err := service.store.List(ctx)
	if items != nil {
		copied := make([]Display, len(items))
		for i, item := range items {
			copied[i] = copyDisplay(item)
		}
		items = copied
	}
	return items, err
}
func (service *Settings) FindByID(ctx context.Context, id string) (Display, error) {
	item, err := service.store.FindByID(ctx, id)
	return copyDisplay(item), err
}
func (service *Settings) FindByTokenID(ctx context.Context, id string) (Display, error) {
	item, err := service.store.FindByTokenID(ctx, id)
	return copyDisplay(item), err
}
func (service *Settings) UpdateSections(ctx context.Context, id string, sections []string) error {
	return service.store.UpdateSections(ctx, id, copyStrings(sections))
}
func (service *Settings) UpdateDetails(ctx context.Context, id, name string) error {
	return service.store.UpdateDetails(ctx, id, name)
}
func (service *Settings) UpdateCommutePreferences(ctx context.Context, id string, p models.DisplayCommutePreferences) error {
	return service.store.UpdateCommutePreferences(ctx, id, copyPreferences(p))
}
func (service *Settings) Revoke(ctx context.Context, id string) error {
	return service.store.Revoke(ctx, id)
}
func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}
func copyPreferences(p models.DisplayCommutePreferences) models.DisplayCommutePreferences {
	if p.ToWork.Weekdays != nil {
		p.ToWork.Weekdays = append([]int{}, p.ToWork.Weekdays...)
	}
	if p.ToHome.Weekdays != nil {
		p.ToHome.Weekdays = append([]int{}, p.ToHome.Weekdays...)
	}
	return p
}
func copyDisplay(item Display) Display {
	item.EnabledSections = copyStrings(item.EnabledSections)
	item.CommutePreferences = copyPreferences(item.CommutePreferences)
	return item
}
