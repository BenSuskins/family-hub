-- Phase 4d: make chore_series the sole owner of the recurrence rule. Backfill
-- any missing series definitions from the rule columns, then rebuild the chores
-- table to drop those now-redundant columns and add a real foreign key on
-- series_id. Runs without a transaction wrapper (NoTxWrap) so foreign_keys can
-- be toggled off during the rebuild — otherwise DROP TABLE chores would
-- cascade-delete chore_assignments and chore_eligible_assignees.

PRAGMA foreign_keys=OFF;

-- Backfill a series definition for every recurring series that lacks one, taken
-- from the series anchor (earliest occurrence). Must precede the column drop so
-- no recurrence data is lost.
INSERT INTO chore_series (
    id, name, description, created_by_user_id, category_id, due_time,
    recurrence_type, recurrence_value, recur_on_complete, recurrence_until, recurrence_count,
    rotation_cursor_user_id, created_at, updated_at)
SELECT a.sid, a.name, a.description, a.created_by_user_id, a.category_id, a.due_time,
    a.recurrence_type, a.recurrence_value, a.recur_on_complete, a.recurrence_until, a.recurrence_count,
    a.assigned_to_user_id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
FROM (
    SELECT COALESCE(series_id, id) AS sid, id, name, description, created_by_user_id, category_id, due_time,
        recurrence_type, recurrence_value, recur_on_complete, recurrence_until, recurrence_count, assigned_to_user_id,
        ROW_NUMBER() OVER (PARTITION BY COALESCE(series_id, id) ORDER BY due_date ASC, id ASC) AS rn
    FROM chores
    WHERE recurrence_type != 'none'
) a
WHERE a.rn = 1
  AND a.sid NOT IN (SELECT id FROM chore_series);

-- Copy the anchor's eligible pool onto the (newly) backfilled series.
INSERT INTO chore_series_eligible_assignees (series_id, user_id)
SELECT a.sid, cea.user_id
FROM (
    SELECT COALESCE(series_id, id) AS sid, id,
        ROW_NUMBER() OVER (PARTITION BY COALESCE(series_id, id) ORDER BY due_date ASC, id ASC) AS rn
    FROM chores
    WHERE recurrence_type != 'none'
) a
JOIN chore_eligible_assignees cea ON cea.chore_id = a.id
WHERE a.rn = 1
  AND NOT EXISTS (
    SELECT 1 FROM chore_series_eligible_assignees x
    WHERE x.series_id = a.sid AND x.user_id = cea.user_id
  );

-- Ensure every recurring occurrence references its series.
UPDATE chores SET series_id = id WHERE recurrence_type != 'none' AND series_id IS NULL;

-- Rebuild chores: drop the series-owned recurrence columns and add a real
-- foreign key on series_id (nullable: non-recurring chores have no series).
CREATE TABLE chores_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT NOT NULL REFERENCES users(id),
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    assigned_to_user_id TEXT REFERENCES users(id),
    last_assigned_index INTEGER NOT NULL DEFAULT 0,
    due_date TIMESTAMP,
    due_time TEXT,
    series_id TEXT REFERENCES chore_series(id),
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'completed', 'overdue')),
    completed_at TIMESTAMP,
    completed_by_user_id TEXT REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO chores_new (
    id, name, description, created_by_user_id, category_id,
    assigned_to_user_id, last_assigned_index, due_date, due_time, series_id,
    status, completed_at, completed_by_user_id, created_at, updated_at)
SELECT
    id, name, description, created_by_user_id, category_id,
    assigned_to_user_id, last_assigned_index, due_date, due_time, series_id,
    status, completed_at, completed_by_user_id, created_at, updated_at
FROM chores;

DROP TABLE chores;
ALTER TABLE chores_new RENAME TO chores;

CREATE INDEX idx_chores_status ON chores(status);
CREATE INDEX idx_chores_assigned_to ON chores(assigned_to_user_id);
CREATE INDEX idx_chores_due_date ON chores(due_date);
CREATE INDEX idx_chores_series_id ON chores(series_id);

PRAGMA foreign_keys=ON;
