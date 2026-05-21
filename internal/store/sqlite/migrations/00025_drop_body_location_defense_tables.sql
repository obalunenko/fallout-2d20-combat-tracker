-- +goose Up
DROP TABLE IF EXISTS player_character_defense_by_location;
DROP TABLE IF EXISTS combatant_defense_by_location;

-- +goose Down
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
