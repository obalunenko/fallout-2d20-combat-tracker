-- +goose Up
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_poison_only_update;

DROP VIEW IF EXISTS combatant_resistance_global;
DROP VIEW IF EXISTS combatant_resistance_by_location;
DROP VIEW IF EXISTS player_character_resistance_global;
DROP VIEW IF EXISTS player_character_resistance_by_location;
DROP VIEW IF EXISTS monster_template_resistance_global;
DROP VIEW IF EXISTS monster_template_resistance_by_location;

CREATE TABLE stat_profile_resistance_global_legacy AS
SELECT stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
FROM stat_profile_resistance_global;

CREATE TABLE stat_profile_resistance_by_location_legacy AS
SELECT stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
FROM stat_profile_resistance_by_location;

DROP TABLE stat_profile_resistance_global;
DROP TABLE stat_profile_resistance_by_location;

CREATE TABLE body_locations_v2 (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN ('global', 'head', 'torso', 'left_arm', 'right_arm', 'left_leg', 'right_leg'))
);

INSERT INTO body_locations_v2 (id, code) VALUES
    (0, 'global'),
    (1, 'head'),
    (2, 'torso'),
    (3, 'left_arm'),
    (4, 'right_arm'),
    (5, 'left_leg'),
    (6, 'right_leg');

DROP TABLE body_locations;
ALTER TABLE body_locations_v2 RENAME TO body_locations;

CREATE TABLE stat_profile_resistance_by_location (
    stat_profile_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (stat_profile_id, damage_type_id, body_location_id),
    FOREIGN KEY (stat_profile_id) REFERENCES stat_profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

INSERT INTO stat_profile_resistance_by_location (
    stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
)
SELECT
    l.stat_profile_id,
    l.damage_type_id,
    l.body_location_id,
    CASE WHEN dt.code = 'poison' THEN 0 ELSE l.resistance END,
    0,
    l.created_at,
    l.updated_at
FROM stat_profile_resistance_by_location_legacy l
JOIN damage_types dt ON dt.id = l.damage_type_id
WHERE NOT EXISTS (
    SELECT 1
    FROM stat_profile_resistance_global_legacy g
    JOIN damage_types gdt ON gdt.id = g.damage_type_id
    WHERE g.stat_profile_id = l.stat_profile_id
      AND g.damage_type_id = l.damage_type_id
      AND (
          (gdt.code = 'poison' AND (g.resistance <> 0 OR g.immune <> 0))
          OR (gdt.code <> 'poison' AND g.immune <> 0)
      )
);

INSERT INTO stat_profile_resistance_by_location (
    stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
)
SELECT
    g.stat_profile_id,
    g.damage_type_id,
    0,
    CASE WHEN dt.code = 'poison' THEN g.resistance ELSE 0 END,
    g.immune,
    g.created_at,
    g.updated_at
FROM stat_profile_resistance_global_legacy g
JOIN damage_types dt ON dt.id = g.damage_type_id
WHERE (dt.code = 'poison' AND (g.resistance <> 0 OR g.immune <> 0))
   OR (dt.code <> 'poison' AND g.immune <> 0);

DROP TABLE stat_profile_resistance_global_legacy;
DROP TABLE stat_profile_resistance_by_location_legacy;

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_global_insert
BEFORE INSERT ON stat_profile_resistance_by_location
WHEN NEW.body_location_id = 0
  AND NEW.resistance <> 0
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
CREATE TRIGGER trg_stat_profile_resistance_location_global_update
BEFORE UPDATE OF damage_type_id, body_location_id, resistance ON stat_profile_resistance_by_location
WHEN NEW.body_location_id = 0
  AND NEW.resistance <> 0
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
CREATE TRIGGER trg_stat_profile_resistance_location_scoped_insert
BEFORE INSERT ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND NEW.immune <> 0
BEGIN
    SELECT RAISE(ABORT, 'location resistance cannot store immunity');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_scoped_update
BEFORE UPDATE OF body_location_id, immune ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND NEW.immune <> 0
BEGIN
    SELECT RAISE(ABORT, 'location resistance cannot store immunity');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_poison_insert
BEFORE INSERT ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND NEW.resistance <> 0
  AND EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'poison resistance must be global');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_poison_update
