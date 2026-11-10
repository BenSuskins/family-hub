package weather

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type Provider interface {
	Forecast(context.Context, models.HomeInfo) (models.Weather, error)
}
type Service struct {
	provider Provider
	now      func() time.Time
	mutex    sync.Mutex
	cached   models.Weather
	cacheKey string
}

func New(provider Provider, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{provider: provider, now: now}
}
func (service *Service) Forecast(ctx context.Context, home models.HomeInfo) (models.Weather, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if home.Latitude == nil || home.Longitude == nil {
		return models.Weather{}, errors.New("home location missing")
	}
	key := fmt.Sprintf("%f:%f:%s", *home.Latitude, *home.Longitude, home.Timezone)
	if key == service.cacheKey && service.now().Sub(service.cached.FetchedAt) < 15*time.Minute {
		return copyWeather(service.cached), nil
	}
	result, err := service.provider.Forecast(ctx, home)
	if err != nil {
		return models.Weather{}, err
	}
	service.cached, service.cacheKey = copyWeather(result), key
	return copyWeather(result), nil
}
func copyWeather(value models.Weather) models.Weather {
	if value.Hours != nil {
		value.Hours = append([]models.WeatherHour{}, value.Hours...)
	}
	if value.Days != nil {
		value.Days = append([]models.WeatherDay{}, value.Days...)
		for index := range value.Days {
			if value.Days[index].RainProbability != nil {
				v := *value.Days[index].RainProbability
				value.Days[index].RainProbability = &v
			}
		}
	}
	if value.Advice != nil {
		value.Advice = append([]string{}, value.Advice...)
	}
	return value
}
