-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS encounters_v3;
DROP TABLE IF EXISTS combatants_v3;
DROP TABLE IF EXISTS encounter_logs_v3;
DROP TABLE IF EXISTS campaigns_v3;
DROP TABLE IF EXISTS players_v3;
DROP TABLE IF EXISTS player_characters_v3;
DROP TABLE IF EXISTS monster_templates_v3;
DROP TABLE IF EXISTS app_state_v3;

CREATE TABLE encounters_v3 (
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

INSERT INTO encounters_v3 (
    id, campaign_id, name, round, turn_index, party_ap, gm_threat,
    difficulty_label, difficulty_score, party_count, party_avg_level,
    party_xp_budget, enemy_count, enemy_avg_level, enemy_total_xp,
    created_at, updated_at, deleted_at
)
SELECT
    id,
    CASE WHEN campaign_id IS NULL OR trim(campaign_id) = '' THEN NULL ELSE campaign_id END,
    COALESCE(NULLIF(trim(name), ''), 'Untitled Encounter'),
    CASE WHEN round < 1 THEN 1 ELSE round END,
    CASE WHEN turn_index < 0 THEN 0 ELSE turn_index END,
    CASE WHEN party_ap < 0 THEN 0 ELSE party_ap END,
    CASE WHEN gm_threat < 0 THEN 0 ELSE gm_threat END,
    CASE
        WHEN difficulty_label IN ('Unknown', 'Trivial', 'Easy', 'Normal', 'Hard', 'Deadly') THEN difficulty_label
        ELSE 'Unknown'
    END,
    CASE WHEN difficulty_score < 0 THEN 0 ELSE difficulty_score END,
    CASE WHEN party_count < 0 THEN 0 ELSE party_count END,
    CASE WHEN party_avg_level < 0 THEN 0 ELSE party_avg_level END,
    CASE WHEN party_xp_budget < 0 THEN 0 ELSE party_xp_budget END,
    CASE WHEN enemy_count < 0 THEN 0 ELSE enemy_count END,
    CASE WHEN enemy_avg_level < 0 THEN 0 ELSE enemy_avg_level END,
    CASE WHEN enemy_total_xp < 0 THEN 0 ELSE enemy_total_xp END,
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM encounters;

DROP TABLE encounters;
ALTER TABLE encounters_v3 RENAME TO encounters;

CREATE TABLE combatants_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    player_character_id TEXT NULL,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    side TEXT NOT NULL CHECK (side IN ('party', 'npc')),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    initiative INTEGER NOT NULL CHECK (initiative >= 0),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
    defeated INTEGER NOT NULL DEFAULT 0 CHECK (defeated IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 0),
    xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp),
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE,
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id)
);

INSERT INTO combatants_v3 (
    id, encounter_id, player_character_id, name, side, torso_only, initiative,
    active, defeated, position, hp, max_hp, defense, level, xp,
    created_at, updated_at, deleted_at
)
SELECT
    id,
    encounter_id,
    player_character_id,
    COALESCE(NULLIF(trim(name), ''), id),
    CASE WHEN side IN ('party', 'npc') THEN side ELSE 'npc' END,
    CASE WHEN torso_only = 0 THEN 0 ELSE 1 END,
    CASE WHEN initiative < 0 THEN 0 ELSE initiative END,
    CASE WHEN active = 0 THEN 0 ELSE 1 END,
    CASE WHEN defeated = 0 THEN 0 ELSE 1 END,
    CASE WHEN position < 0 THEN 0 ELSE position END,
    CASE
        WHEN hp < 0 THEN 0
        WHEN hp > CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
            THEN CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
        ELSE hp
    END,
    CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END,
    CASE WHEN defense < 0 THEN 0 ELSE defense END,
    CASE WHEN level < 0 THEN 0 ELSE level END,
    CASE WHEN xp < 0 THEN 0 ELSE xp END,
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM combatants;

DROP TABLE combatants;
ALTER TABLE combatants_v3 RENAME TO combatants;

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_encounter_player_character
ON combatants(encounter_id, player_character_id)
WHERE player_character_id IS NOT NULL;