BEFORE UPDATE OF damage_type_id, body_location_id, resistance ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND NEW.resistance <> 0
  AND EXISTS (
    SELECT 1 FROM damage_types dt
    WHERE dt.id = NEW.damage_type_id
      AND dt.code = 'poison'
  )
BEGIN
    SELECT RAISE(ABORT, 'poison resistance must be global');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_global_excludes_location_insert
BEFORE INSERT ON stat_profile_resistance_by_location
WHEN NEW.body_location_id = 0
  AND EXISTS (
    SELECT 1
    FROM stat_profile_resistance_by_location spr
    WHERE spr.stat_profile_id = NEW.stat_profile_id
      AND spr.damage_type_id = NEW.damage_type_id
      AND spr.body_location_id <> 0
  )
BEGIN
    SELECT RAISE(ABORT, 'global resistance conflicts with location resistance');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_global_excludes_location_update
BEFORE UPDATE OF stat_profile_id, damage_type_id, body_location_id ON stat_profile_resistance_by_location
WHEN NEW.body_location_id = 0
  AND EXISTS (
    SELECT 1
    FROM stat_profile_resistance_by_location spr
    WHERE spr.stat_profile_id = NEW.stat_profile_id
      AND spr.damage_type_id = NEW.damage_type_id
      AND spr.body_location_id <> 0
  )
BEGIN
    SELECT RAISE(ABORT, 'global resistance conflicts with location resistance');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_excludes_global_insert
BEFORE INSERT ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND EXISTS (
    SELECT 1
    FROM stat_profile_resistance_by_location spr
    WHERE spr.stat_profile_id = NEW.stat_profile_id
      AND spr.damage_type_id = NEW.damage_type_id
      AND spr.body_location_id = 0
  )
BEGIN
    SELECT RAISE(ABORT, 'location resistance conflicts with global resistance');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_stat_profile_resistance_location_excludes_global_update
BEFORE UPDATE OF stat_profile_id, damage_type_id, body_location_id ON stat_profile_resistance_by_location
WHEN NEW.body_location_id <> 0
  AND EXISTS (
    SELECT 1
    FROM stat_profile_resistance_by_location spr
    WHERE spr.stat_profile_id = NEW.stat_profile_id
      AND spr.damage_type_id = NEW.damage_type_id
      AND spr.body_location_id = 0
  )
BEGIN
    SELECT RAISE(ABORT, 'location resistance conflicts with global resistance');
END;
-- +goose StatementEnd

CREATE VIEW combatant_resistance_global AS
SELECT
    c.id AS combatant_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM combatants c
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = c.stat_profile_id
WHERE spr.body_location_id = 0;

CREATE VIEW combatant_resistance_by_location AS
SELECT
    c.id AS combatant_id,
    spr.body_location_id,
    spr.damage_type_id,
    spr.resistance,
    spr.created_at,
    spr.updated_at
FROM combatants c
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = c.stat_profile_id
WHERE spr.body_location_id <> 0;

CREATE VIEW monster_template_resistance_global AS
SELECT
    mt.id AS monster_template_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = mt.stat_profile_id
WHERE spr.body_location_id = 0;

CREATE VIEW monster_template_resistance_by_location AS
SELECT
    mt.id AS monster_template_id,
    spr.body_location_id,
    spr.damage_type_id,
    spr.resistance,
    spr.created_at,
    spr.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = mt.stat_profile_id
WHERE spr.body_location_id <> 0;

CREATE VIEW player_character_resistance_global AS
SELECT
    pc.id AS player_character_id,
    spr.damage_type_id,
    spr.resistance,
    spr.immune,
    spr.created_at,
    spr.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = pc.stat_profile_id
WHERE spr.body_location_id = 0;

