-- Phase 4: chore_series is the definition/template for a recurring chore. It
-- owns the recurrence rule, end conditions, eligible-assignee pool and a durable
-- rotation cursor. Individual chore occurrences continue to live in `chores` and
-- reference their series via the existing series_id column. Backfill of existing
-- series is performed idempotently at startup (ChoreService.BackfillSeries).
CREATE TABLE chore_series (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT NOT NULL REFERENCES users(id),
    category_id TEXT REFERENCES categories(id) ON DELETE SET NULL,
    due_time TEXT,
    recurrence_type TEXT NOT NULL DEFAULT 'none'
        CHECK (recurrence_type IN ('none', 'daily', 'weekly', 'monthly', 'custom', 'calendar')),
    recurrence_value TEXT NOT NULL DEFAULT '',
    recur_on_complete BOOLEAN NOT NULL DEFAULT FALSE,
    recurrence_until TIMESTAMP,
    recurrence_count INTEGER,
    rotation_cursor_user_id TEXT REFERENCES users(id),
    deleted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chore_series_eligible_assignees (
    series_id TEXT NOT NULL REFERENCES chore_series(id) ON DELETE CASCADE,
    user_id   TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (series_id, user_id)
);
