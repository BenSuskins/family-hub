PRAGMA foreign_keys=OFF;
BEGIN IMMEDIATE;

INSERT INTO chore_series (id,name,description,created_by_user_id,category_id,due_time,recurrence_type,rotation_cursor_user_id,created_at,updated_at)
SELECT id,name,description,created_by_user_id,category_id,due_time,'none',assigned_to_user_id,created_at,updated_at
FROM chores WHERE series_id IS NULL;
INSERT INTO chore_series_eligible_assignees (series_id,user_id)
SELECT c.id,e.user_id FROM chores c JOIN chore_eligible_assignees e ON e.chore_id=c.id WHERE c.series_id IS NULL;
UPDATE chores SET series_id=id WHERE series_id IS NULL;

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
        CHECK (status IN ('pending', 'completed', 'overdue', 'skipped', 'superseded')),
    resolved_at TIMESTAMP,
    resolved_by_user_id TEXT REFERENCES users(id),
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


UPDATE chores SET resolved_at=completed_at, resolved_by_user_id=completed_by_user_id WHERE status='completed';
UPDATE chores SET status='superseded', resolved_at=CURRENT_TIMESTAMP WHERE id IN (
 SELECT id FROM (
  SELECT id, ROW_NUMBER() OVER (PARTITION BY series_id ORDER BY due_date ASC NULLS LAST,created_at,id) AS position
  FROM chores WHERE status IN ('pending','overdue')
 ) WHERE position>1
);
UPDATE chore_series SET rotation_cursor_user_id=(
 SELECT assigned_to_user_id FROM chores WHERE series_id=chore_series.id AND status IN ('pending','overdue') LIMIT 1
) WHERE EXISTS (SELECT 1 FROM chores WHERE series_id=chore_series.id AND status IN ('pending','overdue'));
UPDATE chore_assignments SET status='reassigned'
WHERE status='assigned' AND chore_id IN (SELECT id FROM chores WHERE status IN ('completed','superseded'));
INSERT INTO chore_assignments (id,chore_id,user_id,assigned_at,completed_at,status)
SELECT lower(hex(randomblob(16))),c.id,c.completed_by_user_id,COALESCE(c.completed_at,c.updated_at),c.completed_at,'completed'
FROM chores c WHERE c.status='completed' AND c.completed_by_user_id IS NOT NULL
 AND NOT EXISTS (SELECT 1 FROM chore_assignments a WHERE a.chore_id=c.id AND a.status='completed' AND a.user_id=c.completed_by_user_id);
CREATE UNIQUE INDEX idx_chores_one_outstanding ON chores(series_id) WHERE status IN ('pending','overdue');
COMMIT;
PRAGMA foreign_keys=ON;