CREATE VIEW player_character_resistance_by_location AS
SELECT
    pc.id AS player_character_id,
    spr.body_location_id,
    spr.damage_type_id,
    spr.resistance,
    spr.created_at,
    spr.updated_at
FROM player_characters pc
JOIN stat_profile_resistance_by_location spr
ON spr.stat_profile_id = pc.stat_profile_id
WHERE spr.body_location_id <> 0;

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_global_insert
INSTEAD OF INSERT ON combatant_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_global_update
INSTEAD OF UPDATE ON combatant_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_global_delete
INSTEAD OF DELETE ON combatant_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_insert
INSTEAD OF INSERT ON combatant_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_update
INSTEAD OF UPDATE ON combatant_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_delete
INSTEAD OF DELETE ON combatant_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_insert
INSTEAD OF INSERT ON monster_template_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_update
INSTEAD OF UPDATE ON monster_template_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_delete
INSTEAD OF DELETE ON monster_template_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_insert
INSTEAD OF INSERT ON monster_template_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_update
INSTEAD OF UPDATE ON monster_template_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_delete
INSTEAD OF DELETE ON monster_template_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_insert
INSTEAD OF INSERT ON player_character_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
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

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
    )
    SELECT
        pc.stat_profile_id,
        NEW.damage_type_id,
        0,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM player_characters pc
    WHERE pc.id = NEW.player_character_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_global_delete
INSTEAD OF DELETE ON player_character_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT pc.stat_profile_id FROM player_characters pc WHERE pc.id = OLD.player_character_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = 0;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_player_character_resistance_by_location_insert
INSTEAD OF INSERT ON player_character_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown player_character_id')
    WHERE NOT EXISTS (SELECT 1 FROM player_characters pc WHERE pc.id = NEW.player_character_id);
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

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
    SELECT RAISE(ABORT, 'global resistance must use global view')
    WHERE NEW.body_location_id = 0;

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

-- +goose Down
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_global_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_global_update;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_scoped_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_scoped_update;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_poison_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_poison_update;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_excludes_location_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_global_excludes_location_update;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_excludes_global_insert;
DROP TRIGGER IF EXISTS trg_stat_profile_resistance_location_excludes_global_update;

DROP VIEW IF EXISTS combatant_resistance_global;
DROP VIEW IF EXISTS combatant_resistance_by_location;
DROP VIEW IF EXISTS player_character_resistance_global;
DROP VIEW IF EXISTS player_character_resistance_by_location;
DROP VIEW IF EXISTS monster_template_resistance_global;
DROP VIEW IF EXISTS monster_template_resistance_by_location;

CREATE TABLE stat_profile_resistance_unified_legacy AS
SELECT stat_profile_id, damage_type_id, body_location_id, resistance, immune, created_at, updated_at
FROM stat_profile_resistance_by_location;

CREATE TABLE stat_profile_resistance_global (
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

INSERT INTO stat_profile_resistance_global (
    stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
)
SELECT
    u.stat_profile_id,
    u.damage_type_id,
    CASE WHEN dt.code = 'poison' THEN u.resistance ELSE 0 END,
    u.immune,
    u.created_at,
    u.updated_at
FROM stat_profile_resistance_unified_legacy u
JOIN damage_types dt ON dt.id = u.damage_type_id
WHERE u.body_location_id = 0;

DROP TABLE stat_profile_resistance_by_location;

CREATE TABLE body_locations_v1 (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN ('head', 'torso', 'left_arm', 'right_arm', 'left_leg', 'right_leg'))
);

INSERT INTO body_locations_v1 (id, code) VALUES
    (1, 'head'),
    (2, 'torso'),
    (3, 'left_arm'),
    (4, 'right_arm'),
    (5, 'left_leg'),
    (6, 'right_leg');

DROP TABLE body_locations;
ALTER TABLE body_locations_v1 RENAME TO body_locations;

