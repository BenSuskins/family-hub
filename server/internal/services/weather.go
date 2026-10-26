package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type OpenMeteo struct {
	client   *http.Client
	endpoint string
	now      func() time.Time
	mutex    sync.Mutex
	cached   models.Weather
	cacheKey string
}

func NewOpenMeteo(client *http.Client, endpoint string, now func() time.Time) *OpenMeteo {
	return &OpenMeteo{client: client, endpoint: endpoint, now: now}
}

func (provider *OpenMeteo) Forecast(ctx context.Context, home models.HomeInfo) (models.Weather, error) {
	provider.mutex.Lock()
	defer provider.mutex.Unlock()
	if home.Latitude == nil || home.Longitude == nil {
		return models.Weather{}, errors.New("home location missing")
	}
	key := fmt.Sprintf("%f:%f:%s", *home.Latitude, *home.Longitude, home.Timezone)
	now := provider.now()
	if key == provider.cacheKey && now.Sub(provider.cached.FetchedAt) < 15*time.Minute {
		return provider.cached, nil
	}
	parameters := url.Values{
		"latitude": {fmt.Sprint(*home.Latitude)}, "longitude": {fmt.Sprint(*home.Longitude)},
		"timezone": {home.Timezone}, "timeformat": {"unixtime"}, "forecast_days": {"7"},
		"current": {"temperature_2m,weather_code,wind_speed_10m"},
		"hourly":  {"temperature_2m,weather_code,precipitation_probability,wind_speed_10m"},
		"daily":   {"temperature_2m_max,temperature_2m_min,weather_code,precipitation_probability_max"},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.endpoint+"?"+parameters.Encode(), nil)
	if err != nil {
		return models.Weather{}, errors.New("invalid weather endpoint")
	}
	var data struct {
		Current struct {
			Time        int64    `json:"time"`
			Temperature *float64 `json:"temperature_2m"`
			Code        *int     `json:"weather_code"`
			Wind        *float64 `json:"wind_speed_10m"`
		} `json:"current"`
		Hourly struct {
			Time        []int64    `json:"time"`
			Temperature []*float64 `json:"temperature_2m"`
			Code        []*int     `json:"weather_code"`
			Rain        []*float64 `json:"precipitation_probability"`
			Wind        []*float64 `json:"wind_speed_10m"`
		} `json:"hourly"`
		Daily struct {
			Time []int64    `json:"time"`
			High []*float64 `json:"temperature_2m_max"`
			Low  []*float64 `json:"temperature_2m_min"`
			Code []*int     `json:"weather_code"`
			Rain []*float64 `json:"precipitation_probability_max"`
		} `json:"daily"`
	}
	if err := readProviderJSON(provider.client, request, &data); err != nil {
		return models.Weather{}, err
	}
	if data.Current.Temperature == nil || data.Current.Code == nil || data.Current.Wind == nil || data.Current.Time == 0 || len(data.Daily.Time) != 7 || len(data.Daily.High) != 7 || len(data.Daily.Low) != 7 || len(data.Daily.Code) != 7 {
		return models.Weather{}, errors.New("incomplete forecast")
	}
	forecastDays := make([]models.WeatherDay, 0, 7)
	for index, timestamp := range data.Daily.Time {
		if data.Daily.High[index] == nil || data.Daily.Low[index] == nil || data.Daily.Code[index] == nil {
			return models.Weather{}, errors.New("incomplete daily forecast")
		}
		var rainProbability *float64
		if index < len(data.Daily.Rain) {
			rainProbability = data.Daily.Rain[index]
		}
		forecastDays = append(forecastDays, models.WeatherDay{Date: time.Unix(timestamp, 0), High: *data.Daily.High[index], Low: *data.Daily.Low[index], Code: *data.Daily.Code[index], RainProbability: rainProbability})
	}
	count := len(data.Hourly.Time)
	if count == 0 || len(data.Hourly.Temperature) != count || len(data.Hourly.Code) != count || len(data.Hourly.Rain) != count || len(data.Hourly.Wind) != count {
		return models.Weather{}, errors.New("incomplete hourly forecast")
	}
	forecast := models.Weather{Status: "ok", FetchedAt: now, Current: models.WeatherHour{Time: time.Unix(data.Current.Time, 0), Temperature: *data.Current.Temperature, Code: *data.Current.Code, WindSpeed: *data.Current.Wind}, High: *data.Daily.High[0], Low: *data.Daily.Low[0], Hours: []models.WeatherHour{}, Days: forecastDays, Advice: []string{}}
	rain, cold, heat, wind := false, false, false, false
	location, err := time.LoadLocation(home.Timezone)
	if err != nil {
		return models.Weather{}, errors.New("invalid household timezone")
	}
	local := now.In(location)
	hourStart := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), 0, 0, 0, location)
	for index, timestamp := range data.Hourly.Time {
		instant := time.Unix(timestamp, 0)
		if instant.Before(hourStart) || !instant.Before(hourStart.Add(24*time.Hour)) {
			continue
		}
		if data.Hourly.Temperature[index] == nil || data.Hourly.Code[index] == nil || data.Hourly.Rain[index] == nil || data.Hourly.Wind[index] == nil {
			return models.Weather{}, errors.New("missing hourly value")
		}
		hour := models.WeatherHour{Time: instant, Temperature: *data.Hourly.Temperature[index], Code: *data.Hourly.Code[index], RainProbability: *data.Hourly.Rain[index], WindSpeed: *data.Hourly.Wind[index]}
		if math.IsNaN(hour.Temperature) {
			return models.Weather{}, errors.New("invalid temperature")
		}
		forecast.Hours = append(forecast.Hours, hour)
		if instant.Before(hourStart.Add(6 * time.Hour)) {
			rain = rain || hour.RainProbability >= 50
			cold = cold || hour.Temperature < 10
			heat = heat || hour.Temperature >= 28
			wind = wind || hour.WindSpeed >= 40
		}
	}
	if len(forecast.Hours) != 24 || now.Sub(forecast.Current.Time) > 90*time.Minute || forecast.Current.Time.After(now.Add(15*time.Minute)) {
		return models.Weather{}, errors.New("forecast is out of date")
	}
	if rain {
		forecast.Advice = append(forecast.Advice, "Take a raincoat.")
	}
	if cold {
		forecast.Advice = append(forecast.Advice, "Take a warm layer.")
	}
	if heat {
		forecast.Advice = append(forecast.Advice, "Take water and sun protection.")
	}
	if wind {
		forecast.Advice = append(forecast.Advice, "Expect strong winds.")
	}
	if len(forecast.Advice) == 0 {
		forecast.Advice = append(forecast.Advice, "No weather precautions for the next six hours.")
	}
	provider.cached, provider.cacheKey = forecast, key
	return forecast, nil
}

func readProviderJSON(client *http.Client, request *http.Request, target any) error {
	response, err := client.Do(request)
	if err != nil {
		return errors.New("provider request failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.New("provider unavailable")
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2*1024*1024)).Decode(target); err != nil {
		return errors.New("invalid provider response")
	}
	return nil
}
