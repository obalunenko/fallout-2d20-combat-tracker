-- +goose Up
ALTER TABLE combatants DROP COLUMN damage_resistance;

ALTER TABLE campaigns DROP COLUMN created_at;

ALTER TABLE players DROP COLUMN created_at;
ALTER TABLE players DROP COLUMN updated_at;

ALTER TABLE player_characters DROP COLUMN created_at;
ALTER TABLE player_characters DROP COLUMN updated_at;

-- +goose Down
ALTER TABLE player_characters ADD COLUMN updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
ALTER TABLE player_characters ADD COLUMN created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE players ADD COLUMN updated_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));
ALTER TABLE players ADD COLUMN created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE campaigns ADD COLUMN created_at DATETIME NOT NULL DEFAULT (STRFTIME('%Y-%m-%d %H:%M:%f', 'now'));

ALTER TABLE combatants ADD COLUMN damage_resistance INTEGER NOT NULL DEFAULT 0;
