-- +goose Up
ALTER TABLE player_characters
ADD COLUMN availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive'));

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

-- +goose Down
DROP INDEX IF EXISTS idx_player_characters_campaign_availability;

CREATE TABLE player_characters_without_availability_status (
    id TEXT PRIMARY KEY,
    player_id TEXT NOT NULL,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 1,
    initiative INTEGER NOT NULL DEFAULT 1,
    hp INTEGER NOT NULL DEFAULT 1,
    max_hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    torso_only INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO player_characters_without_availability_status (
    id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense,
    torso_only, active, created_at, updated_at, deleted_at
)
SELECT
    id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense,
    torso_only, active, created_at, updated_at, deleted_at
FROM player_characters;

DROP TABLE player_characters;
ALTER TABLE player_characters_without_availability_status RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);
