-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_combatants_delete_stat_profile;
DROP TRIGGER IF EXISTS trg_player_characters_delete_stat_profile;
DROP TRIGGER IF EXISTS trg_monster_templates_delete_stat_profile;
DROP TRIGGER IF EXISTS trg_player_characters_require_level;
DROP TRIGGER IF EXISTS trg_monster_templates_require_level;
DROP TRIGGER IF EXISTS trg_stat_profiles_player_character_level_update;
DROP TRIGGER IF EXISTS trg_stat_profiles_monster_template_level_update;

DROP TABLE IF EXISTS stat_profiles_v1;
DROP TABLE IF EXISTS stat_profile_resistance_global_v1;
DROP TABLE IF EXISTS stat_profile_resistance_by_location_v1;
DROP TABLE IF EXISTS body_locations_v2;
DROP TABLE IF EXISTS damage_types_v2;
DROP TABLE IF EXISTS combatants_v4;
DROP TABLE IF EXISTS player_characters_v4;
DROP TABLE IF EXISTS monster_templates_v4;

CREATE TABLE body_locations_v2 (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN ('head', 'torso', 'left_arm', 'right_arm', 'left_leg', 'right_leg'))
);

INSERT INTO body_locations_v2 (id, code) VALUES
    (1, 'head'),
    (2, 'torso'),
    (3, 'left_arm'),
    (4, 'right_arm'),
    (5, 'left_leg'),
    (6, 'right_leg');

DROP TABLE IF EXISTS body_locations;
ALTER TABLE body_locations_v2 RENAME TO body_locations;

CREATE TABLE damage_types_v2 (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN ('physical', 'energy', 'radiation', 'poison'))
);

INSERT INTO damage_types_v2 (id, code) VALUES
    (1, 'physical'),
    (2, 'energy'),
    (3, 'radiation'),
    (4, 'poison');

DROP TABLE IF EXISTS damage_types;
ALTER TABLE damage_types_v2 RENAME TO damage_types;

CREATE TABLE stat_profiles_v1 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    torso_only INTEGER NOT NULL DEFAULT 0 CHECK (torso_only IN (0, 1)),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level >= 0),
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

INSERT INTO stat_profiles_v1 (
    id, torso_only, level, xp, initiative, hp, max_hp, defense, created_at, updated_at, deleted_at
)
SELECT
    'combatant:' || id,
    torso_only,
    level,
    xp,
    initiative,
    hp,
    max_hp,
    defense,
    created_at,
    updated_at,
    deleted_at
FROM combatants;

INSERT INTO stat_profiles_v1 (
    id, torso_only, level, xp, initiative, hp, max_hp, defense, created_at, updated_at, deleted_at
)
SELECT
    'player_character:' || id,
    torso_only,
    level,
    0,
    initiative,
    hp,
    max_hp,
    defense,
    created_at,
    updated_at,
    deleted_at
FROM player_characters;

INSERT INTO stat_profiles_v1 (
    id, torso_only, level, xp, initiative, hp, max_hp, defense, created_at, updated_at, deleted_at
)
SELECT
    'monster_template:' || id,
    torso_only,
    level,
    xp,
    initiative,
    hp,
    max_hp,
    defense,
    created_at,
    updated_at,
    deleted_at
FROM monster_templates;

