-- +goose Up
ALTER TABLE combatants ADD COLUMN max_hp INTEGER NOT NULL DEFAULT 1;
UPDATE combatants
SET max_hp = CASE
    WHEN hp > 0 THEN hp
    ELSE 1
END
WHERE max_hp <= 1;

ALTER TABLE player_characters ADD COLUMN max_hp INTEGER NOT NULL DEFAULT 1;
UPDATE player_characters
SET max_hp = CASE
    WHEN hp > 0 THEN hp
    ELSE 1
END
WHERE max_hp <= 1;

-- +goose Down
ALTER TABLE player_characters DROP COLUMN max_hp;
ALTER TABLE combatants DROP COLUMN max_hp;
