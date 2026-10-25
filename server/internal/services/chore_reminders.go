package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type ReminderSender interface {
	Send(context.Context, models.NotificationDevice, ChoreReminderSummary, string, time.Time) error
}

type PushError struct {
	Reason        string
	InvalidDevice bool
	Retry         bool
}

func (failure *PushError) Error() string { return "push rejected: " + failure.Reason }

type ChoreReminderService struct {
	settings repository.SettingsRepository
	chores   repository.ChoreRepository
	devices  repository.NotificationRepository
	sender   ReminderSender
	users    repository.UserRepository
}

func NewChoreReminderService(settings repository.SettingsRepository, chores repository.ChoreRepository, devices repository.NotificationRepository, sender ReminderSender, users ...repository.UserRepository) *ChoreReminderService {
	service := &ChoreReminderService{settings: settings, chores: chores, devices: devices, sender: sender}
	if len(users) > 0 {
		service.users = users[0]
	}
	return service
}

func LoadChoreReminders(ctx context.Context, settings repository.SettingsRepository) (models.ChoreReminders, error) {
	result := models.ChoreReminders{Times: [2]string{"10:00", "16:30"}, Timezone: "Europe/London"}
	value, err := settings.Get(ctx, "chore_reminders")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil {
		if err := json.Unmarshal([]byte(value), &result); err != nil {
			return result, err
		}
	}
	home, err := settings.Get(ctx, "home_info")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	if err == nil {
		var info models.HomeInfo
		if err := json.Unmarshal([]byte(home), &info); err != nil {
			return result, err
		}
		if info.Timezone != "" {
			result.Timezone = info.Timezone
		}
	}
	if _, err := time.LoadLocation(result.Timezone); err != nil {
		return result, err
	}
	return result, nil
}

func (service *ChoreReminderService) Check(ctx context.Context, now time.Time) error {
	settings, err := LoadChoreReminders(ctx, service.settings)
	if err != nil {
		return err
	}
	if !settings.Enabled || service.sender == nil {
		return nil
	}
	location, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		return err
	}
	local := now.In(location)
	if err := service.devices.Prune(ctx, local.AddDate(0, 0, -7).Format("2006-01-02")); err != nil {
		return err
	}
	var failures []error
	for slot, clock := range settings.Times {
		parsed, err := time.Parse("15:04", clock)
		if err != nil {
			return err
		}
		scheduled := time.Date(local.Year(), local.Month(), local.Day(), parsed.Hour(), parsed.Minute(), 0, 0, location)
		// Skip wall-clock times that do not exist on the daylight saving change date.
		if scheduled.Format("15:04") != clock {
			continue
		}
		expires := scheduled.Add(5 * time.Minute)
		if now.Before(scheduled) || !now.Before(expires) {
			continue
		}
		devices, err := service.devices.Devices(ctx)
		if err != nil {
			return err
		}
		for _, device := range devices {
			// Read again for each delivery so a retry sees completions from other devices.
			chores, err := service.chores.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
			if err != nil {
				return err
			}
			names := map[string]string{}
			if service.users != nil {
				people, err := service.users.FindAll(ctx)
				if err != nil {
					return err
				}
				for _, person := range people {
					names[person.ID] = person.Name
				}
			}
			sort.SliceStable(chores, func(first, second int) bool {
				a, b := chores[first], chores[second]
				overdue := func(task models.Chore) bool {
					return task.Status == models.ChoreStatusOverdue || task.DueDate != nil && task.DueDate.Format("2006-01-02") < local.Format("2006-01-02")
				}
				if overdue(a) != overdue(b) {
					return overdue(a)
				}
				if a.DueDate != nil && b.DueDate != nil && !a.DueDate.Equal(*b.DueDate) {
					return a.DueDate.Before(*b.DueDate)
				}
				if a.Name != b.Name {
					return a.Name < b.Name
				}
				return a.ID < b.ID
			})
			summary := ChoreReminderSummary{}
			for _, chore := range chores {
				if chore.Status == models.ChoreStatusCompleted {
					continue
				}
				// Chore due dates store calendar dates, rather than instants in the household timezone.
				if chore.Status == models.ChoreStatusOverdue || chore.DueDate == nil || chore.DueDate.Format("2006-01-02") <= local.Format("2006-01-02") {
					summary.Count++
					name := "Unassigned"
					if chore.AssignedToUserID != nil && names[*chore.AssignedToUserID] != "" {
						name = names[*chore.AssignedToUserID]
					}
					summary.Tasks = append(summary.Tasks, ChoreReminderTask{Assignee: name, Name: chore.Name})
				}
			}
			identifier, err := service.devices.Claim(ctx, device.ID, local.Format("2006-01-02"), slot, now)
			if err != nil {
				return err
			}
			if identifier == "" {
				continue
			}
			retry := false
			if summary.Count > 0 {
				err = service.sender.Send(ctx, device, summary, identifier, expires)
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
					failures = append(failures, fmt.Errorf("reminder delivery %s: %w", identifier, err))
				}
			}
			if err := service.devices.Finish(ctx, identifier, retry); err != nil {
				return err
			}
		}
	}
	return errors.Join(failures...)
}

type ChoreReminderTask struct {
	Assignee string
	Name     string
}
type ChoreReminderSummary struct {
	Count int
	Tasks []ChoreReminderTask
}

func (summary ChoreReminderSummary) Message() PushMessage {
	assignees := []string{}
	tasksByAssignee := map[string][]string{}
	for _, task := range summary.Tasks {
		assignee := strings.Join(strings.Fields(task.Assignee), " ")
		if _, exists := tasksByAssignee[assignee]; !exists {
			assignees = append(assignees, assignee)
		}
		tasksByAssignee[assignee] = append(tasksByAssignee[assignee], strings.Join(strings.Fields(task.Name), " "))
	}
	parts := make([]string, 0, len(assignees))
	for _, assignee := range assignees {
		parts = append(parts, assignee+": "+strings.Join(tasksByAssignee[assignee], ", "))
	}
	body := strings.Join(parts, "\n")
	return PushMessage{Title: "Chore reminder", Body: body, URL: "familyhub://today", ThreadID: "chore-reminders"}
}
