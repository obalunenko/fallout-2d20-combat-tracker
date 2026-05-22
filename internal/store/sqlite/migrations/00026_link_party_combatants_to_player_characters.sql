-- +goose Up
ALTER TABLE combatants ADD COLUMN player_character_id TEXT NULL;

UPDATE combatants
SET player_character_id = id
WHERE side = 'party'
  AND EXISTS (
    SELECT 1
    FROM player_characters pc
    JOIN encounters e ON e.id = combatants.encounter_id
    WHERE pc.id = combatants.id
      AND pc.campaign_id = e.campaign_id
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_encounter_player_character
ON combatants(encounter_id, player_character_id)
WHERE player_character_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_combatants_encounter_player_character;

CREATE TABLE combatants_without_player_character_id (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    name TEXT NOT NULL,
    side TEXT NOT NULL,
    torso_only INTEGER NOT NULL DEFAULT 0,
    initiative INTEGER NOT NULL,
    active INTEGER NOT NULL DEFAULT 0,
    defeated INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL,
    hp INTEGER NOT NULL DEFAULT 1,
    max_hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    xp INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO combatants_without_player_character_id (
    id, encounter_id, name, side, torso_only, initiative, active, defeated, position,
    hp, max_hp, defense, level, xp, created_at, updated_at, deleted_at
)
SELECT
    id, encounter_id, name, side, torso_only, initiative, active, defeated, position,
    hp, max_hp, defense, level, xp, created_at, updated_at, deleted_at
FROM combatants;

DROP TABLE combatants;
ALTER TABLE combatants_without_player_character_id RENAME TO combatants;

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);
