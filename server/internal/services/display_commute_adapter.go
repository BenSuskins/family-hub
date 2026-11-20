package services

import (
	"context"
	"encoding/json"

	commuteapp "github.com/bensuskins/family-hub/internal/application/commute"
	displayapp "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type DisplayCommutePreferences struct{ settings *displayapp.Settings }

func NewDisplayCommutePreferences(settings *displayapp.Settings) *DisplayCommutePreferences {
	return &DisplayCommutePreferences{settings}
}
func (adapter *DisplayCommutePreferences) Preferences(ctx context.Context, token string) (commuteapp.DisplayPreferences, error) {
	display, err := adapter.settings.FindByTokenID(ctx, token)
	if err != nil {
		return commuteapp.DisplayPreferences{}, err
	}
	p := display.CommutePreferences
	convert := func(s models.DisplayCommuteSchedule) commuteapp.DisplaySchedule {
		return commuteapp.DisplaySchedule{Enabled: s.Enabled, Weekdays: append([]int(nil), s.Weekdays...), Start: s.Start, End: s.End}
	}
	return commuteapp.DisplayPreferences{Home: p.HomeStationCode, Work: p.WorkStationCode, Count: p.TrainCount, ToWork: convert(p.ToWork), ToHome: convert(p.ToHome)}, nil
}

type DisplayCommuteTimezone struct{ settings repository.SettingsRepository }

func NewDisplayCommuteTimezone(settings repository.SettingsRepository) *DisplayCommuteTimezone {
	return &DisplayCommuteTimezone{settings}
}
func (adapter *DisplayCommuteTimezone) Timezone(ctx context.Context) commuteapp.DisplayTimezone {
	value, err := adapter.settings.Get(ctx, "home_info")
	if err != nil {
		return commuteapp.DisplayTimezone{ReadError: err}
	}
	var home models.HomeInfo
	err = json.Unmarshal([]byte(value), &home)
	return commuteapp.DisplayTimezone{Value: home.Timezone, DecodeError: err}
}

type DisplayCommuteRail struct{ service *commuteapp.CommuteService }

func NewDisplayCommuteRail(service *commuteapp.CommuteService) *DisplayCommuteRail {
	return &DisplayCommuteRail{service}
}
func (adapter *DisplayCommuteRail) Fetch(ctx context.Context, origin, destination string, count int) (commuteapp.DisplayBoard, error) {
	board, err := adapter.service.FetchTrains(ctx, origin, destination, count)
	if err != nil {
		return commuteapp.DisplayBoard{}, err
	}
	result := commuteapp.DisplayBoard{Origin: board.Origin.Code, Destination: board.Destination.Code, GeneratedAt: board.GeneratedAt, FetchedAt: board.FetchedAt, Timezone: board.Timezone}
	if board.Trains != nil {
		result.Trains = make([]commuteapp.DisplayTrain, len(board.Trains))
		for i, t := range board.Trains {
			result.Trains[i] = commuteapp.DisplayTrain{ServiceID: t.ServiceID, Operator: t.Operator, Platform: t.Platform, Status: t.Status, ScheduledDeparture: t.ScheduledDeparture, ExpectedDeparture: t.ExpectedDeparture, ScheduledArrival: t.ScheduledArrival, ExpectedArrival: t.ExpectedArrival, CancellationReason: t.CancellationReason, DelayReason: t.DelayReason}
		}
	}
	return result, nil
}
