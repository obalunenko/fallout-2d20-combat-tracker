-- +goose Up
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

-- +goose Down
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_monster_template_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_player_character_resistance_global_insert;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_by_location_insert;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_delete;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_update;
DROP TRIGGER IF EXISTS trg_compat_combatant_resistance_global_insert;

DROP VIEW IF EXISTS monster_template_resistance_by_location;
DROP VIEW IF EXISTS monster_template_resistance_global;
DROP VIEW IF EXISTS player_character_resistance_by_location;
DROP VIEW IF EXISTS player_character_resistance_global;
DROP VIEW IF EXISTS combatant_resistance_by_location;
DROP VIEW IF EXISTS combatant_resistance_global;
