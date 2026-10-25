package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

const binReminderSlot = 2

type BinReminderService struct {
	settings repository.SettingsRepository
	devices  repository.NotificationRepository
	sender   MessageSender
}

func NewBinReminderService(settings repository.SettingsRepository, devices repository.NotificationRepository, sender MessageSender) *BinReminderService {
	return &BinReminderService{settings: settings, devices: devices, sender: sender}
}

func (service *BinReminderService) Check(ctx context.Context, now time.Time) error {
	if service.sender == nil {
		return nil
	}
	schedule, err := LoadBinSchedule(ctx, service.settings)
	if err != nil || !schedule.Enabled || len(schedule.Collections) == 0 {
		return err
	}
	location, err := HouseholdLocation(ctx, service.settings)
	if err != nil {
		return err
	}
	local := now.In(location)
	if err := service.devices.Prune(ctx, local.AddDate(0, 0, -7).Format("2006-01-02")); err != nil {
		return err
	}
	scheduled := time.Date(local.Year(), local.Month(), local.Day(), 19, 0, 0, 0, location)
	if scheduled.Format("15:04") != "19:00" || now.Before(scheduled) || !now.Before(scheduled.Add(5*time.Minute)) {
		return nil
	}
	tomorrow := local.AddDate(0, 0, 1).Format("2006-01-02")
	due := make([]models.BinCollection, 0)
	for _, collection := range schedule.Collections {
		if collection.Date == tomorrow {
			due = append(due, collection)
		}
	}
	if len(due) == 0 {
		return nil
	}
	devices, err := service.devices.Devices(ctx)
	if err != nil {
		return err
	}
	date, err := time.ParseInLocation("2006-01-02", tomorrow, location)
	if err != nil {
		return err
	}
	message := BinReminderMessage(due, date)
	var failures []error
	for _, device := range devices {
		identifier, err := service.devices.Claim(ctx, device.ID, local.Format("2006-01-02"), binReminderSlot, now)
		if err != nil {
			return err
		}
		if identifier == "" {
			continue
		}
		retry := false
		err = service.sender.SendMessage(ctx, device, message, identifier, scheduled.Add(5*time.Minute))
		if err != nil {
			var pushError *PushError
			retry = true
			if errors.As(err, &pushError) {
				retry = pushError.Retry
				if pushError.InvalidDevice {
					if removeError := service.devices.Invalidate(ctx, device); removeError != nil {
						failures = append(failures, removeError)
					}
				}
			}
			failures = append(failures, fmt.Errorf("bin reminder delivery %s: %w", identifier, err))
		}
		if err := service.devices.Finish(ctx, identifier, retry); err != nil {
			return err
		}
	}
	return errors.Join(failures...)
}

func BinReminderMessage(collections []models.BinCollection, date time.Time) PushMessage {
	labels := make([]string, 0, len(collections))
	seen := map[string]bool{}
	for _, collection := range collections {
		if !seen[collection.Name] {
			seen[collection.Name] = true
			labels = append(labels, collection.Name)
		}
	}
	body := "Bins are due tomorrow, " + date.Format("Mon 2 Jan") + "."
	verb := "are"
	if len(labels) == 1 {
		verb = "is"
	}
	if len(labels) > 0 {
		body = joinBinNames(labels) + " " + verb + " due tomorrow, " + date.Format("Mon 2 Jan") + "."
	}
	return PushMessage{Title: "Bins out tonight", Body: body, URL: "familyhub://today", ThreadID: "bin-reminders"}
}

func joinBinNames(names []string) string {
	switch len(names) {
	case 0:
		return "Bins"
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + lowerInitial(names[1])
	default:
		lowered := append([]string{names[0]}, names[1:]...)
		for index := 1; index < len(lowered); index++ {
			lowered[index] = lowerInitial(lowered[index])
		}
		return strings.Join(lowered[:len(lowered)-1], ", ") + " and " + lowered[len(lowered)-1]
	}
}

func lowerInitial(value string) string {
	if value == "" {
		return value
	}
	characters := []rune(value)
	return strings.ToLower(string(characters[0])) + string(characters[1:])
}

func HouseholdLocation(ctx context.Context, settings repository.SettingsRepository) (*time.Location, error) {
	name := "Europe/London"
	value, err := settings.Get(ctx, "home_info")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		var home models.HomeInfo
		if err := json.Unmarshal([]byte(value), &home); err != nil {
			return nil, err
		}
		if home.Timezone != "" {
			name = home.Timezone
		}
	}
	return time.LoadLocation(name)
}
