ALTER TABLE chores ADD COLUMN series_id TEXT;
CREATE INDEX idx_chores_series_id ON chores(series_id);
