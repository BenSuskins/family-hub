ALTER TABLE inventory_items RENAME COLUMN par TO low_at;
ALTER TABLE inventory_items ADD COLUMN tracking_mode TEXT NOT NULL DEFAULT 'count';
ALTER TABLE inventory_items ADD COLUMN level INTEGER NOT NULL DEFAULT 100;
