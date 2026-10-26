package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/google/uuid"
)

var ErrPairingUnavailable = errors.New("pairing challenge is invalid, expired, or already used")

type PairedDisplay struct {
	ID                 string                           `json:"id"`
	Name               string                           `json:"name"`
	PairedByUserID     string                           `json:"pairedByUserId"`
	CreatedAt          time.Time                        `json:"createdAt"`
	EnabledSections    []string                         `json:"enabledSections"`
	CommutePreferences models.DisplayCommutePreferences `json:"commutePreferences"`
}

type DisplayPairingRepository interface {
	CreateChallenge(context.Context, string, string, time.Time) error
	ApproveChallenge(context.Context, string, string, string, string) error
	ClaimChallenge(context.Context, string, string, models.APIToken) error
	List(context.Context) ([]PairedDisplay, error)
	UpdateSections(context.Context, string, []string) error
	UpdateDetails(context.Context, string, string) error
	UpdateCommutePreferences(context.Context, string, models.DisplayCommutePreferences) error
	FindByTokenID(context.Context, string) (PairedDisplay, error)
	FindByID(context.Context, string) (PairedDisplay, error)
	Revoke(context.Context, string) error
}

type SQLiteDisplayPairingRepository struct{ database *sql.DB }

func NewDisplayPairingRepository(database *sql.DB) *SQLiteDisplayPairingRepository {
	return &SQLiteDisplayPairingRepository{database: database}
}

func (repository *SQLiteDisplayPairingRepository) CreateChallenge(ctx context.Context, id, secretHash string, expiresAt time.Time) error {
	_, err := repository.database.ExecContext(ctx, `INSERT INTO display_pairing_challenges (id,secret_hash,expires_at) VALUES (?,?,?)`, id, secretHash, expiresAt)
	if err != nil {
		return fmt.Errorf("creating display pairing challenge: %w", err)
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) ApproveChallenge(ctx context.Context, id, secretHash, name, userID string) error {
	result, err := repository.database.ExecContext(ctx, `UPDATE display_pairing_challenges SET status='approved',display_name=?,approved_by_user_id=? WHERE id=? AND secret_hash=? AND status='pending' AND expires_at>CURRENT_TIMESTAMP`, name, userID, id, secretHash)
	if err != nil {
		return fmt.Errorf("approving display pairing challenge: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking display pairing approval: %w", err)
	}
	if count != 1 {
		return ErrPairingUnavailable
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) ClaimChallenge(ctx context.Context, id, secretHash string, token models.APIToken) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning display claim: %w", err)
	}
	defer transaction.Rollback()
	var name, userID string
	err = transaction.QueryRowContext(ctx, `SELECT display_name,approved_by_user_id FROM display_pairing_challenges WHERE id=? AND secret_hash=? AND status='approved' AND expires_at>CURRENT_TIMESTAMP`, id, secretHash).Scan(&name, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrPairingUnavailable
	}
	if err != nil {
		return fmt.Errorf("reading approved display challenge: %w", err)
	}
	if token.ID == "" {
		token.ID = uuid.NewString()
	}
	now := time.Now()
	if _, err = transaction.ExecContext(ctx, `INSERT INTO api_tokens (id,name,token_hash,scope,created_by_user_id,expires_at,created_at) VALUES (?,?,?,?,NULL,NULL,?)`, token.ID, "Wall display: "+name, token.TokenHash, models.TokenScopeDisplay, now); err != nil {
		return fmt.Errorf("creating display token: %w", err)
	}
	sections, _ := json.Marshal(models.DisplayDefaultSections)
	commute, _ := json.Marshal(models.DefaultDisplayCommutePreferences())
	if _, err = transaction.ExecContext(ctx, `INSERT INTO paired_displays (id,name,api_token_id,paired_by_user_id,created_at,enabled_sections,commute_preferences) VALUES (?,?,?,?,?,?,?)`, id, name, token.ID, userID, now, string(sections), string(commute)); err != nil {
		return fmt.Errorf("saving paired display: %w", err)
	}
	if _, err = transaction.ExecContext(ctx, `UPDATE display_pairing_challenges SET status='claimed',claimed_at=? WHERE id=? AND status='approved'`, now, id); err != nil {
		return fmt.Errorf("completing display claim: %w", err)
	}
	if err = transaction.Commit(); err != nil {
		return fmt.Errorf("committing display claim: %w", err)
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) List(ctx context.Context) ([]PairedDisplay, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT id,name,COALESCE(paired_by_user_id,''),created_at,enabled_sections,commute_preferences FROM paired_displays ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing paired displays: %w", err)
	}
	defer rows.Close()
	var displays []PairedDisplay
	for rows.Next() {
		var display PairedDisplay
		var sections, commute string
		if err := rows.Scan(&display.ID, &display.Name, &display.PairedByUserID, &display.CreatedAt, &sections, &commute); err != nil {
			return nil, fmt.Errorf("scanning paired display: %w", err)
		}
		if err := json.Unmarshal([]byte(sections), &display.EnabledSections); err != nil {
			return nil, fmt.Errorf("decoding paired display sections: %w", err)
		}
		if err := json.Unmarshal([]byte(commute), &display.CommutePreferences); err != nil {
			return nil, fmt.Errorf("decoding paired display commute settings: %w", err)
		}
		displays = append(displays, display)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading paired displays: %w", err)
	}
	if displays == nil {
		displays = []PairedDisplay{}
	}
	return displays, nil
}

