package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	Register(context.Context, models.NotificationDevice) error
	Remove(context.Context, string, string) error
	Devices(context.Context) ([]models.NotificationDevice, error)
	Claim(context.Context, string, string, int, time.Time) (string, error)
	Finish(context.Context, string, bool) error
	Invalidate(context.Context, models.NotificationDevice) error
	Prune(context.Context, string) error
}

type SQLiteNotificationRepository struct{ database *sql.DB }

func NewNotificationRepository(database *sql.DB) *SQLiteNotificationRepository {
	return &SQLiteNotificationRepository{database: database}
}

func (repository *SQLiteNotificationRepository) Register(ctx context.Context, device models.NotificationDevice) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	var conflictingOwners int
	err = transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_devices WHERE (id = ? OR (token = ? AND environment = ?)) AND user_id != ?`, device.ID, device.Token, device.Environment, device.UserID).Scan(&conflictingOwners)
	if err != nil {
		return err
	}
	if conflictingOwners != 0 {
		return models.ErrDeviceOwnership
	}
	// A token can change while the installation identifier remains stable.
	if _, err := transaction.ExecContext(ctx, `DELETE FROM notification_devices WHERE token = ? AND environment = ? AND id != ? AND user_id = ?`, device.Token, device.Environment, device.ID, device.UserID); err != nil {
		return err
	}
	if _, err := transaction.ExecContext(ctx, `INSERT INTO notification_devices (id,user_id,token,environment,api_token_hash) VALUES (?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET token=excluded.token, environment=excluded.environment, api_token_hash=excluded.api_token_hash WHERE user_id=excluded.user_id`, device.ID, device.UserID, device.Token, device.Environment, device.APITokenHash); err != nil {
		return err
	}
	return transaction.Commit()
}

func (repository *SQLiteNotificationRepository) Remove(ctx context.Context, identifier, userID string) error {
	_, err := repository.database.ExecContext(ctx, `DELETE FROM notification_devices WHERE id = ? AND user_id = ?`, identifier, userID)
	return err
}

func (repository *SQLiteNotificationRepository) Devices(ctx context.Context) ([]models.NotificationDevice, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT d.id,d.user_id,d.token,d.environment FROM notification_devices d LEFT JOIN api_tokens t ON t.token_hash=d.api_token_hash WHERE d.api_token_hash IS NULL OR (t.token_hash IS NOT NULL AND (t.expires_at IS NULL OR t.expires_at > CURRENT_TIMESTAMP)) ORDER BY d.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	devices := []models.NotificationDevice{}
	for rows.Next() {
		var device models.NotificationDevice
		if err := rows.Scan(&device.ID, &device.UserID, &device.Token, &device.Environment); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

func (repository *SQLiteNotificationRepository) Claim(ctx context.Context, deviceID, date string, slot int, now time.Time) (string, error) {
	identifier := uuid.NewString()
	result, err := repository.database.ExecContext(ctx, `INSERT INTO reminder_deliveries (id,device_id,local_date,slot,state,lease_until) VALUES (?,?,?,?,'sending',?) ON CONFLICT(device_id,local_date,slot) DO UPDATE SET state='sending', lease_until=excluded.lease_until WHERE state='retry' OR (state='sending' AND lease_until <= ?)`, identifier, deviceID, date, slot, now.Add(time.Minute).Unix(), now.Unix())
	if err != nil {
		return "", err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed == 0 {
		return "", err
	}
	err = repository.database.QueryRowContext(ctx, `SELECT id FROM reminder_deliveries WHERE device_id=? AND local_date=? AND slot=?`, deviceID, date, slot).Scan(&identifier)
	return identifier, err
}

func (repository *SQLiteNotificationRepository) Finish(ctx context.Context, identifier string, retry bool) error {
	state := "finished"
	if retry {
		state = "retry"
	}
	_, err := repository.database.ExecContext(ctx, `UPDATE reminder_deliveries SET state=? WHERE id=?`, state, identifier)
	return err
}

func (repository *SQLiteNotificationRepository) Invalidate(ctx context.Context, device models.NotificationDevice) error {
	_, err := repository.database.ExecContext(ctx, `DELETE FROM notification_devices WHERE id=? AND token=? AND environment=?`, device.ID, device.Token, device.Environment)
	return err
}

func (repository *SQLiteNotificationRepository) Prune(ctx context.Context, before string) error {
	_, err := repository.database.ExecContext(ctx, `DELETE FROM reminder_deliveries WHERE local_date < ?`, before)
	if err != nil {
		return fmt.Errorf("pruning reminder deliveries: %w", err)
	}
	return nil
}
