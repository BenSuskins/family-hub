CREATE TABLE shopping_items (
 id TEXT PRIMARY KEY,
 name TEXT NOT NULL,
 inventory_item_id TEXT REFERENCES inventory_items(id) ON DELETE SET NULL,
 purchased INTEGER NOT NULL DEFAULT 0,
 purchase_id TEXT,
 restocked INTEGER NOT NULL DEFAULT 0,
 version INTEGER NOT NULL DEFAULT 0,
 created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX shopping_active_inventory ON shopping_items(inventory_item_id)
 WHERE inventory_item_id IS NOT NULL AND purchased = 0;
CREATE TABLE shopping_purchases (
 id TEXT PRIMARY KEY,
 shopping_item_id TEXT NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
 restocked_at DATETIME
);
