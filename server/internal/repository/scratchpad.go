package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

type ScratchpadRepository interface {
	Get(context.Context) (models.Scratchpad, error)
	Save(context.Context, []models.ScratchpadStroke, string) (models.Scratchpad, error)
}

type SQLiteScratchpadRepository struct{ database *sql.DB }

func NewScratchpadRepository(database *sql.DB) *SQLiteScratchpadRepository {
	return &SQLiteScratchpadRepository{database: database}
}

func (repository *SQLiteScratchpadRepository) Get(ctx context.Context) (models.Scratchpad, error) {
	var encoded string
	var scratchpad models.Scratchpad
	if err := repository.database.QueryRowContext(ctx, `SELECT strokes_json,updated_at FROM household_scratchpad WHERE id=1`).Scan(&encoded, &scratchpad.UpdatedAt); err != nil {
		return models.Scratchpad{}, fmt.Errorf("reading scratchpad: %w", err)
	}
	if err := json.Unmarshal([]byte(encoded), &scratchpad.Strokes); err != nil {
		return models.Scratchpad{}, fmt.Errorf("decoding scratchpad: %w", err)
	}
	if scratchpad.Strokes == nil {
		scratchpad.Strokes = []models.ScratchpadStroke{}
	}
	return scratchpad, nil
}

func (repository *SQLiteScratchpadRepository) Save(ctx context.Context, strokes []models.ScratchpadStroke, userID string) (models.Scratchpad, error) {
	encoded, err := json.Marshal(strokes)
	if err != nil {
		return models.Scratchpad{}, fmt.Errorf("encoding scratchpad: %w", err)
	}
	now := time.Now().UTC()
	_, err = repository.database.ExecContext(ctx, `INSERT INTO household_scratchpad (id,strokes_json,updated_by_user_id,updated_at) VALUES (1,?,?,?) ON CONFLICT(id) DO UPDATE SET strokes_json=excluded.strokes_json,updated_by_user_id=excluded.updated_by_user_id,updated_at=excluded.updated_at`, string(encoded), userID, now)
	if err != nil {
		return models.Scratchpad{}, fmt.Errorf("saving scratchpad: %w", err)
	}
	return models.Scratchpad{Strokes: strokes, UpdatedAt: now}, nil
}
