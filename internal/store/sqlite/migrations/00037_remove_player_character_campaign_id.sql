-- +goose NO TRANSACTION
-- +goose Up
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_player_characters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_encounters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_update;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_insert;
DROP TRIGGER IF EXISTS trg_players_campaign_update_keeps_characters_consistent;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_update;
DROP TRIGGER IF EXISTS trg_player_characters_campaign_matches_player_insert;
DROP TRIGGER IF EXISTS trg_player_characters_player_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_stat_profiles_player_character_level_update;
DROP TRIGGER IF EXISTS trg_player_characters_require_level;
DROP TRIGGER IF EXISTS trg_player_characters_delete_stat_profile;
DROP VIEW IF EXISTS player_character_resistance_by_location;
DROP VIEW IF EXISTS player_character_resistance_global;

DROP TABLE IF EXISTS player_characters_v5;

CREATE TABLE player_characters_v5 (
    id TEXT PRIMARY KEY CHECK (trim(id) <> ''),
    player_id TEXT NOT NULL CHECK (trim(player_id) <> ''),
    stat_profile_id TEXT NOT NULL CHECK (trim(stat_profile_id) <> ''),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    availability_status TEXT NOT NULL DEFAULT 'active' CHECK (availability_status IN ('active', 'inactive')),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    deleted_at DATETIME NULL,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id)
);

INSERT INTO player_characters_v5 (
    id, player_id, stat_profile_id, name, active, availability_status,
    created_at, updated_at, deleted_at
)
SELECT
    id,
    player_id,
    stat_profile_id,
    name,
    active,
    availability_status,
    created_at,
    updated_at,
    deleted_at
FROM player_characters;

DROP TABLE player_characters;
ALTER TABLE player_characters_v5 RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_player_active_availability
ON player_characters(player_id, active, availability_status, name);

CREATE VIEW player_character_resistance_global AS
SELECT
    pc.id AS player_character_id,
    sprg.damage_type_id,
    sprg.resistance,
    sprg.immune,
    sprg.created_at,
    sprg.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_global sprg
ON sprg.stat_profile_id = pc.stat_profile_id;

CREATE VIEW player_character_resistance_by_location AS
SELECT
    pc.id AS player_character_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = pc.stat_profile_id;

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_delete_stat_profile
AFTER DELETE ON player_characters
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
CREATE TRIGGER trg_stat_profiles_player_character_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM player_characters pc WHERE pc.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'player character level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_insert
INSTEAD OF INSERT ON player_character_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_update
INSTEAD OF UPDATE ON player_character_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id;

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_delete
INSTEAD OF DELETE ON player_character_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_insert
INSTEAD OF INSERT ON player_character_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_update
INSTEAD OF UPDATE ON player_character_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_delete
INSTEAD OF DELETE ON player_character_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_players_campaign_update_keeps_characters_consistent
BEFORE UPDATE OF campaign_id ON players
WHEN EXISTS (
    SELECT 1
    FROM player_characters pc
    JOIN combatants c ON c.player_character_id = pc.id
    JOIN encounters e ON e.id = c.encounter_id
    WHERE pc.player_id = NEW.id
      AND e.campaign_id <> NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player campaign update would mismatch linked combatants');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_player_update_keeps_combatants_consistent
BEFORE UPDATE OF player_id ON player_characters
WHEN EXISTS (
    SELECT 1
    FROM combatants c
    JOIN encounters e ON e.id = c.encounter_id
    JOIN players p ON p.id = NEW.player_id
    WHERE c.player_character_id = NEW.id
      AND p.campaign_id <> e.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'player character player update would mismatch linked combatants');
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
    JOIN players p ON p.id = pc.player_id
    WHERE e.id = NEW.encounter_id
      AND p.campaign_id = e.campaign_id
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
    JOIN players p ON p.id = pc.player_id
    WHERE e.id = NEW.encounter_id
      AND p.campaign_id = e.campaign_id
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
    JOIN players p ON p.id = pc.player_id
    WHERE c.encounter_id = NEW.id
      AND p.campaign_id <> NEW.campaign_id
)
BEGIN
    SELECT RAISE(ABORT, 'encounter campaign update would mismatch linked combatants');
END;
-- +goose StatementEnd

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

DROP TRIGGER IF EXISTS trg_encounters_campaign_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_update;
DROP TRIGGER IF EXISTS trg_combatants_player_character_campaign_insert;
DROP TRIGGER IF EXISTS trg_player_characters_player_update_keeps_combatants_consistent;
DROP TRIGGER IF EXISTS trg_players_campaign_update_keeps_characters_consistent;
DROP TRIGGER IF EXISTS trg_stat_profiles_player_character_level_update;
DROP TRIGGER IF EXISTS trg_player_characters_require_level;
DROP TRIGGER IF EXISTS trg_player_characters_delete_stat_profile;
DROP VIEW IF EXISTS player_character_resistance_by_location;
DROP VIEW IF EXISTS player_character_resistance_global;

DROP TABLE IF EXISTS player_characters_v5_down;

CREATE TABLE player_characters_v5_down (
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

INSERT INTO player_characters_v5_down (
    id, player_id, campaign_id, stat_profile_id, name, active, availability_status,
    created_at, updated_at, deleted_at
)
SELECT
    pc.id,
    pc.player_id,
    p.campaign_id,
    pc.stat_profile_id,
    pc.name,
    pc.active,
    pc.availability_status,
    pc.created_at,
    pc.updated_at,
    pc.deleted_at
FROM player_characters pc
JOIN players p ON p.id = pc.player_id;

DROP TABLE player_characters;
ALTER TABLE player_characters_v5_down RENAME TO player_characters;

CREATE UNIQUE INDEX IF NOT EXISTS idx_player_characters_one_active
ON player_characters(player_id)
WHERE active = 1;

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_active
ON player_characters(campaign_id, active, name);

CREATE INDEX IF NOT EXISTS idx_player_characters_campaign_availability
ON player_characters(campaign_id, active, availability_status, name);

CREATE VIEW player_character_resistance_global AS
SELECT
    pc.id AS player_character_id,
    sprg.damage_type_id,
    sprg.resistance,
    sprg.immune,
    sprg.created_at,
    sprg.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_global sprg
ON sprg.stat_profile_id = pc.stat_profile_id;

CREATE VIEW player_character_resistance_by_location AS
SELECT
    pc.id AS player_character_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = pc.stat_profile_id;

-- +goose StatementBegin
CREATE TRIGGER trg_player_characters_delete_stat_profile
AFTER DELETE ON player_characters
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
CREATE TRIGGER trg_stat_profiles_player_character_level_update
BEFORE UPDATE OF level ON stat_profiles
WHEN NEW.level < 1
  AND EXISTS (SELECT 1 FROM player_characters pc WHERE pc.stat_profile_id = NEW.id)
BEGIN
    SELECT RAISE(ABORT, 'player character level must be at least 1');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_insert
INSTEAD OF INSERT ON player_character_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_update
INSTEAD OF UPDATE ON player_character_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id;

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_delete
INSTEAD OF DELETE ON player_character_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_insert
INSTEAD OF INSERT ON player_character_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_update
INSTEAD OF UPDATE ON player_character_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_delete
INSTEAD OF DELETE ON player_character_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
END;
-- +goose StatementEnd

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
