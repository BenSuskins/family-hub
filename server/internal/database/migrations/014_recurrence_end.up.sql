-- Recurrence end conditions. Stored per occurrence row, consistent with the
-- existing series-of-rows model; a future chore_series table will own these.
ALTER TABLE chores ADD COLUMN recurrence_until TIMESTAMP;
ALTER TABLE chores ADD COLUMN recurrence_count INTEGER;