CREATE TABLE stat_profile_resistance_by_location (
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

INSERT INTO stat_profile_resistance_by_location (
    stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
)
SELECT stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
FROM stat_profile_resistance_unified_legacy
WHERE body_location_id <> 0;

DROP TABLE stat_profile_resistance_unified_legacy;

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

CREATE VIEW combatant_resistance_global AS
SELECT
    c.id AS combatant_id,
    sprg.damage_type_id,
    sprg.resistance,
    sprg.immune,
    sprg.created_at,
    sprg.updated_at
FROM combatants c
JOIN stat_profile_resistance_global sprg
ON sprg.stat_profile_id = c.stat_profile_id;

CREATE VIEW combatant_resistance_by_location AS
SELECT
    c.id AS combatant_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM combatants c
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = c.stat_profile_id;

CREATE VIEW monster_template_resistance_global AS
SELECT
    mt.id AS monster_template_id,
    sprg.damage_type_id,
    sprg.resistance,
    sprg.immune,
    sprg.created_at,
    sprg.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_global sprg
ON sprg.stat_profile_id = mt.stat_profile_id;

CREATE VIEW monster_template_resistance_by_location AS
SELECT
    mt.id AS monster_template_id,
    sprl.damage_type_id,
    sprl.body_location_id,
    sprl.resistance,
    sprl.created_at,
    sprl.updated_at
FROM monster_templates mt
JOIN stat_profile_resistance_by_location sprl
ON sprl.stat_profile_id = mt.stat_profile_id;

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
CREATE TRIGGER trg_compat_combatant_resistance_global_insert
INSTEAD OF INSERT ON combatant_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_global_update
INSTEAD OF UPDATE ON combatant_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id;

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_global_delete
INSTEAD OF DELETE ON combatant_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_insert
INSTEAD OF INSERT ON combatant_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_update
INSTEAD OF UPDATE ON combatant_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown combatant_id')
    WHERE NOT EXISTS (SELECT 1 FROM combatants c WHERE c.id = NEW.combatant_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        c.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM combatants c
    WHERE c.id = NEW.combatant_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_combatant_resistance_by_location_delete
INSTEAD OF DELETE ON combatant_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT c.stat_profile_id FROM combatants c WHERE c.id = OLD.combatant_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_insert
INSTEAD OF INSERT ON monster_template_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_update
INSTEAD OF UPDATE ON monster_template_resistance_global
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id;

    INSERT INTO stat_profile_resistance_global (
        stat_profile_id, damage_type_id, resistance, immune, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.resistance,
        NEW.immune,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id) DO UPDATE SET
        resistance = excluded.resistance,
        immune = excluded.immune,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_global_delete
INSTEAD OF DELETE ON monster_template_resistance_global
BEGIN
    DELETE FROM stat_profile_resistance_global
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_insert
INSTEAD OF INSERT ON monster_template_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_update
INSTEAD OF UPDATE ON monster_template_resistance_by_location
BEGIN
    SELECT RAISE(ABORT, 'unknown monster_template_id')
    WHERE NOT EXISTS (SELECT 1 FROM monster_templates mt WHERE mt.id = NEW.monster_template_id);

    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;

    INSERT INTO stat_profile_resistance_by_location (
        stat_profile_id, damage_type_id, body_location_id, resistance, created_at, updated_at
    )
    SELECT
        mt.stat_profile_id,
        NEW.damage_type_id,
        NEW.body_location_id,
        NEW.resistance,
        COALESCE(NEW.created_at, OLD.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
        COALESCE(NEW.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
    FROM monster_templates mt
    WHERE mt.id = NEW.monster_template_id
    ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
        resistance = excluded.resistance,
        updated_at = excluded.updated_at;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_compat_monster_template_resistance_by_location_delete
INSTEAD OF DELETE ON monster_template_resistance_by_location
BEGIN
    DELETE FROM stat_profile_resistance_by_location
    WHERE stat_profile_id = (SELECT mt.stat_profile_id FROM monster_templates mt WHERE mt.id = OLD.monster_template_id)
      AND damage_type_id = OLD.damage_type_id
      AND body_location_id = OLD.body_location_id;
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
