package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	shoppingapp "github.com/bensuskins/family-hub/internal/application/shopping"
	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
)

type SQLiteShoppingRepository struct {
	database  *sql.DB
	inventory func(*sql.Tx) shoppingapp.Inventory
}

func NewShoppingRepository(database *sql.DB, inventory func(*sql.Tx) shoppingapp.Inventory) *SQLiteShoppingRepository {
	return &SQLiteShoppingRepository{database: database, inventory: inventory}
}

const shoppingColumns = "id,name,inventory_item_id,purchased,purchase_id,restocked,version"

func scanShopping(row interface{ Scan(...any) error }) (shoppingdomain.Item, error) {
	var item shoppingdomain.Item
	err := row.Scan(&item.ID, &item.Name, &item.InventoryItemID, &item.Purchased, &item.PurchaseID, &item.Restocked, &item.Version)
	return item, err
}
func (repository *SQLiteShoppingRepository) FindAll(ctx context.Context) ([]shoppingdomain.Item, error) {
	rows, err := repository.database.QueryContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items ORDER BY purchased,created_at,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []shoppingdomain.Item{}
	for rows.Next() {
		item, err := scanShopping(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
func (repository *SQLiteShoppingRepository) Transact(ctx context.Context, operation func(shoppingapp.Transaction) (bool, error)) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	commit, err := operation(&shoppingTransaction{transaction: transaction, inventory: repository.inventory(transaction)})
	if err != nil {
		return err
	}
	if !commit {
		return nil
	}
	return transaction.Commit()
}

type shoppingTransaction struct {
	transaction *sql.Tx
	inventory   shoppingapp.Inventory
}

func (transaction *shoppingTransaction) Inventory() shoppingapp.Inventory {
	return transaction.inventory
}
func (transaction *shoppingTransaction) Find(ctx context.Context, id string) (shoppingdomain.Aggregate, error) {
	item, err := scanShopping(transaction.transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE id=?", id))
	if err != nil {
		return shoppingdomain.RestoreLegacy(item), shoppingError(err)
	}
	aggregate, restoreErr := shoppingdomain.Restore(item)
	if restoreErr != nil {
		aggregate = shoppingdomain.RestoreLegacy(item)
	}
	return aggregate, nil
}

func (transaction *shoppingTransaction) FindActive(ctx context.Context, id string) (shoppingdomain.Item, bool, error) {
	item, err := scanShopping(transaction.transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE inventory_item_id=? AND purchased=0", id))
	if errors.Is(err, sql.ErrNoRows) {
		return shoppingdomain.Item{}, false, nil
	}
	return item, err == nil, err
}
func (transaction *shoppingTransaction) Insert(ctx context.Context, item shoppingdomain.Item) (shoppingdomain.Item, error) {
	_, err := transaction.transaction.ExecContext(ctx, "INSERT INTO shopping_items(id,name,inventory_item_id) VALUES(?,?,?)", item.ID, item.Name, item.InventoryItemID)
	if err != nil {
		return shoppingdomain.Item{}, shoppingError(err)
	}
	return scanShopping(transaction.transaction.QueryRowContext(ctx, "SELECT "+shoppingColumns+" FROM shopping_items WHERE id=?", item.ID))
}
func (transaction *shoppingTransaction) InsertPurchase(ctx context.Context, purchaseID, itemID string) error {
	_, err := transaction.transaction.ExecContext(ctx, "INSERT INTO shopping_purchases(id,shopping_item_id) VALUES(?,?)", purchaseID, itemID)
	return err
}
func (transaction *shoppingTransaction) Save(ctx context.Context, item shoppingdomain.Item) error {
	result, err := transaction.transaction.ExecContext(ctx, "UPDATE shopping_items SET name=?,purchased=?,purchase_id=?,restocked=?,version=version+1 WHERE id=? AND version=?", item.Name, item.Purchased, item.PurchaseID, item.Restocked, item.ID, item.Version)
	if err != nil {
		return shoppingError(err)
	}
	changed, _ := result.RowsAffected()
	if changed != 1 {
		return shoppingdomain.ErrConflict
	}
	return nil
}
func (transaction *shoppingTransaction) Claim(ctx context.Context, purchaseID string, at time.Time) (bool, error) {
	result, err := transaction.transaction.ExecContext(ctx, "UPDATE shopping_purchases SET restocked_at=? WHERE id=? AND restocked_at IS NULL", at, purchaseID)
	if err != nil {
		return false, err
	}
	count, _ := result.RowsAffected()
	return count == 1, nil
}
func (transaction *shoppingTransaction) MarkRestocked(ctx context.Context, id string) error {
	_, err := transaction.transaction.ExecContext(ctx, "UPDATE shopping_items SET restocked=1,version=version+1 WHERE id=?", id)
	return err
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
