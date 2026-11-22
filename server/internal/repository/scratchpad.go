package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	domain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
)

type SQLiteScratchpadRepository struct{ database *sql.DB }

func NewScratchpadRepository(database *sql.DB) *SQLiteScratchpadRepository {
	return &SQLiteScratchpadRepository{database: database}
}

type scratchpadPointRecord struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type scratchpadStrokeRecord struct {
	Color  string                  `json:"color"`
	Points []scratchpadPointRecord `json:"points"`
}

func fromScratchpadRecords(records []scratchpadStrokeRecord) []domain.Stroke {
	if records == nil {
		return nil
	}
	strokes := make([]domain.Stroke, len(records))
	for i, record := range records {
		strokes[i].Color = record.Color
		if record.Points != nil {
			strokes[i].Points = make([]domain.Point, len(record.Points))
			for j, point := range record.Points {
				strokes[i].Points[j] = domain.Point{X: point.X, Y: point.Y}
			}
		}
	}
	return strokes
}
func toScratchpadRecords(strokes []domain.Stroke) []scratchpadStrokeRecord {
	if strokes == nil {
		return nil
	}
	records := make([]scratchpadStrokeRecord, len(strokes))
	for i, stroke := range strokes {
		records[i].Color = stroke.Color
		if stroke.Points != nil {
			records[i].Points = make([]scratchpadPointRecord, len(stroke.Points))
			for j, point := range stroke.Points {
				records[i].Points[j] = scratchpadPointRecord{X: point.X, Y: point.Y}
			}
		}
	}
	return records
}
func (repository *SQLiteScratchpadRepository) Get(ctx context.Context) (domain.Snapshot, error) {
	var encoded string
	var updatedAt time.Time
	if err := repository.database.QueryRowContext(ctx, `SELECT strokes_json,updated_at FROM household_scratchpad WHERE id=1`).Scan(&encoded, &updatedAt); err != nil {
		return domain.Snapshot{}, fmt.Errorf("reading scratchpad: %w", err)
	}
	var records []scratchpadStrokeRecord
	if err := json.Unmarshal([]byte(encoded), &records); err != nil {
		return domain.Snapshot{}, fmt.Errorf("decoding scratchpad: %w", err)
	}
	strokes := fromScratchpadRecords(records)
	if strokes == nil {
		strokes = []domain.Stroke{}
	}
	return domain.Snapshot{Strokes: strokes, UpdatedAt: updatedAt}, nil
}
func (repository *SQLiteScratchpadRepository) Save(ctx context.Context, strokes []domain.Stroke, userID string) (domain.Snapshot, error) {
	encoded, err := json.Marshal(toScratchpadRecords(strokes))
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("encoding scratchpad: %w", err)
	}
	now := time.Now().UTC()
	_, err = repository.database.ExecContext(ctx, `INSERT INTO household_scratchpad (id,strokes_json,updated_by_user_id,updated_at) VALUES (1,?,?,?) ON CONFLICT(id) DO UPDATE SET strokes_json=excluded.strokes_json,updated_by_user_id=excluded.updated_by_user_id,updated_at=excluded.updated_at`, string(encoded), userID, now)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("saving scratchpad: %w", err)
	}
	return domain.Snapshot{Strokes: domain.CopyStrokes(strokes), UpdatedAt: now}, nil
}
