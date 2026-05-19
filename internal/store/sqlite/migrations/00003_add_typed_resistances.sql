-- +goose Up
ALTER TABLE combatants ADD COLUMN damage_resistance_physical INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_energy INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_radiation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE combatants ADD COLUMN damage_resistance_poison INTEGER NOT NULL DEFAULT 0;

UPDATE combatants
SET damage_resistance_physical = damage_resistance
WHERE damage_resistance_physical = 0;

-- +goose Down
ALTER TABLE combatants DROP COLUMN damage_resistance_poison;
ALTER TABLE combatants DROP COLUMN damage_resistance_radiation;
ALTER TABLE combatants DROP COLUMN damage_resistance_energy;
ALTER TABLE combatants DROP COLUMN damage_resistance_physical;
