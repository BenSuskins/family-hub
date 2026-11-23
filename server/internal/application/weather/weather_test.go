package weather_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/weather"
)

type providerFake struct {
	calls int
	now   *time.Time
	rain  float64
}

func (provider *providerFake) Forecast(context.Context, weather.Location) (weather.Forecast, error) {
	provider.calls++
	return weather.Forecast{FetchedAt: *provider.now, Advice: []string{"rain"}, Hours: []weather.Hour{{Code: 1}}, Days: []weather.Day{{RainProbability: &provider.rain}}}, nil
}
func TestForecastCachesAndCopiesResults(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	latitude, longitude := 1.0, 2.0
	provider := &providerFake{now: &now, rain: 42}
	service := weather.New(provider, func() time.Time { return now })
	home := weather.Location{Latitude: &latitude, Longitude: &longitude, Timezone: "UTC"}
	first, err := service.Forecast(context.Background(), home)
	if err != nil {
		t.Fatal(err)
	}
	first.Advice[0] = "changed"
	first.Hours[0].Code = 99
	*first.Days[0].RainProbability = 99
	provider.rain = 88
	second, err := service.Forecast(context.Background(), home)
	if err != nil || second.Advice[0] != "rain" || second.Hours[0].Code != 1 || *second.Days[0].RainProbability != 42 || provider.calls != 1 {
		t.Fatalf("cache=%+v calls=%d err=%v", second, provider.calls, err)
	}
	now = now.Add(15 * time.Minute)
	_, err = service.Forecast(context.Background(), home)
	if err != nil || provider.calls != 2 {
		t.Fatalf("expiry calls=%d err=%v", provider.calls, err)
	}
}
