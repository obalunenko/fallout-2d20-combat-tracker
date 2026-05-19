-- +goose Up
ALTER TABLE combatants ADD COLUMN hp INTEGER NOT NULL DEFAULT 1;
ALTER TABLE combatants ADD COLUMN defense INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE combatants DROP COLUMN damage_resistance;
ALTER TABLE combatants DROP COLUMN defense;
ALTER TABLE combatants DROP COLUMN hp;
