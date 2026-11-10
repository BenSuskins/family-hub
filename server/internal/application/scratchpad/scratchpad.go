package scratchpad

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
	"github.com/bensuskins/family-hub/internal/models"
)

type Store interface {
	Get(context.Context) (models.Scratchpad, error)
	Save(context.Context, []models.ScratchpadStroke, string) (models.Scratchpad, error)
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) Get(ctx context.Context) (models.Scratchpad, error) {
	result, err := service.store.Get(ctx)
	result.Strokes = copyStrokes(result.Strokes)
	return result, err
}
func (service *Service) Save(ctx context.Context, strokes []models.ScratchpadStroke, userID string) (models.Scratchpad, error) {
	if err := domain.Validate(strokes); err != nil {
		return models.Scratchpad{}, err
	}
	result, err := service.store.Save(ctx, copyStrokes(strokes), userID)
	result.Strokes = copyStrokes(result.Strokes)
	return result, err
}

func copyStrokes(strokes []models.ScratchpadStroke) []models.ScratchpadStroke {
	if strokes == nil {
		return nil
	}
	result := make([]models.ScratchpadStroke, len(strokes))
	for index, stroke := range strokes {
		result[index] = stroke
		if stroke.Points != nil {
			result[index].Points = append([]models.ScratchpadPoint{}, stroke.Points...)
		}
	}
	return result
}
