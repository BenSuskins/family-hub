package services

import (
	"context"
	"database/sql"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

type ChoreDefinition struct {
	ID          string
	Chore       models.Chore
	CurrentTask *models.Chore
	Active      bool
	DeletedAt   *time.Time
}

func (service *ChoreService) Definitions(ctx context.Context, includeInactive bool) ([]ChoreDefinition, error) {
	definitions, err := service.seriesRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	result := []ChoreDefinition{}
	for _, definition := range definitions {
		detail, err := service.Definition(ctx, definition.ID)
		if err != nil {
			return nil, err
		}
		if includeInactive || detail.Active {
			result = append(result, detail)
		}
	}
	return result, nil
}
func (service *ChoreService) Definition(ctx context.Context, identifier string) (ChoreDefinition, error) {
	definition, err := service.seriesRepo.FindByID(ctx, identifier)
	if err != nil {
		return ChoreDefinition{}, err
	}
	if definition == nil {
		return ChoreDefinition{}, sql.ErrNoRows
	}
	task := models.Chore{ID: identifier, Name: definition.Name, Description: definition.Description, CreatedByUserID: definition.CreatedByUserID, SeriesID: &identifier, EligibleAssignees: definition.EligibleAssignees, Status: models.ChoreStatusPending}
	task = applySeriesRule(task, definition)
	result := ChoreDefinition{ID: identifier, Chore: task, DeletedAt: definition.DeletedAt}
	if definition.DeletedAt != nil {
		return result, nil
	}
	tasks, err := service.choreRepo.FindAll(ctx, repository.ChoreFilter{SeriesID: &identifier, Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
	if err != nil {
		return result, err
	}
	if len(tasks) > 0 {
		result.CurrentTask = &tasks[0]
		result.Active = true
		result.Chore.DueDate = tasks[0].DueDate
		result.Chore.AssignedToUserID = tasks[0].AssignedToUserID
	}
	return result, nil
}
