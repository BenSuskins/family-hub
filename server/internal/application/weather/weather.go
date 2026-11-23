package weather

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Provider interface {
	Forecast(context.Context, Location) (Forecast, error)
}
type Service struct {
	provider Provider
	now      func() time.Time
	mutex    sync.Mutex
	cached   Forecast
	cacheKey string
}

func New(provider Provider, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{provider: provider, now: now}
}
func (service *Service) Forecast(ctx context.Context, home Location) (Forecast, error) {
	service.mutex.Lock()
	defer service.mutex.Unlock()
	if home.Latitude == nil || home.Longitude == nil {
		return Forecast{}, errors.New("home location missing")
	}
	key := fmt.Sprintf("%f:%f:%s", *home.Latitude, *home.Longitude, home.Timezone)
	if key == service.cacheKey && service.now().Sub(service.cached.FetchedAt) < 15*time.Minute {
		return copyWeather(service.cached), nil
	}
	result, err := service.provider.Forecast(ctx, home)
	if err != nil {
		return Forecast{}, err
	}
	service.cached, service.cacheKey = copyWeather(result), key
	return copyWeather(result), nil
}
func copyWeather(value Forecast) Forecast {
	if value.Hours != nil {
		value.Hours = append([]Hour{}, value.Hours...)
	}
	if value.Days != nil {
		value.Days = append([]Day{}, value.Days...)
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
