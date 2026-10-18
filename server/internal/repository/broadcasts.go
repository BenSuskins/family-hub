package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type BroadcastDelivery struct {
	ID     string
	Device models.NotificationDevice
	State  string
}

type BroadcastRepository interface {
	Prepare(context.Context, string, string, string, time.Time) (time.Time, error)
	Deliveries(context.Context, string) ([]BroadcastDelivery, error)
	Claim(context.Context, string, time.Time) (bool, error)
	Finish(context.Context, string, string) error
}

type SQLiteBroadcastRepository struct{ database *sql.DB }

func NewBroadcastRepository(database *sql.DB) *SQLiteBroadcastRepository {
	return &SQLiteBroadcastRepository{database: database}
}

func (repository *SQLiteBroadcastRepository) Prepare(ctx context.Context, identifier, userID, message string, now time.Time) (time.Time, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return time.Time{}, err
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(ctx, `DELETE FROM notification_broadcasts WHERE created_at < ?`, now.AddDate(0, 0, -30).Unix()); err != nil {
		return time.Time{}, err
	}
	result, err := transaction.ExecContext(ctx, `INSERT INTO notification_broadcasts (id,user_id,message,created_at) VALUES (?,?,?,?) ON CONFLICT(id) DO NOTHING`, identifier, userID, message, now.Unix())
	if err != nil {
		return time.Time{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, err
	}
	var owner, storedMessage string
	var createdAt int64
	if err := transaction.QueryRowContext(ctx, `SELECT user_id,message,created_at FROM notification_broadcasts WHERE id=?`, identifier).Scan(&owner, &storedMessage, &createdAt); err != nil {
		return time.Time{}, err
	}
	if owner != userID || storedMessage != message {
		return time.Time{}, models.ErrBroadcastConflict
	}
	if changed > 0 {
		rows, err := transaction.QueryContext(ctx, `SELECT d.id FROM notification_devices d LEFT JOIN api_tokens t ON t.token_hash=d.api_token_hash WHERE d.api_token_hash IS NULL OR (t.token_hash IS NOT NULL AND (t.expires_at IS NULL OR t.expires_at > CURRENT_TIMESTAMP))`)
		if err != nil {
			return time.Time{}, err
		}
		var identifiers []string
		for rows.Next() {
			var deviceID string
			if err := rows.Scan(&deviceID); err != nil {
				rows.Close()
				return time.Time{}, err
			}
			identifiers = append(identifiers, deviceID)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return time.Time{}, err
		}
		for _, deviceID := range identifiers {
			if _, err := transaction.ExecContext(ctx, `INSERT INTO broadcast_deliveries (id,broadcast_id,device_id) VALUES (?,?,?)`, uuid.NewString(), identifier, deviceID); err != nil {
				return time.Time{}, err
			}
		}
	}
	if err := transaction.Commit(); err != nil {
		return time.Time{}, err
	}
	return time.Unix(createdAt, 0).Add(5 * time.Minute), nil
}

func (repository *SQLiteBroadcastRepository) Deliveries(ctx context.Context, identifier string) ([]BroadcastDelivery, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT b.id,b.state,COALESCE(d.id,''),COALESCE(d.user_id,''),COALESCE(d.token,''),COALESCE(d.environment,'') FROM broadcast_deliveries b LEFT JOIN notification_devices d ON d.id=b.device_id AND (d.api_token_hash IS NULL OR EXISTS(SELECT 1 FROM api_tokens t WHERE t.token_hash=d.api_token_hash AND (t.expires_at IS NULL OR t.expires_at > CURRENT_TIMESTAMP))) WHERE b.broadcast_id=? ORDER BY b.id`, identifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []BroadcastDelivery{}
	for rows.Next() {
		var delivery BroadcastDelivery
		if err := rows.Scan(&delivery.ID, &delivery.State, &delivery.Device.ID, &delivery.Device.UserID, &delivery.Device.Token, &delivery.Device.Environment); err != nil {
			return nil, err
		}
		result = append(result, delivery)
	}
	return result, rows.Err()
}

func (repository *SQLiteBroadcastRepository) Claim(ctx context.Context, identifier string, now time.Time) (bool, error) {
	result, err := repository.database.ExecContext(ctx, `UPDATE broadcast_deliveries SET state='sending',lease_until=? WHERE id=? AND (state IN ('pending','retry') OR (state='sending' AND lease_until<=?))`, now.Add(time.Minute).Unix(), identifier, now.Unix())
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	return changed > 0, err
}

func (repository *SQLiteBroadcastRepository) Finish(ctx context.Context, identifier, state string) error {
	_, err := repository.database.ExecContext(ctx, `UPDATE broadcast_deliveries SET state=? WHERE id=?`, state, identifier)
	return err
}
