package inventory

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

type Store interface {
	FindAllAreas(context.Context) ([]domain.Area, error)
	FindAreaByID(context.Context, string) (domain.Area, error)
	CreateArea(context.Context, domain.Area) (domain.Area, error)
	UpdateArea(context.Context, domain.Area) error
	DeleteArea(context.Context, string) error
	FindItemByID(context.Context, string) (domain.Item, error)
	CreateItem(context.Context, domain.Item) (domain.Item, error)
	UpdateItem(context.Context, domain.Item) error
	DeleteItem(context.Context, string) error
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func copyArea(area domain.Area) domain.Area {
	if area.Items != nil {
		area.Items = append([]domain.Item{}, area.Items...)
	}
	return area
}
func (service *Service) FindAllAreas(ctx context.Context) ([]domain.Area, error) {
	areas, err := service.store.FindAllAreas(ctx)
	if areas != nil {
		copies := make([]domain.Area, len(areas))
		for index, area := range areas {
			copies[index] = copyArea(area)
		}
		areas = copies
	}
	return areas, err
}
func (service *Service) FindAreaByID(ctx context.Context, id string) (domain.Area, error) {
	area, err := service.store.FindAreaByID(ctx, id)
	return copyArea(area), err
}
func (service *Service) CreateArea(ctx context.Context, area domain.Area) (domain.Area, error) {
	result, err := service.store.CreateArea(ctx, copyArea(area))
	return copyArea(result), err
}
func (service *Service) SaveArea(ctx context.Context, area domain.Area) (domain.Area, error) {
	if err := service.store.UpdateArea(ctx, copyArea(area)); err != nil {
		return domain.Area{}, err
	}
	updated, err := service.store.FindAreaByID(ctx, area.ID)
	if err != nil {
		return copyArea(area), nil
	}
	return copyArea(updated), nil
}
func (service *Service) DeleteArea(ctx context.Context, id string) error {
	return service.store.DeleteArea(ctx, id)
}
func (service *Service) FindItemByID(ctx context.Context, id string) (domain.Item, error) {
	return service.store.FindItemByID(ctx, id)
}
func (service *Service) CreateItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	return service.store.CreateItem(ctx, domain.NormalizeItem(item))
}
func (service *Service) SaveItem(ctx context.Context, item domain.Item) (domain.Item, error) {
	if err := service.store.UpdateItem(ctx, domain.NormalizeItem(item)); err != nil {
		return domain.Item{}, err
	}
	updated, err := service.store.FindItemByID(ctx, item.ID)
	if err != nil {
		return item, nil
	}
	return updated, nil
}
func (service *Service) DeleteItem(ctx context.Context, id string) error {
	return service.store.DeleteItem(ctx, id)
}
