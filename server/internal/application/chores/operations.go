package chores

import (
	"context"
	"errors"

	"github.com/bensuskins/family-hub/internal/models"
)

type ChoreHistory struct {
	Items      []models.Chore
	NextCursor string
}

func (service *ChoreService) History(ctx context.Context, limit int, cursor, definitionID string) (ChoreHistory, error) {
	if limit < 1 || limit > 100 {
		return ChoreHistory{}, errors.New("limit must be between 1 and 100")
	}
	filter := ChoreFilter{IncludeHistory: true, HistoryOnly: true, OrderBy: OrderHistory, Limit: limit + 1, AfterID: cursor}
	if definitionID != "" {
		filter.SeriesID = &definitionID
	}
	items, err := service.choreRepo.FindAll(ctx, filter)
	if err != nil {
		return ChoreHistory{}, err
	}
	if items == nil {
		items = []models.Chore{}
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = items[len(items)-1].ID
	}
	return ChoreHistory{Items: items, NextCursor: next}, nil
}

func (service *ChoreService) UpdateExistingChore(ctx context.Context, task models.Chore, eligible []string) (models.Chore, error) {
	if task.Status == models.ChoreStatusSkipped || task.Status == models.ChoreStatusSuperseded {
		return models.Chore{}, ErrChoreAlreadyComplete
	}
	if task.SeriesID == nil {
		task.SeriesID = &task.ID
	}
	if eligible == nil {
		definition, err := service.Definition(ctx, *task.SeriesID)
		if err != nil {
			return models.Chore{}, err
		}
		eligible = definition.Chore.EligibleAssignees
	}
	if err := service.UpdateDefinition(ctx, task, eligible); err != nil {
		return models.Chore{}, err
	}
	return service.choreRepo.FindByID(ctx, task.ID)
}

func (service *ChoreService) DeleteExistingChore(ctx context.Context, task models.Chore) error {
	if task.SeriesID != nil {
		return service.DeleteSeriesDefinition(ctx, *task.SeriesID)
	}
	return nil
}

func (service *ChoreService) FindChore(ctx context.Context, id string) (models.Chore, error) {
	return service.choreRepo.FindByID(ctx, id)
}

func (service *ChoreService) FindChores(ctx context.Context, filter ChoreFilter) ([]models.Chore, error) {
	return service.choreRepo.FindAll(ctx, filter)
}
