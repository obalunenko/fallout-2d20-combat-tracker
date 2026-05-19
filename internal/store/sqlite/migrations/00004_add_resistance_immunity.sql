-- +goose Up
ALTER TABLE combatants ADD COLUMN damage_resistance_physical_immune INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy_immune INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation_immune INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_poison_immune INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE combatants DROP COLUMN damage_resistance_poison_immune;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation_immune;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy_immune;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical_immune;
