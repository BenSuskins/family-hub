CREATE TABLE notification_devices (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    api_token_hash TEXT REFERENCES api_tokens(token_hash) ON DELETE CASCADE,
    environment TEXT NOT NULL CHECK (environment IN ('sandbox', 'production')),
    UNIQUE(token, environment)
);

CREATE TABLE reminder_deliveries (
    id TEXT PRIMARY KEY,
    device_id TEXT NOT NULL REFERENCES notification_devices(id) ON DELETE CASCADE,
    local_date TEXT NOT NULL,
    slot INTEGER NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('sending', 'retry', 'finished')),
    lease_until INTEGER NOT NULL,
    UNIQUE(device_id, local_date, slot)
);
