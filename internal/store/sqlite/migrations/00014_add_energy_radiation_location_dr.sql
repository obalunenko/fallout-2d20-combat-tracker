-- +goose Up
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_right_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_right_leg INTEGER NOT NULL DEFAULT 0;

ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_energy_right_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_head INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_torso INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_left_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_right_arm INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_left_leg INTEGER NOT NULL DEFAULT 0;
ALTER TABLE player_characters ADD COLUMN damage_resistance_radiation_right_leg INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_right_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_left_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_right_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_left_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_torso;
ALTER TABLE player_characters DROP COLUMN damage_resistance_radiation_head;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_right_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_left_leg;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_right_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_left_arm;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_torso;
ALTER TABLE player_characters DROP COLUMN damage_resistance_energy_head;

ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_right_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_left_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_right_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_left_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_torso;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_head;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_right_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_left_leg;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_right_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_left_arm;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_torso;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_head;
