package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type InventoryRepository interface {
	// Areas
	FindAllAreas(ctx context.Context) ([]models.InventoryArea, error)
	FindAreaByID(ctx context.Context, id string) (models.InventoryArea, error)
	CreateArea(ctx context.Context, area models.InventoryArea) (models.InventoryArea, error)
	UpdateArea(ctx context.Context, area models.InventoryArea) error
	DeleteArea(ctx context.Context, id string) error

	// Items
	FindItemByID(ctx context.Context, id string) (models.InventoryItem, error)
	CreateItem(ctx context.Context, item models.InventoryItem) (models.InventoryItem, error)
	UpdateItem(ctx context.Context, item models.InventoryItem) error
	DeleteItem(ctx context.Context, id string) error
}

type SQLiteInventoryRepository struct {
	database *sql.DB
}

func NewInventoryRepository(database *sql.DB) *SQLiteInventoryRepository {
	return &SQLiteInventoryRepository{database: database}
}

// FindAllAreas returns every area with its items nested. Items are loaded in a
// single query and attached to their area to avoid an N+1.
func (repository *SQLiteInventoryRepository) FindAllAreas(ctx context.Context) ([]models.InventoryArea, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, icon, tint, created_by_user_id, created_at, updated_at
		FROM inventory_areas ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding inventory areas: %w", err)
	}
	defer rows.Close()

	var areas []models.InventoryArea
	index := map[string]int{}
	for rows.Next() {
		var area models.InventoryArea
		if err := rows.Scan(
			&area.ID, &area.Name, &area.Icon, &area.Tint,
			&area.CreatedByUserID, &area.CreatedAt, &area.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning inventory area: %w", err)
		}
		area.Items = []models.InventoryItem{}
		index[area.ID] = len(areas)
		areas = append(areas, area)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(areas) == 0 {
		return areas, nil
	}

	itemRows, err := repository.database.QueryContext(ctx,
		`SELECT id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at
		FROM inventory_items ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding inventory items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item models.InventoryItem
		if err := scanItem(itemRows, &item); err != nil {
			return nil, err
		}
		if i, ok := index[item.AreaID]; ok {
			areas[i].Items = append(areas[i].Items, item)
		}
	}
	return areas, itemRows.Err()
}

// FindAreaByID returns a single area with its items nested.
func (repository *SQLiteInventoryRepository) FindAreaByID(ctx context.Context, id string) (models.InventoryArea, error) {
	var area models.InventoryArea
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, name, icon, tint, created_by_user_id, created_at, updated_at
		FROM inventory_areas WHERE id = ?`, id,
	).Scan(
		&area.ID, &area.Name, &area.Icon, &area.Tint,
		&area.CreatedByUserID, &area.CreatedAt, &area.UpdatedAt,
	)
	if err != nil {
		return models.InventoryArea{}, fmt.Errorf("finding inventory area by id: %w", err)
	}

	items, err := repository.findItemsByArea(ctx, id)
	if err != nil {
		return models.InventoryArea{}, err
	}
	area.Items = items
	return area, nil
}

func (repository *SQLiteInventoryRepository) CreateArea(ctx context.Context, area models.InventoryArea) (models.InventoryArea, error) {
	if area.ID == "" {
		area.ID = uuid.New().String()
	}
	if area.Icon == "" {
		area.Icon = "box"
	}
	if area.Tint == "" {
		area.Tint = "blue"
	}
	now := time.Now()
	area.CreatedAt = now
	area.UpdatedAt = now

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO inventory_areas (id, name, icon, tint, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		area.ID, area.Name, area.Icon, area.Tint, area.CreatedByUserID, area.CreatedAt, area.UpdatedAt,
	)
	if err != nil {
		return models.InventoryArea{}, fmt.Errorf("creating inventory area: %w", err)
	}
	if area.Items == nil {
		area.Items = []models.InventoryItem{}
	}
	return area, nil
}

