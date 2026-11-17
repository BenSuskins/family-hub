package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
	"github.com/google/uuid"
)

type inventoryItemRecord struct {
	ID, AreaID, Name, TrackingMode string
	Quantity, Level                int
	Unit                           string
	LowAt                          int
	CreatedByUserID                string
	CreatedAt, UpdatedAt           time.Time
}

func inventoryItemFromRecord(record inventoryItemRecord) inventorydomain.Item {
	return inventorydomain.Item{ID: record.ID, AreaID: record.AreaID, Name: record.Name, TrackingMode: record.TrackingMode, Quantity: record.Quantity, Level: record.Level, Unit: record.Unit, LowAt: record.LowAt, CreatedByUserID: record.CreatedByUserID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func inventoryItemToRecord(item inventorydomain.Item) inventoryItemRecord {
	return inventoryItemRecord{ID: item.ID, AreaID: item.AreaID, Name: item.Name, TrackingMode: item.TrackingMode, Quantity: item.Quantity, Level: item.Level, Unit: item.Unit, LowAt: item.LowAt, CreatedByUserID: item.CreatedByUserID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

type inventoryAreaRecord struct {
	ID, Name, Icon, Tint, CreatedByUserID string
	CreatedAt, UpdatedAt                  time.Time
}

func inventoryAreaFromRecord(record inventoryAreaRecord) inventorydomain.Area {
	return inventorydomain.Area{ID: record.ID, Name: record.Name, Icon: record.Icon, Tint: record.Tint, CreatedByUserID: record.CreatedByUserID, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt}
}
func inventoryAreaToRecord(area inventorydomain.Area) inventoryAreaRecord {
	return inventoryAreaRecord{ID: area.ID, Name: area.Name, Icon: area.Icon, Tint: area.Tint, CreatedByUserID: area.CreatedByUserID, CreatedAt: area.CreatedAt, UpdatedAt: area.UpdatedAt}
}
func scanInventoryArea(row interface{ Scan(...any) error }) (inventorydomain.Area, error) {
	var record inventoryAreaRecord
	err := row.Scan(&record.ID, &record.Name, &record.Icon, &record.Tint, &record.CreatedByUserID, &record.CreatedAt, &record.UpdatedAt)
	return inventoryAreaFromRecord(record), err
}

type SQLiteInventoryRepository struct {
	database *sql.DB
}

func NewInventoryRepository(database *sql.DB) *SQLiteInventoryRepository {
	return &SQLiteInventoryRepository{database: database}
}

// FindAllAreas returns every area with its items nested. Items are loaded in a
// single query and attached to their area to avoid an N+1.
func (repository *SQLiteInventoryRepository) FindAllAreas(ctx context.Context) ([]inventorydomain.Area, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, name, icon, tint, created_by_user_id, created_at, updated_at
		FROM inventory_areas ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding inventory areas: %w", err)
	}
	defer rows.Close()

	var areas []inventorydomain.Area
	index := map[string]int{}
	for rows.Next() {
		area, err := scanInventoryArea(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning inventory area: %w", err)
		}
		area.Items = []inventorydomain.Item{}
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
		var item inventorydomain.Item
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
func (repository *SQLiteInventoryRepository) FindAreaByID(ctx context.Context, id string) (inventorydomain.Area, error) {
	area, err := scanInventoryArea(repository.database.QueryRowContext(ctx,
		`SELECT id, name, icon, tint, created_by_user_id, created_at, updated_at
        FROM inventory_areas WHERE id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return inventorydomain.Area{}, inventorydomain.ErrAreaNotFound
	}
	if err != nil {
		return inventorydomain.Area{}, fmt.Errorf("finding inventory area by id: %w", err)
	}

	items, err := repository.findItemsByArea(ctx, id)
	if err != nil {
		return inventorydomain.Area{}, err
	}
	area.Items = items
	return area, nil
}

func (repository *SQLiteInventoryRepository) CreateArea(ctx context.Context, area inventorydomain.Area) (inventorydomain.Area, error) {
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

	record := inventoryAreaToRecord(area)
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO inventory_areas (id, name, icon, tint, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.Name, record.Icon, record.Tint, record.CreatedByUserID, record.CreatedAt, record.UpdatedAt,
	)
	if err != nil {
		return inventorydomain.Area{}, fmt.Errorf("creating inventory area: %w", err)
	}
	if area.Items == nil {
		area.Items = []inventorydomain.Item{}
	}
	return area, nil
}

func (repository *SQLiteInventoryRepository) UpdateArea(ctx context.Context, area inventorydomain.Area) error {
	area.UpdatedAt = time.Now()
	record := inventoryAreaToRecord(area)
	_, err := repository.database.ExecContext(ctx,
		`UPDATE inventory_areas SET name = ?, icon = ?, tint = ?, updated_at = ? WHERE id = ?`,
		record.Name, record.Icon, record.Tint, record.UpdatedAt, record.ID,
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

func (repository *SQLiteInventoryRepository) FindItemByID(ctx context.Context, id string) (inventorydomain.Item, error) {
	item, err := readInventoryItem(repository.database.QueryRowContext(ctx,
		`SELECT id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at
        FROM inventory_items WHERE id = ?`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return inventorydomain.Item{}, inventorydomain.ErrItemNotFound
	}
	if err != nil {
		return inventorydomain.Item{}, fmt.Errorf("finding inventory item by id: %w", err)
	}
	return item, nil
}

func (repository *SQLiteInventoryRepository) CreateItem(ctx context.Context, item inventorydomain.Item) (inventorydomain.Item, error) {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item = inventorydomain.NormalizeItem(item)
	now := time.Now()
	item.CreatedAt = now
	item.UpdatedAt = now

	record := inventoryItemToRecord(item)
	_, err := repository.database.ExecContext(ctx,
		`INSERT INTO inventory_items (id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.ID, record.AreaID, record.Name, record.TrackingMode, record.Quantity, record.Level, record.Unit, record.LowAt,
		record.CreatedByUserID, record.CreatedAt, record.UpdatedAt,
	)
	if err != nil {
		return inventorydomain.Item{}, fmt.Errorf("creating inventory item: %w", err)
	}
	return item, nil
}

func (repository *SQLiteInventoryRepository) UpdateItem(ctx context.Context, item inventorydomain.Item) error {
	item = inventorydomain.NormalizeItem(item)
	item.UpdatedAt = time.Now()
	record := inventoryItemToRecord(item)
	_, err := repository.database.ExecContext(ctx,
		`UPDATE inventory_items SET name = ?, tracking_mode = ?, quantity = ?, level = ?, unit = ?, low_at = ?, updated_at = ? WHERE id = ?`,
		record.Name, record.TrackingMode, record.Quantity, record.Level, record.Unit, record.LowAt, record.UpdatedAt, record.ID,
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

func (repository *SQLiteInventoryRepository) findItemsByArea(ctx context.Context, areaID string) ([]inventorydomain.Item, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at
		FROM inventory_items WHERE area_id = ? ORDER BY name ASC`, areaID,
	)
	if err != nil {
		return nil, fmt.Errorf("finding inventory items: %w", err)
	}
	defer rows.Close()

	items := []inventorydomain.Item{}
	for rows.Next() {
		var item inventorydomain.Item
		if err := scanItem(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func readInventoryItem(row interface{ Scan(...any) error }) (inventorydomain.Item, error) {
	var record inventoryItemRecord
	err := row.Scan(&record.ID, &record.AreaID, &record.Name, &record.TrackingMode, &record.Quantity, &record.Level, &record.Unit, &record.LowAt, &record.CreatedByUserID, &record.CreatedAt, &record.UpdatedAt)
	return inventoryItemFromRecord(record), err
}
func scanItem(rows interface{ Scan(...any) error }, item *inventorydomain.Item) error {
	record, err := readInventoryItem(rows)
	if err != nil {
		return fmt.Errorf("scanning inventory item: %w", err)
	}
	*item = record
	return nil
}

func (repository *SQLiteInventoryRepository) ModifyStock(ctx context.Context, id string, change func(inventorydomain.Stock) (inventorydomain.Stock, error)) (inventorydomain.Item, error) {
	for {
		if err := ctx.Err(); err != nil {
			return inventorydomain.Item{}, err
		}
		current, err := repository.FindItemByID(ctx, id)
		if err != nil {
			return inventorydomain.Item{}, err
		}
		stock, err := inventorydomain.RestoreStock(current)
		if err != nil {
			stock = inventorydomain.RestoreLegacyStock(current)
		}
		updated, err := change(stock)
		if err != nil {
			return inventorydomain.Item{}, err
		}
		next := inventoryItemToRecord(updated.Snapshot())
		row := repository.database.QueryRowContext(ctx,
			`UPDATE inventory_items SET quantity = ?, level = ?, updated_at = ?
    WHERE id = ? AND tracking_mode = ? AND quantity = ? AND level = ?
    RETURNING id, area_id, name, tracking_mode, quantity, level, unit, low_at, created_by_user_id, created_at, updated_at`,
			next.Quantity, next.Level, time.Now(), id, current.TrackingMode, current.Quantity, current.Level)
		var saved inventorydomain.Item
		if err := scanItem(row, &saved); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return inventorydomain.Item{}, err
		}
		return saved, nil
	}
}
