package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/bensuskins/family-hub/internal/application/shopping"
	inventorydomain "github.com/bensuskins/family-hub/internal/domain/inventory"
)

type shoppingInventory struct{ transaction *sql.Tx }

func NewShoppingInventory(transaction *sql.Tx) shopping.Inventory {
	return &shoppingInventory{transaction: transaction}
}

func (inventory *shoppingInventory) TrackingMode(ctx context.Context, id string) (string, error) {
	var mode string
	if err := inventory.transaction.QueryRowContext(ctx, "SELECT tracking_mode FROM inventory_items WHERE id=?", id).Scan(&mode); err != nil {
		return "", shoppingError(err)
	}
	return mode, nil
}

func (inventory *shoppingInventory) ApplyRestock(ctx context.Context, id, mode string, quantity, level *int, at time.Time) error {
	if mode == inventorydomain.TrackingModeCount {
		_, err := inventory.transaction.ExecContext(ctx, "UPDATE inventory_items SET quantity=quantity+?,updated_at=? WHERE id=?", *quantity, at, id)
		return err
	}
	_, err := inventory.transaction.ExecContext(ctx, "UPDATE inventory_items SET level=?,updated_at=? WHERE id=?", *level, at, id)
	return err
}
