package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

func (service *ChoreService) Forecast(ctx context.Context, start, end time.Time) ([]models.Chore, error) {
	result := []models.Chore{}
	now, err := service.householdNow(ctx)
	if err != nil {
		return nil, err
	}
	tasks, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
	if err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if task.RecurOnComplete || task.RecurrenceType == models.RecurrenceNone || task.DueDate == nil {
			continue
		}
		candidates, err := service.findCandidates(ctx, task.ID, service.loadSeries(ctx, task.SeriesID))
		if err != nil {
			return nil, err
		}
		remaining := maxExpansionIterations
		if task.RecurrenceCount != nil && task.SeriesID != nil {
			count, err := service.choreRepo.CountBySeries(ctx, *task.SeriesID)
			if err != nil {
				return nil, err
			}
			remaining = *task.RecurrenceCount - count
		}
		date, err := nextTaskDate(task, now)
		if err != nil {
			return nil, err
		}
		config, err := parseConfig(task.RecurrenceValue)
		if err != nil {
			return nil, err
		}
		owner := task.AssignedToUserID
		index := task.LastAssignedIndex
		for iteration := 0; iteration < remaining && iteration < maxExpansionIterations && date != nil; iteration++ {
			if date.Format("2006-01-02") > end.Format("2006-01-02") {
				break
			}
			if task.RecurrenceUntil != nil && date.Format("2006-01-02") > task.RecurrenceUntil.Format("2006-01-02") {
				break
			}
			if len(candidates) > 0 {
				index = rotationStart(candidates, owner, index)
				selected := candidates[index].ID
				owner = &selected
			}
			if date.Format("2006-01-02") >= start.Format("2006-01-02") {
				forecast := task
				forecast.ID = fmt.Sprintf("forecast:%s:%s", task.ID, date.Format("2006-01-02"))
				forecast.Name = task.Name + " (provisional)"
				forecast.DueDate = date
				forecast.AssignedToUserID = owner
				forecast.IsForecast = true
				forecast.Status = models.ChoreStatusPending
				result = append(result, forecast)
			}
			next := advanceToNextOccurrence(*date, task.RecurrenceType, config)
			if !next.After(*date) {
				break
			}
			date = &next
		}
	}
	return result, nil
}

func (service *ChoreService) DueTasks(ctx context.Context) (today, overdue []models.Chore, err error) {
	today = []models.Chore{}
	overdue = []models.Chore{}
	now, err := service.householdNow(ctx)
	if err != nil {
		return nil, nil, err
	}
	tasks, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
	if err != nil {
		return nil, nil, err
	}
	for _, task := range tasks {
		if taskIsOverdue(task, now) {
			task.Status = models.ChoreStatusOverdue
			overdue = append(overdue, task)
		} else if task.DueDate == nil || task.DueDate.Format("2006-01-02") == now.Format("2006-01-02") {
			task.Status = models.ChoreStatusPending
			today = append(today, task)
		}
	}
	return today, overdue, nil
}

func taskIsOverdue(task models.Chore, now time.Time) bool {
	return repository.IsOverdue(task, now) || task.DueDate == nil && task.Status == models.ChoreStatusOverdue
}