func (repository *SQLiteInventoryRepository) UpdateArea(ctx context.Context, area models.InventoryArea) error {
	area.UpdatedAt = time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE inventory_areas SET name = ?, icon = ?, tint = ?, updated_at = ? WHERE id = ?`,
		area.Name, area.Icon, area.Tint, area.UpdatedAt, area.ID,
	)
	if err != nil {
		return fmt.Errorf("updating inventory area: %w", err)
	}
	return nil
}

func (repository *SQLiteInventoryRepository) DeleteArea(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM inventory_areas WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting inventory area: %w", err)
	}
	return nil
}

func (repository *SQLiteInventoryRepository) FindItemByID(ctx context.Context, id string) (models.InventoryItem, error) {
	var item models.InventoryItem
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at
		FROM inventory_items WHERE id = ?`, id,
	).Scan(
		&item.ID, &item.AreaID, &item.Name, &item.TrackingMode, &item.Quantity, &item.Level, &item.Unit, &item.LowAt,
		&item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return models.InventoryItem{}, fmt.Errorf("finding inventory item by id: %w", err)
	}
	return item, nil
}

func (repository *SQLiteInventoryRepository) CreateItem(ctx context.Context, item models.InventoryItem) (models.InventoryItem, error) {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item = normalizeItem(item)
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO inventory_items (id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.AreaID, item.Name, item.TrackingMode, item.Quantity, item.Level, item.Unit, item.LowAt,
		item.CreatedByUserID, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return models.InventoryItem{}, fmt.Errorf("creating inventory item: %w", err)
	}
	return item, nil
}

func (repository *SQLiteInventoryRepository) UpdateItem(ctx context.Context, item models.InventoryItem) error {
	item = normalizeItem(item)
	item.UpdatedAt = time.Now()
	_, err := repository.database.ExecContext(ctx,
		`UPDATE inventory_items SET name = ?, tracking_mode = ?, quantity = ?, level = ?, unit = ?, low_at = ?, updated_at = ? WHERE id = ?`,
		item.Name, item.TrackingMode, item.Quantity, item.Level, item.Unit, item.LowAt, item.UpdatedAt, item.ID,
	)
	if err != nil {
		return fmt.Errorf("updating inventory item: %w", err)
	}
	return nil
}

func (repository *SQLiteInventoryRepository) DeleteItem(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM inventory_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting inventory item: %w", err)
	}
	return nil
}

func (repository *SQLiteInventoryRepository) findItemsByArea(ctx context.Context, areaID string) ([]models.InventoryItem, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at
		FROM inventory_items WHERE area_id = ? ORDER BY name ASC`, areaID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding inventory items: %w", err)
	}
	defer rows.Close()

	items := []models.InventoryItem{}
	for rows.Next() {
		var item models.InventoryItem
		if err := scanItem(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanItem(rows *sql.Rows, item *models.InventoryItem) error {
	if err := rows.Scan(
		&item.ID, &item.AreaID, &item.Name, &item.TrackingMode, &item.Quantity, &item.Level, &item.Unit, &item.LowAt,
		&item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return fmt.Errorf("scanning inventory item: %w", err)
	}
	return nil
}

// normalizeItem clamps an item's fields to valid ranges before persisting:
// quantity and low_at are never negative, level is bounded to 0–100, and in
// level mode low_at is read as a percentage so it is bounded to 0–100 too. An
// unrecognized tracking mode falls back to count.
func normalizeItem(item models.InventoryItem) models.InventoryItem {
	if item.TrackingMode != models.TrackingModeLevel {
		item.TrackingMode = models.TrackingModeCount
	}
	item.Quantity = atLeast(item.Quantity, 0)
	item.Level = clamp(item.Level, 0, 100)
	if item.TrackingMode == models.TrackingModeLevel {
		item.LowAt = clamp(item.LowAt, 0, 100)
	} else {
		item.LowAt = atLeast(item.LowAt, 0)
	}
	return item
}

func atLeast(value, low int) int {
	if value < low {
		return low
	}
	return value
}

func clamp(value, low, high int) int {
	return atLeast(min(value, high), low)
}
