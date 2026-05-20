-- +goose Up
CREATE TABLE IF NOT EXISTS body_locations (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

INSERT OR IGNORE INTO body_locations (id, code) VALUES
    (1, 'head'),
    (2, 'torso'),
    (3, 'left_arm'),
    (4, 'right_arm'),
    (5, 'left_leg'),
    (6, 'right_leg');

CREATE TABLE IF NOT EXISTS combatant_defense_by_location (
    combatant_id TEXT NOT NULL,
    body_location_id INTEGER NOT NULL,
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (combatant_id, body_location_id),
    FOREIGN KEY (combatant_id) REFERENCES combatants(id) ON DELETE CASCADE,
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

CREATE TABLE IF NOT EXISTS player_character_defense_by_location (
    player_character_id TEXT NOT NULL,
    body_location_id INTEGER NOT NULL,
    defense INTEGER NOT NULL DEFAULT 0 CHECK (defense >= 0),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, body_location_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (body_location_id) REFERENCES body_locations(id)
);

INSERT INTO combatant_defense_by_location (combatant_id, body_location_id, defense, created_at, updated_at)
SELECT
    c.id,
    bl.id,
    CASE bl.code
        WHEN 'head' THEN c.defense_head
        WHEN 'torso' THEN c.defense_torso
        WHEN 'left_arm' THEN c.defense_left_arm
        WHEN 'right_arm' THEN c.defense_right_arm
        WHEN 'left_leg' THEN c.defense_left_leg
        WHEN 'right_leg' THEN c.defense_right_leg
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM combatants c
JOIN body_locations bl
ON 1 = 1
ON CONFLICT (combatant_id, body_location_id) DO NOTHING;

INSERT INTO player_character_defense_by_location (player_character_id, body_location_id, defense, created_at, updated_at)
SELECT
    pc.id,
    bl.id,
    CASE bl.code
        WHEN 'head' THEN pc.defense_head
        WHEN 'torso' THEN pc.defense_torso
        WHEN 'left_arm' THEN pc.defense_left_arm
        WHEN 'right_arm' THEN pc.defense_right_arm
        WHEN 'left_leg' THEN pc.defense_left_leg
        WHEN 'right_leg' THEN pc.defense_right_leg
        ELSE 0
    END,
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
FROM player_characters pc
JOIN body_locations bl
ON 1 = 1
ON CONFLICT (player_character_id, body_location_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS player_character_defense_by_location;
DROP TABLE IF EXISTS combatant_defense_by_location;
DROP TABLE IF EXISTS body_locations;
