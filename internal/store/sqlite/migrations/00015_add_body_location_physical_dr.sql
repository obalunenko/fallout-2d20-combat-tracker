-- +goose Up
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_right_leg INTEGER NOT NULL DEFAULT 0;

ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_physical_right_leg INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_right_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_left_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_right_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_left_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_torso;
ALTER TABLE player_characters DROP COLUMN damage_resistance_physical_head;

ALTER TABLE combatants DROP COLUMN damage_resistance_physical_right_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_left_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_right_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_left_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_torso;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_head;
