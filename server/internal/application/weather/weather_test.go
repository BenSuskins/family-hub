package weather_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/weather"
	"github.com/bensuskins/family-hub/internal/models"
)

type providerFake struct {
	calls int
	now   *time.Time
}

func (provider *providerFake) Forecast(context.Context, models.HomeInfo) (models.Weather, error) {
	provider.calls++
	return models.Weather{FetchedAt: *provider.now, Advice: []string{"rain"}}, nil
}
func TestForecastCachesAndCopiesResults(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	latitude, longitude := 1.0, 2.0
	provider := &providerFake{now: &now}
	service := weather.New(provider, func() time.Time { return now })
	home := models.HomeInfo{Latitude: &latitude, Longitude: &longitude, Timezone: "UTC"}
	first, err := service.Forecast(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	first.Advice[0] = "changed"
	second, err := service.Forecast(context.Background(), home)
	if err != nil || second.Advice[0] != "rain" || provider.calls != 1 {
		t.Fatalf("cache=%+v calls=%d err=%v", second, provider.calls, err)
	}
	now = now.Add(15 * time.Minute)
	_, err = service.Forecast(context.Background(), home)
	if err != nil || provider.calls != 2 {
		t.Fatalf("expiry calls=%d err=%v", provider.calls, err)
	}
}
