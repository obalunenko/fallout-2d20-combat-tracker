-- +goose Up
ALTER TABLE combatants ADD COLUMN defense_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN defense_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN defense_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN defense_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN defense_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN defense_right_leg INTEGER NOT NULL DEFAULT 0;

UPDATE combatants
SET defense_head = defense,
    defense_torso = defense,
    defense_left_arm = defense,
    defense_right_arm = defense,
    defense_left_leg = defense,
    defense_right_leg = defense;

ALTER TABLE player_characters ADD COLUMN defense_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN defense_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN defense_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN defense_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN defense_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN defense_right_leg INTEGER NOT NULL DEFAULT 0;

UPDATE player_characters
SET defense_head = defense,
    defense_torso = defense,
    defense_left_arm = defense,
    defense_right_arm = defense,
    defense_left_leg = defense,
    defense_right_leg = defense;

-- +goose Down
ALTER TABLE player_characters DROP COLUMN defense_right_leg;
ALTER TABLE player_characters DROP COLUMN defense_left_leg;
ALTER TABLE player_characters DROP COLUMN defense_right_arm;
ALTER TABLE player_characters DROP COLUMN defense_left_arm;
ALTER TABLE player_characters DROP COLUMN defense_torso;
ALTER TABLE player_characters DROP COLUMN defense_head;

ALTER TABLE combatants DROP COLUMN defense_right_leg;
ALTER TABLE combatants DROP COLUMN defense_left_leg;
ALTER TABLE combatants DROP COLUMN defense_right_arm;
ALTER TABLE combatants DROP COLUMN defense_left_arm;
ALTER TABLE combatants DROP COLUMN defense_torso;
ALTER TABLE combatants DROP COLUMN defense_head;
