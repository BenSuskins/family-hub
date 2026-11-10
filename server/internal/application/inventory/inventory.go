package inventory

import (
	"context"

	domain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/bensuskins/family-hub/internal/models"
)

type Store interface {
	FindAllAreas(context.Context) ([]models.InventoryArea, error)
	FindAreaByID(context.Context, string) (models.InventoryArea, error)
	CreateArea(context.Context, models.InventoryArea) (models.InventoryArea, error)
	UpdateArea(context.Context, models.InventoryArea) error
	DeleteArea(context.Context, string) error
	FindItemByID(context.Context, string) (models.InventoryItem, error)
	CreateItem(context.Context, models.InventoryItem) (models.InventoryItem, error)
	UpdateItem(context.Context, models.InventoryItem) error
	DeleteItem(context.Context, string) error
}
type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }
func copyArea(area models.InventoryArea) models.InventoryArea {
	if area.Items != nil {
		area.Items = append([]models.InventoryItem{}, area.Items...)
	}
	return area
}
func (service *Service) FindAllAreas(ctx context.Context) ([]models.InventoryArea, error) {
	areas, err := service.store.FindAllAreas(ctx)
	if areas != nil {
		copies := make([]models.InventoryArea, len(areas))
		for index, area := range areas {
			copies[index] = copyArea(area)
		}
		areas = copies
	}
	return areas, err
}
func (service *Service) FindAreaByID(ctx context.Context, id string) (models.InventoryArea, error) {
	area, err := service.store.FindAreaByID(ctx, id)
	return copyArea(area), err
}
func (service *Service) CreateArea(ctx context.Context, area models.InventoryArea) (models.InventoryArea, error) {
	result, err := service.store.CreateArea(ctx, copyArea(area))
	return copyArea(result), err
}
func (service *Service) SaveArea(ctx context.Context, area models.InventoryArea) (models.InventoryArea, error) {
	if err := service.store.UpdateArea(ctx, copyArea(area)); err != nil {
		return models.InventoryArea{}, err
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
func (service *Service) FindItemByID(ctx context.Context, id string) (models.InventoryItem, error) {
	return service.store.FindItemByID(ctx, id)
}
func (service *Service) CreateItem(ctx context.Context, item models.InventoryItem) (models.InventoryItem, error) {
	return service.store.CreateItem(ctx, domain.NormalizeItem(item))
}
func (service *Service) SaveItem(ctx context.Context, item models.InventoryItem) (models.InventoryItem, error) {
	if err := service.store.UpdateItem(ctx, domain.NormalizeItem(item)); err != nil {
		return models.InventoryItem{}, err
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
