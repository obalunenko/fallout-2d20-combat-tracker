-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

INSERT OR IGNORE INTO campaigns (id, name, start_date, created_at, updated_at)
SELECT
    '00000000-0000-0000-0000-000000000001',
    'Legacy Campaign',
    STRFTIME('%Y-%m-%d 00:00:00.000', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE EXISTS (
    SELECT 1
    FROM encounters
    WHERE campaign_id IS NULL OR trim(campaign_id) = ''
);

UPDATE encounters
SET campaign_id = '00000000-0000-0000-0000-000000000001'
WHERE campaign_id IS NULL OR trim(campaign_id) = '';

INSERT OR IGNORE INTO campaigns (id, name, start_date, created_at, updated_at)
SELECT DISTINCT
    trim(e.campaign_id),
    'Recovered Campaign ' || trim(e.campaign_id),
    STRFTIME('%Y-%m-%d 00:00:00.000', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM encounters e
LEFT JOIN campaigns c ON c.id = trim(e.campaign_id)
WHERE e.campaign_id IS NOT NULL
  AND trim(e.campaign_id) <> ''
  AND c.id IS NULL;

UPDATE player_characters
SET campaign_id = (
    SELECT p.campaign_id
    FROM players p
    WHERE p.id = player_characters.player_id
)
WHERE EXISTS (
    SELECT 1
    FROM players p
    WHERE p.id = player_characters.player_id
      AND p.campaign_id <> player_characters.campaign_id
);

UPDATE combatants
SET player_character_id = NULL
WHERE player_character_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM encounters e
    JOIN player_characters pc ON pc.id = combatants.player_character_id
    WHERE e.id = combatants.encounter_id
      AND pc.campaign_id = e.campaign_id
  );

DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_insert;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_update;
DROP TRIGGER IF EXISTS trg_players_campaign_update_keeps_characters_consistent;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_insert;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_update;
DROP TRIGGER IF EXISTS trg_encounters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_update_keeps_combatants_consistent;

DROP TABLE IF EXISTS encounters_v4;

CREATE TABLE encounters_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    turn_index INTEGER NOT NULL CHECK (turn_index >= 0),
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown' CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly')),
    difficulty_score REAL NOT NULL DEFAULT 0 CHECK (difficulty_score >= 0),
    party_count INTEGER NOT NULL DEFAULT 0 CHECK (party_count >= 0),
    party_avg_level REAL NOT NULL DEFAULT 0 CHECK (party_avg_level >= 0),
    party_xp_budget INTEGER NOT NULL DEFAULT 0 CHECK (party_xp_budget >= 0),
    enemy_count INTEGER NOT NULL DEFAULT 0 CHECK (enemy_count >= 0),
    enemy_avg_level REAL NOT NULL DEFAULT 0 CHECK (enemy_avg_level >= 0),
    enemy_total_xp INTEGER NOT NULL DEFAULT 0 CHECK (enemy_total_xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO encounters_v4 (
    id, campaign_id, name, round, turn_index, party_ap, gm_threat,
    difficulty_label, difficulty_score, party_count, party_avg_level,
    party_xp_budget, enemy_count, enemy_avg_level, enemy_total_xp,
    created_at, updated_at, deleted_at
)
SELECT
    id,
    trim(campaign_id),
    name,
    round,
    turn_index,
    party_ap,
    gm_threat,
    difficulty_label,
    difficulty_score,
    party_count,
    party_avg_level,
    party_xp_budget,
    enemy_count,
    enemy_avg_level,
    enemy_total_xp,
    created_at,
    updated_at,
    deleted_at
FROM encounters;

DROP TABLE encounters;
ALTER TABLE encounters_v4 RENAME TO encounters;

CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC, id DESC);

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_campaign_matches_player_insert
BEFORE INSERT ON player_characters
WHEN NOT EXISTS (
    SELECT 1
    FROM players p
    WHERE p.id = NEW.player_id
      AND p.campaign_id = NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player character campaign must match player campaign');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_campaign_matches_player_update
BEFORE UPDATE OF player_id, campaign_id ON player_characters
WHEN NOT EXISTS (
    SELECT 1
    FROM players p
    WHERE p.id = NEW.player_id
      AND p.campaign_id = NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player character campaign must match player campaign');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_players_campaign_update_keeps_characters_consistent
BEFORE UPDATE OF campaign_id ON players
WHEN EXISTS (
    SELECT 1
    FROM player_characters pc
    WHERE pc.player_id = NEW.id
      AND pc.campaign_id <> NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player campaign update would mismatch player characters');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_combatants_player_character_campaign_insert
BEFORE INSERT ON combatants
WHEN NEW.player_character_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM encounters e
    JOIN player_characters pc ON pc.id = NEW.player_character_id
    WHERE e.id = NEW.encounter_id
      AND pc.campaign_id = e.campaign_id
  )
BEGIN
    SELECT RAISE(ABORT, 'combatant player character must belong to encounter campaign');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_combatants_player_character_campaign_update
BEFORE UPDATE OF encounter_id, player_character_id ON combatants
WHEN NEW.player_character_id IS NOT NULL
  AND NOT EXISTS (
    SELECT 1
    FROM encounters e
    JOIN player_characters pc ON pc.id = NEW.player_character_id
    WHERE e.id = NEW.encounter_id
      AND pc.campaign_id = e.campaign_id
  )
BEGIN
    SELECT RAISE(ABORT, 'combatant player character must belong to encounter campaign');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_encounters_campaign_update_keeps_combatants_consistent
BEFORE UPDATE OF campaign_id ON encounters
WHEN EXISTS (
    SELECT 1
    FROM combatants c
    JOIN player_characters pc ON pc.id = c.player_character_id
    WHERE c.encounter_id = NEW.id
      AND pc.campaign_id <> NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'encounter campaign update would mismatch linked combatants');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_campaign_update_keeps_combatants_consistent
BEFORE UPDATE OF campaign_id ON player_characters
WHEN EXISTS (
    SELECT 1
    FROM combatants c
    JOIN encounters e ON e.id = c.encounter_id
    WHERE c.player_character_id = NEW.id
      AND e.campaign_id <> NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player character campaign update would mismatch linked combatants');
END;
-- +goose StatementEnd

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_player_characters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_encounters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_update;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_insert;
DROP TRIGGER IF EXISTS trg_players_campaign_update_keeps_characters_consistent;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_update;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_insert;

DROP TABLE IF EXISTS encounters_v4_down;

CREATE TABLE encounters_v4_down (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NULL CHECK (campaign_id IS NULL OR trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    turn_index INTEGER NOT NULL CHECK (turn_index >= 0),
    party_ap INTEGER NOT NULL DEFAULT 0 CHECK (party_ap >= 0),
    gm_threat INTEGER NOT NULL DEFAULT 0 CHECK (gm_threat >= 0),
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown' CHECK (difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly')),
    difficulty_score REAL NOT NULL DEFAULT 0 CHECK (difficulty_score >= 0),
    party_count INTEGER NOT NULL DEFAULT 0 CHECK (party_count >= 0),
    party_avg_level REAL NOT NULL DEFAULT 0 CHECK (party_avg_level >= 0),
    party_xp_budget INTEGER NOT NULL DEFAULT 0 CHECK (party_xp_budget >= 0),
    enemy_count INTEGER NOT NULL DEFAULT 0 CHECK (enemy_count >= 0),
    enemy_avg_level REAL NOT NULL DEFAULT 0 CHECK (enemy_avg_level >= 0),
    enemy_total_xp INTEGER NOT NULL DEFAULT 0 CHECK (enemy_total_xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

INSERT INTO encounters_v4_down (
    id, campaign_id, name, round, turn_index, party_ap, gm_threat,
    difficulty_label, difficulty_score, party_count, party_avg_level,
    party_xp_budget, enemy_count, enemy_avg_level, enemy_total_xp,
    created_at, updated_at, deleted_at
)
SELECT
    id,
    campaign_id,
    name,
    round,
    turn_index,
    party_ap,
    gm_threat,
    difficulty_label,
    difficulty_score,
    party_count,
    party_avg_level,
    party_xp_budget,
    enemy_count,
    enemy_avg_level,
    enemy_total_xp,
    created_at,
    updated_at,
    deleted_at
FROM encounters;

DROP TABLE encounters;
ALTER TABLE encounters_v4_down RENAME TO encounters;

CREATE INDEX IF NOT EXISTS idx_encounters_campaign_deleted_updated
ON encounters(campaign_id, deleted_at, updated_at DESC, id DESC);

PRAGMA foreign_keys = ON;
