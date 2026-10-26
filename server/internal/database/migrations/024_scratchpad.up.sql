CREATE TABLE household_scratchpad (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    strokes_json TEXT NOT NULL DEFAULT '[]',
    updated_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO household_scratchpad (id, strokes_json) VALUES (1, '[]');
