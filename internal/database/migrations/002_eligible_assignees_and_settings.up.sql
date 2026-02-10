CREATE TABLE chore_eligible_assignees (
    chore_id TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id),
    PRIMARY KEY (chore_id, user_id)
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO settings (key, value) VALUES ('family_name', 'Family');
