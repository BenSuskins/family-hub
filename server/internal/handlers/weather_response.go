package handlers

import (
	"time"

	weatherapp "github.com/bensuskins/family-hub/internal/application/weather"
)

type weatherHourResponse struct {
	Time            time.Time `json:"time"`
	Temperature     float64   `json:"temperature"`
	Code            int       `json:"code"`
	RainProbability float64   `json:"rainProbability"`
	WindSpeed       float64   `json:"windSpeed"`
}
type weatherDayResponse struct {
	Date            time.Time `json:"date"`
	High            float64   `json:"high"`
	Low             float64   `json:"low"`
	Code            int       `json:"code"`
	RainProbability *float64  `json:"rainProbability,omitempty"`
}
type weatherResponse struct {
	Status    string                `json:"status"`
	FetchedAt time.Time             `json:"fetchedAt"`
	Current   weatherHourResponse   `json:"current"`
	High      float64               `json:"high"`
	Low       float64               `json:"low"`
	Hours     []weatherHourResponse `json:"hours"`
	Days      []weatherDayResponse  `json:"days"`
	Advice    []string              `json:"advice"`
}

func weatherHourResponseFor(hour weatherapp.Hour) weatherHourResponse {
	return weatherHourResponse{hour.Time, hour.Temperature, hour.Code, hour.RainProbability, hour.WindSpeed}
}
func weatherResponseFor(forecast weatherapp.Forecast) weatherResponse {
	response := weatherResponse{Status: forecast.Status, FetchedAt: forecast.FetchedAt, Current: weatherHourResponseFor(forecast.Current), High: forecast.High, Low: forecast.Low, Advice: forecast.Advice}
	if forecast.Hours != nil {
		response.Hours = make([]weatherHourResponse, len(forecast.Hours))
		for index, hour := range forecast.Hours {
			response.Hours[index] = weatherHourResponseFor(hour)
		}
	}
	if forecast.Days != nil {
		response.Days = make([]weatherDayResponse, len(forecast.Days))
		for index, day := range forecast.Days {
			response.Days[index] = weatherDayResponse{day.Date, day.High, day.Low, day.Code, day.RainProbability}
		}
	}
	return response
}
