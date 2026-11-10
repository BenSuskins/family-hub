package household

import (
	"context"
	"encoding/json"
	"strings"

	domain "github.com/bensuskins/family-hub/internal/domain/household"
	"github.com/bensuskins/family-hub/internal/models"
)

type Settings interface {
	Load(context.Context, string) (string, bool, error)
	Set(context.Context, string, string) error
}
type Service struct{ settings Settings }

func New(settings Settings) *Service { return &Service{settings: settings} }
func (service *Service) Home(ctx context.Context) (models.HomeInfo, error) {
	home := models.HomeInfo{ArrivalMarginMinutes: 10, Timezone: "Europe/London"}
	value, found, err := service.settings.Load(ctx, "home_info")
	if err != nil {
		return home, err
	}
	if found {
		err = json.Unmarshal([]byte(value), &home)
	}
	return home, err
}
func (service *Service) FamilyName(ctx context.Context) string {
	value, found, err := service.settings.Load(ctx, "family_name")
	if err != nil || !found || strings.TrimSpace(value) == "" {
		return "Family"
	}
	return value
}
func (service *Service) SaveHome(ctx context.Context, home models.HomeInfo) (models.HomeInfo, error) {
	if err := domain.ValidateHome(home); err != nil {
		return models.HomeInfo{}, err
	}
	value, err := json.Marshal(home)
	if err != nil {
		return models.HomeInfo{}, err
	}
	if err := service.settings.Set(ctx, "home_info", string(value)); err != nil {
		return models.HomeInfo{}, err
	}
	return home, nil
}