CREATE TABLE stat_profile_resistance_global_v1 (
    stat_profile_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (stat_profile_id, damage_type_id),
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

CREATE TABLE stat_profile_resistance_by_location_v1 (
    stat_profile_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (stat_profile_id, damage_type_id, body_location_id),
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

INSERT INTO stat_profile_resistance_global_v1 (
    stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
)
SELECT
    'combatant:' || combatant_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
FROM combatant_resistance_global;

INSERT INTO stat_profile_resistance_global_v1 (
    stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
)
SELECT
    'player_character:' || player_character_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
FROM player_character_resistance_global;

INSERT INTO stat_profile_resistance_global_v1 (
    stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
)
SELECT
    'monster_template:' || monster_template_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
FROM monster_template_resistance_global;

INSERT INTO stat_profile_resistance_by_location_v1 (
    stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
)
SELECT
    'combatant:' || combatant_id,
    damage_type_id,
    body_location_id,
    resistance,
    created_at,
    updated_at
FROM combatant_resistance_by_location;

INSERT INTO stat_profile_resistance_by_location_v1 (
    stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
)
SELECT
    'player_character:' || player_character_id,
    damage_type_id,
    body_location_id,
    resistance,
    created_at,
    updated_at
FROM player_character_resistance_by_location;

INSERT INTO stat_profile_resistance_by_location_v1 (
    stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
)
SELECT
    'monster_template:' || monster_template_id,
    damage_type_id,
    body_location_id,
    resistance,
    created_at,
    updated_at
FROM monster_template_resistance_by_location;

DROP TABLE combatant_resistance_by_location;
DROP TABLE combatant_resistance_global;
DROP TABLE player_character_resistance_by_location;
DROP TABLE player_character_resistance_global;
DROP TABLE monster_template_resistance_by_location;
DROP TABLE monster_template_resistance_global;

DROP TABLE IF EXISTS stat_profiles;
ALTER TABLE stat_profiles_v1 RENAME TO stat_profiles;
ALTER TABLE stat_profile_resistance_global_v1 RENAME TO stat_profile_resistance_global;
ALTER TABLE stat_profile_resistance_by_location_v1 RENAME TO stat_profile_resistance_by_location;

CREATE TABLE combatants_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    encounter_id TEXT NOT NULL CHECK (trim(encounter_id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    player_character_id TEXT NULL,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    side TEXT NOT NULL CHECK (side IN ('party', 'npc')),
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
    defeated INTEGER NOT NULL DEFAULT 0 CHECK (defeated IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (encounter_id) REFERENCES encounters(id) ON DELETE CASCADE,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id)
);

INSERT INTO combatants_v4 (
    id, encounter_id, stat_profile_id, player_character_id, name, side,
    active, defeated, position, created_at, updated_at, deleted_at
)
SELECT
    id,
    encounter_id,
    'combatant:' || id,
    player_character_id,
    name,
    side,
    active,
    defeated,
    position,
    created_at,
    updated_at,
    deleted_at
FROM combatants;

DROP TABLE combatants;
ALTER TABLE combatants_v4 RENAME TO combatants;

CREATE INDEX IF NOT EXISTS idx_combatants_encounter_position
ON combatants(encounter_id, position);

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_encounter_player_character
ON combatants(encounter_id, player_character_id)
WHERE player_character_id IS NOT NULL;

CREATE TABLE player_characters_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    player_id TEXT NOT NULL CHECK (trim(player_id) <> ''),
    campaign_id TEXT NOT NULL CHECK (trim(campaign_id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
);

INSERT INTO player_characters_v4 (
    id, player_id, campaign_id, stat_profile_id, name, active,
    availability_status, created_at, updated_at, deleted_at
)
SELECT
    id,
    player_id,
    campaign_id,
    'player_character:' || id,
    name,
    active,
    availability_status,
    created_at,
    updated_at,
    deleted_at
FROM player_characters;

DROP TABLE player_characters;
ALTER TABLE player_characters_v4 RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

CREATE TABLE monster_templates_v4 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    name_key TEXT NOT NULL UNIQUE CHECK (trim(name_key) <> ''),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
);

INSERT INTO monster_templates_v4 (
    id, stat_profile_id, name, name_key, created_at, updated_at, deleted_at
)
SELECT
    id,
    'monster_template:' || id,
    name,
    name_key,
    created_at,
    updated_at,
    deleted_at
FROM monster_templates;

DROP TABLE monster_templates;
ALTER TABLE monster_templates_v4 RENAME TO monster_templates;

CREATE INDEX IF NOT EXISTS idx_monster_templates_deleted_name
ON monster_templates(deleted_at, name COLLATE NOCASE);

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_global_poison_only_insert
BEFORE INSERT ON stat_profile_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON stat_profile_resistance_global
WHEN NEW.resistance <> 0
  AND NOT EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'non-poison global resistance must be zero');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_combatants_delete_stat_profile
AFTER DELETE ON combatants
BEGIN
    DELETE FROM stat_profiles
    WHERE id = OLD.stat_profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_delete_stat_profile
AFTER DELETE ON player_characters
BEGIN
    DELETE FROM stat_profiles
    WHERE id = OLD.stat_profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_delete_stat_profile
AFTER DELETE ON monster_templates
BEGIN
    DELETE FROM stat_profiles
    WHERE id = OLD.stat_profile_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_require_level
BEFORE INSERT ON player_characters
WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1
BEGIN
    SELECT RAISE(ABORT, 'player character level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_monster_templates_require_level
BEFORE INSERT ON monster_templates
WHEN (SELECT level FROM stat_profiles WHERE id = NEW.stat_profile_id) < 1
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profiles_player_character_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM player_characters pc WHERE pc.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'player character level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profiles_monster_template_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'monster template level must be at least 1');
END;
-- +goose StatementEnd

PRAGMA foreign_keys = ON;

-- +goose Down
SELECT 1;
