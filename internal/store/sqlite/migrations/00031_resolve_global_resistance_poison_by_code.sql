-- +goose Up
DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_update;

DROP TABLE IF EXISTS combatant_resistance_global_v3;
DROP TABLE IF EXISTS player_character_resistance_global_v3;
DROP TABLE IF EXISTS monster_template_resistance_global_v3;

CREATE TABLE combatant_resistance_global_v3 (
    combatant_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, damage_type_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO combatant_resistance_global_v3 (
    combatant_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    crg.combatant_id,
    crg.damage_type_id,
    CASE WHEN dt.code = 'poison' THEN crg.resistance ELSE 0 END,
    crg.immune,
    COALESCE(crg.created_at, crg.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(crg.updated_at, crg.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
FROM combatant_resistance_global crg
JOIN damage_types dt ON dt.id = crg.damage_type_id;

DROP TABLE combatant_resistance_global;
ALTER TABLE combatant_resistance_global_v3 RENAME TO combatant_resistance_global;

CREATE TABLE player_character_resistance_global_v3 (
    player_character_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, damage_type_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO player_character_resistance_global_v3 (
    player_character_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    pcrg.player_character_id,
    pcrg.damage_type_id,
    CASE WHEN dt.code = 'poison' THEN pcrg.resistance ELSE 0 END,
    pcrg.immune,
    COALESCE(pcrg.created_at, pcrg.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(pcrg.updated_at, pcrg.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
FROM player_character_resistance_global pcrg
JOIN damage_types dt ON dt.id = pcrg.damage_type_id;

DROP TABLE player_character_resistance_global;
ALTER TABLE player_character_resistance_global_v3 RENAME TO player_character_resistance_global;

CREATE TABLE monster_template_resistance_global_v3 (
    monster_template_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (monster_template_id, damage_type_id),
    FOREIGN KEY (monster_template_id) REFERENCES monster_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO monster_template_resistance_global_v3 (
    monster_template_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    mtg.monster_template_id,
    mtg.damage_type_id,
    CASE WHEN dt.code = 'poison' THEN mtg.resistance ELSE 0 END,
    mtg.immune,
    COALESCE(mtg.created_at, mtg.updated_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    COALESCE(mtg.updated_at, mtg.created_at, STRFTIME('%Y-%m-%d %H:%M:%f', 'now'))
FROM monster_template_resistance_global mtg
JOIN damage_types dt ON dt.id = mtg.damage_type_id;

DROP TABLE monster_template_resistance_global;
ALTER TABLE monster_template_resistance_global_v3 RENAME TO monster_template_resistance_global;

-- +goose StatementBegin
CREATE TRIGGER trg_combatant_resistance_global_poison_only_insert
BEFORE INSERT ON combatant_resistance_global
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
CREATE TRIGGER trg_combatant_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON combatant_resistance_global
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
CREATE TRIGGER trg_player_character_resistance_global_poison_only_insert
BEFORE INSERT ON player_character_resistance_global
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
CREATE TRIGGER trg_player_character_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON player_character_resistance_global
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
CREATE TRIGGER trg_monster_template_resistance_global_poison_only_insert
BEFORE INSERT ON monster_template_resistance_global
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
CREATE TRIGGER trg_monster_template_resistance_global_poison_only_update
BEFORE UPDATE OF damage_type_id, resistance ON monster_template_resistance_global
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

-- +goose Down
DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_combatant_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_player_character_resistance_global_poison_only_update;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_insert;
DROP TRIGGER IF EXISTS trg_monster_template_resistance_global_poison_only_update;
