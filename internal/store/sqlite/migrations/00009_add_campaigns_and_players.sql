-- +goose Up
CREATE TABLE IF NOT EXISTS campaigns (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    start_date TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
);

CREATE TABLE IF NOT EXISTS app_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    active_campaign_id TEXT NULL,
    FOREIGN KEY (active_campaign_id) REFERENCES campaigns(id)
);

INSERT OR IGNORE INTO app_state (id, active_campaign_id) VALUES (1, NULL);

ALTER TABLE encounters ADD COLUMN campaign_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC);

CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_players_campaign_name
ON players(campaign_id, lower(trim(name)));

CREATE TABLE IF NOT EXISTS player_characters (
    id TEXT PRIMARY KEY,
    player_id TEXT NOT NULL,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    initiative INTEGER NOT NULL DEFAULT 1,
    hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    damage_resistance_physical INTEGER NOT NULL DEFAULT 0,
    damage_resistance_energy INTEGER NOT NULL DEFAULT 0,
    damage_resistance_radiation INTEGER NOT NULL DEFAULT 0,
    damage_resistance_poison INTEGER NOT NULL DEFAULT 0,
    damage_resistance_physical_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_energy_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_radiation_immune INTEGER NOT NULL DEFAULT 0,
    damage_resistance_poison_immune INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);

INSERT OR IGNORE INTO campaigns (id, name, start_date, created_at, updated_at)
SELECT
    '00000000-0000-0000-0000-000000000001',
    'Legacy Campaign',
    DATE('now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE EXISTS (
    SELECT 1 FROM encounters
);

UPDATE encounters
SET campaign_id = '00000000-0000-0000-0000-000000000001'
WHERE campaign_id IS NULL;

UPDATE app_state
SET active_campaign_id = (
    SELECT id
    FROM campaigns
    ORDER BY updated_at DESC, id DESC
    LIMIT 1
)
WHERE id = 1 AND active_campaign_id IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_player_characters_campaign_active;
DROP INDEX IF EXISTS idx_player_characters_one_active;
DROP TABLE IF EXISTS player_characters;

DROP INDEX IF EXISTS idx_players_campaign_name;
DROP TABLE IF EXISTS players;

DROP INDEX IF EXISTS idx_encounters_campaign_deleted_updated;
ALTER TABLE encounters DROP COLUMN campaign_id;

DROP TABLE IF EXISTS app_state;
DROP TABLE IF EXISTS campaigns;
