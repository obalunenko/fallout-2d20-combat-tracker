-- +goose Up
CREATE TABLE IF NOT EXISTS damage_types (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO damage_types (id, code) VALUES
    (1, 'physical'),
    (2, 'energy'),
    (3, 'radiation'),
    (4, 'poison');

CREATE TABLE IF NOT EXISTS combatant_resistance_global (
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

CREATE TABLE IF NOT EXISTS combatant_resistance_by_location (
    combatant_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, damage_type_id, body_location_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE TABLE IF NOT EXISTS player_character_resistance_global (
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

CREATE TABLE IF NOT EXISTS player_character_resistance_by_location (
    player_character_id TEXT NOT NULL,
    damage_type_id INTEGER NOT NULL,
    body_location_id INTEGER NOT NULL,
    resistance INTEGER NOT NULL DEFAULT 0 CHECK (resistance >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, damage_type_id, body_location_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (damage_type_id) REFERENCES damage_types(id),
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune, created_at, updated_at)
SELECT
    c.id,
    dt.id,
    CASE dt.code
        WHEN 'physical' THEN c.damage_resistance_physical
        WHEN 'energy' THEN c.damage_resistance_energy
        WHEN 'radiation' THEN c.damage_resistance_radiation
        WHEN 'poison' THEN c.damage_resistance_poison
        ELSE 0
    END,
    CASE dt.code
        WHEN 'physical' THEN c.damage_resistance_physical_immune
        WHEN 'energy' THEN c.damage_resistance_energy_immune
        WHEN 'radiation' THEN c.damage_resistance_radiation_immune
        WHEN 'poison' THEN c.damage_resistance_poison_immune
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM combatants c
JOIN damage_types dt
ON 1 = 1
ON CONFLICT (combatant_id, damage_type_id) DO NOTHING;

INSERT INTO player_character_resistance_global (player_character_id, damage_type_id, resistance, immune, created_at, updated_at)
SELECT
    pc.id,
    dt.id,
    CASE dt.code
        WHEN 'physical' THEN pc.damage_resistance_physical
        WHEN 'energy' THEN pc.damage_resistance_energy
        WHEN 'radiation' THEN pc.damage_resistance_radiation
        WHEN 'poison' THEN pc.damage_resistance_poison
        ELSE 0
    END,
    CASE dt.code
        WHEN 'physical' THEN pc.damage_resistance_physical_immune
        WHEN 'energy' THEN pc.damage_resistance_energy_immune
        WHEN 'radiation' THEN pc.damage_resistance_radiation_immune
        WHEN 'poison' THEN pc.damage_resistance_poison_immune
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM player_characters pc
JOIN damage_types dt
ON 1 = 1
ON CONFLICT (player_character_id, damage_type_id) DO NOTHING;

INSERT INTO combatant_resistance_by_location (combatant_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
SELECT
    c.id,
    dt.id,
    bl.id,
    CASE
        WHEN dt.code = 'physical' AND bl.code = 'head' THEN c.damage_resistance_physical_head
        WHEN dt.code = 'physical' AND bl.code = 'torso' THEN c.damage_resistance_physical_torso
        WHEN dt.code = 'physical' AND bl.code = 'left_arm' THEN c.damage_resistance_physical_left_arm
        WHEN dt.code = 'physical' AND bl.code = 'right_arm' THEN c.damage_resistance_physical_right_arm
        WHEN dt.code = 'physical' AND bl.code = 'left_leg' THEN c.damage_resistance_physical_left_leg
        WHEN dt.code = 'physical' AND bl.code = 'right_leg' THEN c.damage_resistance_physical_right_leg
        WHEN dt.code = 'energy' AND bl.code = 'head' THEN c.damage_resistance_energy_head
        WHEN dt.code = 'energy' AND bl.code = 'torso' THEN c.damage_resistance_energy_torso
        WHEN dt.code = 'energy' AND bl.code = 'left_arm' THEN c.damage_resistance_energy_left_arm
        WHEN dt.code = 'energy' AND bl.code = 'right_arm' THEN c.damage_resistance_energy_right_arm
        WHEN dt.code = 'energy' AND bl.code = 'left_leg' THEN c.damage_resistance_energy_left_leg
        WHEN dt.code = 'energy' AND bl.code = 'right_leg' THEN c.damage_resistance_energy_right_leg
        WHEN dt.code = 'radiation' AND bl.code = 'head' THEN c.damage_resistance_radiation_head
        WHEN dt.code = 'radiation' AND bl.code = 'torso' THEN c.damage_resistance_radiation_torso
        WHEN dt.code = 'radiation' AND bl.code = 'left_arm' THEN c.damage_resistance_radiation_left_arm
        WHEN dt.code = 'radiation' AND bl.code = 'right_arm' THEN c.damage_resistance_radiation_right_arm
        WHEN dt.code = 'radiation' AND bl.code = 'left_leg' THEN c.damage_resistance_radiation_left_leg
        WHEN dt.code = 'radiation' AND bl.code = 'right_leg' THEN c.damage_resistance_radiation_right_leg
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM combatants c
JOIN damage_types dt
ON dt.code IN ('physical', 'energy', 'radiation')
JOIN body_locations bl
ON 1 = 1
ON CONFLICT (combatant_id, damage_type_id, body_location_id) DO NOTHING;

INSERT INTO player_character_resistance_by_location (player_character_id, damage_type_id, body_location_id, resistance, created_at, updated_at)
SELECT
    pc.id,
    dt.id,
    bl.id,
    CASE
        WHEN dt.code = 'physical' AND bl.code = 'head' THEN pc.damage_resistance_physical_head
        WHEN dt.code = 'physical' AND bl.code = 'torso' THEN pc.damage_resistance_physical_torso
        WHEN dt.code = 'physical' AND bl.code = 'left_arm' THEN pc.damage_resistance_physical_left_arm
        WHEN dt.code = 'physical' AND bl.code = 'right_arm' THEN pc.damage_resistance_physical_right_arm
        WHEN dt.code = 'physical' AND bl.code = 'left_leg' THEN pc.damage_resistance_physical_left_leg
        WHEN dt.code = 'physical' AND bl.code = 'right_leg' THEN pc.damage_resistance_physical_right_leg
        WHEN dt.code = 'energy' AND bl.code = 'head' THEN pc.damage_resistance_energy_head
        WHEN dt.code = 'energy' AND bl.code = 'torso' THEN pc.damage_resistance_energy_torso
        WHEN dt.code = 'energy' AND bl.code = 'left_arm' THEN pc.damage_resistance_energy_left_arm
        WHEN dt.code = 'energy' AND bl.code = 'right_arm' THEN pc.damage_resistance_energy_right_arm
        WHEN dt.code = 'energy' AND bl.code = 'left_leg' THEN pc.damage_resistance_energy_left_leg
        WHEN dt.code = 'energy' AND bl.code = 'right_leg' THEN pc.damage_resistance_energy_right_leg
        WHEN dt.code = 'radiation' AND bl.code = 'head' THEN pc.damage_resistance_radiation_head
        WHEN dt.code = 'radiation' AND bl.code = 'torso' THEN pc.damage_resistance_radiation_torso
        WHEN dt.code = 'radiation' AND bl.code = 'left_arm' THEN pc.damage_resistance_radiation_left_arm
        WHEN dt.code = 'radiation' AND bl.code = 'right_arm' THEN pc.damage_resistance_radiation_right_arm
        WHEN dt.code = 'radiation' AND bl.code = 'left_leg' THEN pc.damage_resistance_radiation_left_leg
        WHEN dt.code = 'radiation' AND bl.code = 'right_leg' THEN pc.damage_resistance_radiation_right_leg
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM player_characters pc
JOIN damage_types dt
ON dt.code IN ('physical', 'energy', 'radiation')
JOIN body_locations bl
ON 1 = 1
ON CONFLICT (player_character_id, damage_type_id, body_location_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS player_character_resistance_by_location;
DROP TABLE IF EXISTS player_character_resistance_global;
DROP TABLE IF EXISTS combatant_resistance_by_location;
DROP TABLE IF EXISTS combatant_resistance_global;
DROP TABLE IF EXISTS damage_types;
