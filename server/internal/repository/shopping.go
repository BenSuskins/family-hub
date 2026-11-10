package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type ShoppingRepository interface {
	FindAll(context.Context) ([]models.ShoppingItem, error)
	Create(context.Context, string, *string) (models.ShoppingItem, error)
	Update(context.Context, string, models.ShoppingChange) (models.ShoppingItem, error)
	Delete(context.Context, string) error
	ClearPurchased(context.Context) error
	Restock(context.Context, string, models.ShoppingRestock) (models.ShoppingItem, error)
}
type SQLiteShoppingRepository struct{ database *sql.DB }

func NewShoppingRepository(database *sql.DB) *SQLiteShoppingRepository {
	return &SQLiteShoppingRepository{database: database}
}

const shoppingColumns = "id,name,inventory_item_id,purchased,purchase_id,restocked,version"

func scanShopping(row interface{ Scan(...any) error }) (models.ShoppingItem, error) {
	var item models.ShoppingItem
	err := row.Scan(&item.ID, &item.Name, &item.InventoryItemID, &item.Purchased, &item.PurchaseID, &item.Restocked, &item.Version)
	return item, err
}
func (repository *SQLiteShoppingRepository) FindAll(ctx context.Context) ([]models.ShoppingItem, error) {
	rows, err := repository.database.QueryContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items ORDER BY purchased,created_at,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.ShoppingItem{}
	for rows.Next() {
		item, err := scanShopping(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (repository *SQLiteShoppingRepository) Create(ctx context.Context, name string, inventoryID *string) (models.ShoppingItem, error) {

	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return models.ShoppingItem{}, err
	}
	defer transaction.Rollback()
	if inventoryID != nil {
		var inventoryName string
		if err := transaction.QueryRowContext(ctx, "SELECT name FROM inventory_items WHERE id=?", *inventoryID).Scan(&inventoryName); err != nil {
			return models.ShoppingItem{}, shoppingError(err)
		}
		existing, err := scanShopping(transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE inventory_item_id=? AND purchased=0", *inventoryID))
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return models.ShoppingItem{}, err
		}
	}
	identifier := uuid.NewString()
	_, err = transaction.ExecContext(ctx, "INSERT INTO shopping_items(id,name,inventory_item_id) VALUES(?,?,?)", identifier, name, inventoryID)
	if err != nil {
		return models.ShoppingItem{}, shoppingError(err)
	}
	item, err := scanShopping(transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE id=?", identifier))
	if err != nil {
		return item, err
	}
	return item, transaction.Commit()
}
func (repository *SQLiteShoppingRepository) Update(ctx context.Context, id string, change models.ShoppingChange) (models.ShoppingItem, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return models.ShoppingItem{}, err
	}
	defer transaction.Rollback()
	item, err := scanShopping(transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE id=?", id))
	if err != nil {
		return item, shoppingError(err)
	}
	var createPurchase bool
	item, createPurchase, err = shoppingdomain.ApplyChange(item, change, uuid.NewString())
	if err != nil {
		return item, err
	}
	if createPurchase {
		if _, err := transaction.ExecContext(ctx, "INSERT INTO shopping_purchases(id,shopping_item_id) VALUES(?,?)", *item.PurchaseID, id); err != nil {
			return item, err
		}
	}
	result, err := transaction.ExecContext(ctx, "UPDATE shopping_items SET name=?,purchased=?,purchase_id=?,restocked=?,version=version+1 WHERE id=? AND version=?", item.Name, item.Purchased, item.PurchaseID, item.Restocked, id, item.Version)
	if err != nil {
		return item, shoppingError(err)
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return item, shoppingdomain.ErrConflict
	}
	item.Version++
	return item, transaction.Commit()
}
func (repository *SQLiteShoppingRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM shopping_items WHERE id=?", id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return shoppingdomain.ErrNotFound
	}
	return nil
}
func (repository *SQLiteShoppingRepository) ClearPurchased(ctx context.Context) error {
	_, err := repository.database.ExecContext(ctx, "DELETE FROM shopping_items WHERE purchased=1")
	return err
}
func (repository *SQLiteShoppingRepository) Restock(ctx context.Context, id string, request models.ShoppingRestock) (models.ShoppingItem, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return models.ShoppingItem{}, err
	}
	defer transaction.Rollback()
	item, err := scanShopping(transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE id=?", id))
	if err != nil {
		return item, shoppingError(err)
	}
	alreadyRestocked, err := shoppingdomain.CheckRestock(item, request)
	if err != nil {
		return item, err
	}
	if alreadyRestocked {
		return item, nil
	}
	var mode string
	if err := transaction.QueryRowContext(ctx, "SELECT tracking_mode FROM inventory_items WHERE id=?", *item.InventoryItemID).Scan(&mode); err != nil {
		return item, shoppingError(err)
	}
	if err := shoppingdomain.ValidateRestock(mode, request); err != nil {
		return item, err
	}
	if mode == models.TrackingModeCount {
		_, err = transaction.ExecContext(ctx, "UPDATE inventory_items SET quantity=quantity+?,updated_at=? WHERE id=?", *request.Quantity, time.Now(), *item.InventoryItemID)
	} else {
		_, err = transaction.ExecContext(ctx, "UPDATE inventory_items SET level=?,updated_at=? WHERE id=?", *request.Level, time.Now(), *item.InventoryItemID)
	}
	if err != nil {
		return item, err
	}
	result, err := transaction.ExecContext(ctx, "UPDATE shopping_purchases SET restocked_at=? WHERE id=? AND restocked_at IS NULL", time.Now(), request.PurchaseID)
	if err != nil {
		return item, err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return item, shoppingdomain.ErrConflict
	}
	if _, err := transaction.ExecContext(ctx, "UPDATE shopping_items SET restocked=1,version=version+1 WHERE id=?", id); err != nil {
		return item, err
	}
	item.Restocked = true
	item.Version++
	return item, transaction.Commit()
}
func shoppingError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return shoppingdomain.ErrNotFound
	}
	if err != nil && (strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "locked")) {
		return shoppingdomain.ErrConflict
	}
	return err
}
