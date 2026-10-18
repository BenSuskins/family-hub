CREATE TABLE notification_broadcasts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE broadcast_deliveries (
    id TEXT PRIMARY KEY,
    broadcast_id TEXT NOT NULL REFERENCES notification_broadcasts(id) ON DELETE CASCADE,
    device_id TEXT REFERENCES notification_devices(id) ON DELETE SET NULL,
    state TEXT NOT NULL DEFAULT 'pending',
    lease_until INTEGER NOT NULL DEFAULT 0,
    UNIQUE(broadcast_id, device_id)
);
