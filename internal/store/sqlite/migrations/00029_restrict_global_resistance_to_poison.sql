-- +goose Up
CREATE TABLE combatant_resistance_global_v2 (
    combatant_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, damage_type_id),
    CHECK (damage_type_id = 4 OR resistance = 0),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO combatant_resistance_global_v2 (
    combatant_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    combatant_id,
    damage_type_id,
    CASE WHEN damage_type_id = 4 THEN resistance ELSE 0 END,
    immune,
    created_at,
    updated_at
FROM combatant_resistance_global;

DROP TABLE combatant_resistance_global;
ALTER TABLE combatant_resistance_global_v2 RENAME TO combatant_resistance_global;

CREATE TABLE player_character_resistance_global_v2 (
    player_character_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, damage_type_id),
    CHECK (damage_type_id = 4 OR resistance = 0),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO player_character_resistance_global_v2 (
    player_character_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    player_character_id,
    damage_type_id,
    CASE WHEN damage_type_id = 4 THEN resistance ELSE 0 END,
    immune,
    created_at,
    updated_at
FROM player_character_resistance_global;

DROP TABLE player_character_resistance_global;
ALTER TABLE player_character_resistance_global_v2 RENAME TO player_character_resistance_global;

CREATE TABLE monster_template_resistance_global_v2 (
    monster_template_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    immune INTEGER NOT NULL DEFAULT 0 CHECK (immune IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (monster_template_id, damage_type_id),
    CHECK (damage_type_id = 4 OR resistance = 0),
    FOREIGN KEY (monster_template_id) REFERENCES monster_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id)
);

INSERT INTO monster_template_resistance_global_v2 (
    monster_template_id,
    damage_type_id,
    resistance,
    immune,
    created_at,
    updated_at
)
SELECT
    monster_template_id,
    damage_type_id,
    CASE WHEN damage_type_id = 4 THEN resistance ELSE 0 END,
    immune,
    created_at,
    updated_at
FROM monster_template_resistance_global;

DROP TABLE monster_template_resistance_global;
ALTER TABLE monster_template_resistance_global_v2 RENAME TO monster_template_resistance_global;

-- +goose Down
CREATE TABLE combatant_resistance_global_v1 (
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

INSERT INTO combatant_resistance_global_v1
SELECT * FROM combatant_resistance_global;

DROP TABLE combatant_resistance_global;
ALTER TABLE combatant_resistance_global_v1 RENAME TO combatant_resistance_global;

CREATE TABLE player_character_resistance_global_v1 (
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

INSERT INTO player_character_resistance_global_v1
SELECT * FROM player_character_resistance_global;

DROP TABLE player_character_resistance_global;
ALTER TABLE player_character_resistance_global_v1 RENAME TO player_character_resistance_global;

CREATE TABLE monster_template_resistance_global_v1 (
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

INSERT INTO monster_template_resistance_global_v1
SELECT * FROM monster_template_resistance_global;

DROP TABLE monster_template_resistance_global;
ALTER TABLE monster_template_resistance_global_v1 RENAME TO monster_template_resistance_global;
