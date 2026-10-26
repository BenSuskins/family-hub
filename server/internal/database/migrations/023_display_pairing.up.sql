CREATE TABLE display_pairing_challenges (
    id TEXT PRIMARY KEY,
    secret_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'claimed')),
    display_name TEXT NOT NULL DEFAULT 'Family display',
    approved_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMP NOT NULL,
    claimed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_display_pairing_expiry ON display_pairing_challenges(expires_at);

CREATE TABLE paired_displays (
    id TEXT PRIMARY KEY REFERENCES display_pairing_challenges(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    api_token_id TEXT NOT NULL UNIQUE REFERENCES api_tokens(id) ON DELETE CASCADE,
    paired_by_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
