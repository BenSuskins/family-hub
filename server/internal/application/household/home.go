package household

import (
	"context"
	"strings"

	domain "github.com/bensuskins/family-hub/internal/domain/household"
)

type Settings interface {
	Load(context.Context, string) (string, bool, error)
	Set(context.Context, string, string) error
}
type HomeStore interface {
	LoadHome(context.Context, domain.Home) (domain.Home, error)
	SaveHome(context.Context, domain.Home) error
}
type Service struct {
	settings Settings
	homes    HomeStore
}

func New(settings Settings, homes HomeStore) *Service {
	return &Service{settings: settings, homes: homes}
}
func (service *Service) Home(ctx context.Context) (domain.Home, error) {
	defaults := domain.Home{ArrivalMarginMinutes: 10, Timezone: "Europe/London"}
	home, err := service.homes.LoadHome(ctx, defaults)
	return copyHome(home), err
}
func (service *Service) FamilyName(ctx context.Context) string {
	value, found, err := service.settings.Load(ctx, "family_name")
	if err != nil || !found || strings.TrimSpace(value) == "" {
		return "Family"
	}
	return value
}
func (service *Service) SaveHome(ctx context.Context, home domain.Home) (domain.Home, error) {
	home = copyHome(home)
	if err := domain.ValidateHome(home); err != nil {
		return domain.Home{}, err
	}
	if err := service.homes.SaveHome(ctx, copyHome(home)); err != nil {
		return domain.Home{}, err
	}
	return copyHome(home), nil
}
func copyHome(home domain.Home) domain.Home {
	if home.Latitude != nil {
		value := *home.Latitude
		home.Latitude = &value
	}
	if home.Longitude != nil {
		value := *home.Longitude
		home.Longitude = &value
	}
	return home
}
