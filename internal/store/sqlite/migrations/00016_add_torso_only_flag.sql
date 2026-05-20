-- +goose Up
ALTER TABLE combatants ADD COLUMN torso_only INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN torso_only INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE player_characters DROP COLUMN torso_only;
ALTER TABLE combatants DROP COLUMN torso_only;
