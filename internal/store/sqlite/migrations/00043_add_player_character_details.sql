-- +goose Up
ALTER TABLE player_characters ADD COLUMN notes TEXT NOT NULL DEFAULT '';

CREATE TABLE special_attributes (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code IN (
        'strength', 'perception', 'endurance', 'charisma',
        'intelligence', 'agility', 'luck'
    ))
);

INSERT INTO special_attributes (id, code) VALUES
    (1, 'strength'),
    (2, 'perception'),
    (3, 'endurance'),
    (4, 'charisma'),
    (5, 'intelligence'),
    (6, 'agility'),
    (7, 'luck');

CREATE TABLE player_character_special_attributes (
    player_character_id TEXT NOT NULL,
    special_attribute_id INTEGER NOT NULL,
    value INTEGER NOT NULL DEFAULT 1 CHECK (value >= 1),
    created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
    PRIMARY KEY (player_character_id, special_attribute_id),
    FOREIGN KEY (player_character_id) REFERENCES player_characters(id) ON DELETE CASCADE,
    FOREIGN KEY (special_attribute_id) REFERENCES special_attributes(id)
);

INSERT INTO player_character_special_attributes (player_character_id, special_attribute_id, value)
SELECT pc.id, sa.id, 1
FROM player_characters pc
CROSS JOIN special_attributes sa;

CREATE INDEX idx_player_character_special_attribute
ON player_character_special_attributes(special_attribute_id, player_character_id);

-- +goose Down
DROP INDEX IF EXISTS idx_player_character_special_attribute;
DROP TABLE IF EXISTS player_character_special_attributes;
DROP TABLE IF EXISTS special_attributes;
ALTER TABLE player_characters DROP COLUMN notes;
