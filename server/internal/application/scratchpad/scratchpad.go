package scratchpad

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
)

type Store interface {
	Get(context.Context) (domain.Snapshot, error)
	Save(context.Context, []domain.Stroke, string) (domain.Snapshot, error)
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func (service *Service) Get(ctx context.Context) (domain.Snapshot, error) {
	result, err := service.store.Get(ctx)
	result.Strokes = copyStrokes(result.Strokes)
	return result, err
}
func (service *Service) Save(ctx context.Context, strokes []domain.Stroke, userID string) (domain.Snapshot, error) {
	drawing, err := domain.NewDrawing(strokes)
	if err != nil {
		return domain.Snapshot{}, err
	}
	result, err := service.store.Save(ctx, drawing.Strokes(), userID)
	result.Strokes = copyStrokes(result.Strokes)
	return result, err
}

func copyStrokes(strokes []domain.Stroke) []domain.Stroke {
	if strokes == nil {
		return nil
	}
	result := make([]domain.Stroke, len(strokes))
	for index, stroke := range strokes {
		result[index] = stroke
		if stroke.Points != nil {
			result[index].Points = append([]domain.Point{}, stroke.Points...)
		}
	}
	return result
}
