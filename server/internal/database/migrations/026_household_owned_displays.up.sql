PRAGMA foreign_keys = OFF;

CREATE TABLE api_tokens_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    token_hash TEXT UNIQUE NOT NULL,
    scope TEXT NOT NULL DEFAULT 'api',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO api_tokens_new (id, name, token_hash, scope, created_by_user_id, expires_at, created_at)
SELECT id, name, token_hash, scope, created_by_user_id, expires_at, created_at FROM api_tokens;
DROP TABLE api_tokens;
ALTER TABLE api_tokens_new RENAME TO api_tokens;

CREATE TABLE paired_displays_new (
    id TEXT PRIMARY KEY REFERENCES display_pairing_challenges(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    api_token_id TEXT NOT NULL UNIQUE REFERENCES api_tokens(id) ON DELETE CASCADE,
    paired_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    enabled_sections TEXT NOT NULL DEFAULT '["calendar","meals","chores","weather","departures","trains","bins","shopping"]',
    commute_preferences TEXT NOT NULL DEFAULT ''
);
INSERT INTO paired_displays_new (id, name, api_token_id, paired_by_user_id, created_at, enabled_sections)
SELECT id, name, api_token_id, paired_by_user_id, created_at, enabled_sections FROM paired_displays;
UPDATE paired_displays_new SET commute_preferences = '{"toWork":{"enabled":false,"weekdays":[2,3,4,5,6],"start":"06:30","end":"09:30"},"toHome":{"enabled":false,"weekdays":[2,3,4,5,6],"start":"16:00","end":"19:30"},"homeStationCode":"","workStationCode":"","trainCount":3}';
DROP TABLE paired_displays;
ALTER TABLE paired_displays_new RENAME TO paired_displays;

CREATE TABLE meal_plans_new (
    date TEXT NOT NULL,
    meal_type TEXT NOT NULL CHECK (meal_type IN ('breakfast', 'lunch', 'dinner')),
    recipe_id TEXT REFERENCES recipes(id),
    name TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (date, meal_type)
);
INSERT INTO meal_plans_new (date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at)
SELECT date, meal_type, recipe_id, name, notes, created_by_user_id, created_at, updated_at FROM meal_plans;
DROP TABLE meal_plans;
ALTER TABLE meal_plans_new RENAME TO meal_plans;
CREATE INDEX idx_meal_plans_date ON meal_plans(date);
CREATE INDEX idx_meal_plans_recipe_id ON meal_plans(recipe_id);

PRAGMA foreign_keys = ON;