CREATE TABLE encounter_logs_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    round INTEGER NOT NULL CHECK (round >= 1),
    message TEXT NOT NULL CHECK (trim(message) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO encounter_logs_v3 (id, encounter_id, round, message, created_at, updated_at, deleted_at)
SELECT
    id,
    encounter_id,
    CASE WHEN round < 1 THEN 1 ELSE round END,
    COALESCE(NULLIF(trim(message), ''), 'Log entry'),
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM encounter_logs;

DROP TABLE encounter_logs;
ALTER TABLE encounter_logs_v3 RENAME TO encounter_logs;

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

CREATE TABLE campaigns_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    start_date DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

INSERT INTO campaigns_v3 (id, name, start_date, created_at, updated_at, deleted_at)
SELECT
    id,
    COALESCE(NULLIF(trim(name), ''), 'Untitled Campaign'),
    COALESCE(start_date, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM campaigns;

DROP TABLE campaigns;
ALTER TABLE campaigns_v3 RENAME TO campaigns;

CREATE TABLE players_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO players_v3 (id, campaign_id, name, created_at, updated_at, deleted_at)
SELECT
    id,
    campaign_id,
    COALESCE(NULLIF(trim(name), ''), id),
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM players;

DROP TABLE players;
ALTER TABLE players_v3 RENAME TO players;

CREATE UNIQUE INDEX IF NOT EXISTS idx_players_campaign_name
ON players(campaign_id, lower(trim(name)));

CREATE TABLE player_characters_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    player_id TEXT NOT NULL CHECK (trim(player_id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 1),
    initiative INTEGER NOT NULL DEFAULT 1 CHECK (initiative >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp),
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO player_characters_v3 (
    id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense,
    torso_only, active, availability_status, created_at, updated_at, deleted_at
)
SELECT
    id,
    player_id,
    campaign_id,
    COALESCE(NULLIF(trim(name), ''), id),
    CASE WHEN level < 1 THEN 1 ELSE level END,
    CASE WHEN initiative < 0 THEN 0 ELSE initiative END,
    CASE
        WHEN hp < 0 THEN 0
        WHEN hp > CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
            THEN CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
        ELSE hp
    END,
    CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END,
    CASE WHEN defense < 0 THEN 0 ELSE defense END,
    CASE WHEN torso_only = 0 THEN 0 ELSE 1 END,
    CASE WHEN active = 0 THEN 0 ELSE 1 END,
    CASE WHEN availability_status IN ('active', 'inactive') THEN availability_status ELSE 'active' END,
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM player_characters;

DROP TABLE player_characters;
ALTER TABLE player_characters_v3 RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

CREATE TABLE monster_templates_v3 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    name_key TEXT NOT NULL UNIQUE CHECK (trim(name_key) <> ''),
    torso_only INTEGER NOT NULL DEFAULT 1 CHECK (torso_only IN (0, 1)),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 1),
    xp INTEGER NOT NULL DEFAULT 0 CHECK (xp >= 0),
    initiative INTEGER NOT NULL DEFAULT 1 CHECK (initiative >= 0),
    hp INTEGER NOT NULL DEFAULT 1 CHECK (hp >= 0),
    max_hp INTEGER NOT NULL DEFAULT 1 CHECK (max_hp >= 1),
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    CHECK (hp <= max_hp)
);

INSERT INTO monster_templates_v3 (
    id, name, name_key, torso_only, level, xp, initiative, hp, max_hp,
    defense, created_at, updated_at, deleted_at
)
SELECT
    id,
    COALESCE(NULLIF(trim(name), ''), id),
    COALESCE(NULLIF(trim(name_key), ''), lower(trim(COALESCE(NULLIF(name, ''), id)))),
    CASE WHEN torso_only = 0 THEN 0 ELSE 1 END,
    CASE WHEN level < 1 THEN 1 ELSE level END,
    CASE WHEN xp < 0 THEN 0 ELSE xp END,
    CASE WHEN initiative < 0 THEN 0 ELSE initiative END,
    CASE
        WHEN hp < 0 THEN 0
        WHEN hp > CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
            THEN CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END
        ELSE hp
    END,
    CASE WHEN max_hp > 0 THEN max_hp WHEN hp > 0 THEN hp ELSE 1 END,
    CASE WHEN defense < 0 THEN 0 ELSE defense END,
    COALESCE(created_at, updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(updated_at, created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at
FROM monster_templates;

DROP TABLE monster_templates;
ALTER TABLE monster_templates_v3 RENAME TO monster_templates;

CREATE INDEX IF NOT EXISTS idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

CREATE TABLE app_state_v3 (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    active_campaign_id TEXT NULL,
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (active_campaign_id) REFERENCES campaigns(id)
);

INSERT INTO app_state_v3 (id, active_campaign_id, updated_at)
SELECT id, active_campaign_id, COALESCE(updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
FROM app_state;

DROP TABLE app_state;
ALTER TABLE app_state_v3 RENAME TO app_state;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS encounters_v2;
DROP TABLE IF EXISTS combatants_v2;
DROP TABLE IF EXISTS encounter_logs_v2;
DROP TABLE IF EXISTS campaigns_v2;
DROP TABLE IF EXISTS players_v2;
DROP TABLE IF EXISTS player_characters_v2;
DROP TABLE IF EXISTS monster_templates_v2;
DROP TABLE IF EXISTS app_state_v2;

CREATE TABLE encounters_v2 (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NULL,
    name TEXT NOT NULL,
    round INTEGER NOT NULL,
    turn_index INTEGER NOT NULL,
    party_ap INTEGER NOT NULL DEFAULT 0,
    gm_threat INTEGER NOT NULL DEFAULT 0,
    difficulty_label TEXT NOT NULL DEFAULT 'Unknown',
    difficulty_score REAL NOT NULL DEFAULT 0,
    party_count INTEGER NOT NULL DEFAULT 0,
    party_avg_level REAL NOT NULL DEFAULT 0,
    party_xp_budget INTEGER NOT NULL DEFAULT 0,
    enemy_count INTEGER NOT NULL DEFAULT 0,
    enemy_avg_level REAL NOT NULL DEFAULT 0,
    enemy_total_xp INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL
);

INSERT INTO encounters_v2 SELECT * FROM encounters;
DROP TABLE encounters;
ALTER TABLE encounters_v2 RENAME TO encounters;

CREATE TABLE combatants_v2 (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    player_character_id TEXT NULL,
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
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE,
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id)
);

INSERT INTO combatants_v2 SELECT * FROM combatants;
DROP TABLE combatants;
ALTER TABLE combatants_v2 RENAME TO combatants;

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_encounter_player_character
ON combatants(encounter_id, player_character_id)
WHERE player_character_id IS NOT NULL;

CREATE TABLE encounter_logs_v2 (
    id TEXT PRIMARY KEY,
    encounter_id TEXT NOT NULL,
    round INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE
);

INSERT INTO encounter_logs_v2 SELECT * FROM encounter_logs;
DROP TABLE encounter_logs;
ALTER TABLE encounter_logs_v2 RENAME TO encounter_logs;

CREATE INDEX IF NOT EXISTS idx_encounter_logs_encounter_created
ON encounter_logs(encounter_id, created_at DESC, id DESC);

CREATE TABLE campaigns_v2 (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    start_date DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

INSERT INTO campaigns_v2 SELECT * FROM campaigns;
DROP TABLE campaigns;
ALTER TABLE campaigns_v2 RENAME TO campaigns;

CREATE TABLE players_v2 (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO players_v2 SELECT * FROM players;
DROP TABLE players;
ALTER TABLE players_v2 RENAME TO players;

CREATE UNIQUE INDEX IF NOT EXISTS idx_players_campaign_name
ON players(campaign_id, lower(trim(name)));

CREATE TABLE player_characters_v2 (
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
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

INSERT INTO player_characters_v2 SELECT * FROM player_characters;
DROP TABLE player_characters;
ALTER TABLE player_characters_v2 RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

CREATE TABLE monster_templates_v2 (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    name_key TEXT NOT NULL UNIQUE,
    torso_only INTEGER NOT NULL DEFAULT 1,
    level INTEGER NOT NULL DEFAULT 1,
    xp INTEGER NOT NULL DEFAULT 0,
    initiative INTEGER NOT NULL DEFAULT 1,
    hp INTEGER NOT NULL DEFAULT 1,
    max_hp INTEGER NOT NULL DEFAULT 1,
    defense INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL
);

INSERT INTO monster_templates_v2 SELECT * FROM monster_templates;
DROP TABLE monster_templates;
ALTER TABLE monster_templates_v2 RENAME TO monster_templates;

CREATE INDEX IF NOT EXISTS idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

CREATE TABLE app_state_v2 (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    active_campaign_id TEXT NULL,
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    FOREIGN KEY (active_campaign_id) REFERENCES campaigns(id)
);

INSERT INTO app_state_v2 SELECT * FROM app_state;
DROP TABLE app_state;
ALTER TABLE app_state_v2 RENAME TO app_state;

PRAGMA foreign_keys = ON;