func (repository *SQLiteDisplayPairingRepository) UpdateDetails(ctx context.Context, id, name string) error {
	result, err := repository.database.ExecContext(ctx, `UPDATE paired_displays SET name=? WHERE id=?`, name, id)
	if err != nil {
		return fmt.Errorf("updating display name: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking display name update: %w", err)
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) UpdateCommutePreferences(ctx context.Context, id string, preferences models.DisplayCommutePreferences) error {
	encoded, err := json.Marshal(preferences)
	if err != nil {
		return fmt.Errorf("encoding display commute settings: %w", err)
	}
	result, err := repository.database.ExecContext(ctx, `UPDATE paired_displays SET commute_preferences=? WHERE id=?`, string(encoded), id)
	if err != nil {
		return fmt.Errorf("updating display commute settings: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking display commute update: %w", err)
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) UpdateSections(ctx context.Context, id string, sections []string) error {
	encoded, err := json.Marshal(sections)
	if err != nil {
		return fmt.Errorf("encoding display sections: %w", err)
	}
	result, err := repository.database.ExecContext(ctx, `UPDATE paired_displays SET enabled_sections=? WHERE id=?`, string(encoded), id)
	if err != nil {
		return fmt.Errorf("updating display sections: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking display update: %w", err)
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (repository *SQLiteDisplayPairingRepository) FindByTokenID(ctx context.Context, tokenID string) (PairedDisplay, error) {
	return repository.find(ctx, `WHERE api_token_id=?`, tokenID)
}

func (repository *SQLiteDisplayPairingRepository) FindByID(ctx context.Context, id string) (PairedDisplay, error) {
	return repository.find(ctx, `WHERE id=?`, id)
}

func (repository *SQLiteDisplayPairingRepository) find(ctx context.Context, where string, value string) (PairedDisplay, error) {
	var display PairedDisplay
	var sections string
	var commute string
	err := repository.database.QueryRowContext(ctx, `SELECT id,name,COALESCE(paired_by_user_id,''),created_at,enabled_sections,commute_preferences FROM paired_displays `+where, value).Scan(&display.ID, &display.Name, &display.PairedByUserID, &display.CreatedAt, &sections, &commute)
	if err != nil {
		return PairedDisplay{}, err
	}
	if err := json.Unmarshal([]byte(sections), &display.EnabledSections); err != nil {
		return PairedDisplay{}, fmt.Errorf("decoding paired display sections: %w", err)
	}
	if err := json.Unmarshal([]byte(commute), &display.CommutePreferences); err != nil {
		return PairedDisplay{}, fmt.Errorf("decoding paired display commute settings: %w", err)
	}
	return display, nil
}

func (repository *SQLiteDisplayPairingRepository) Revoke(ctx context.Context, id string) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning display revocation: %w", err)
	}
	defer transaction.Rollback()
	var tokenID string
	if err := transaction.QueryRowContext(ctx, `SELECT api_token_id FROM paired_displays WHERE id=?`, id).Scan(&tokenID); errors.Is(err, sql.ErrNoRows) {
		return sql.ErrNoRows
	} else if err != nil {
		return fmt.Errorf("finding display token: %w", err)
	}
	if _, err := transaction.ExecContext(ctx, `DELETE FROM api_tokens WHERE id=?`, tokenID); err != nil {
		return fmt.Errorf("revoking display token: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("committing display revocation: %w", err)
	}
	return nil
}
