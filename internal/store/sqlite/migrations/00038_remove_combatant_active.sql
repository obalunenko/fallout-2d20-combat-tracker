-- +goose Up
DROP INDEX IF EXISTS idx_combatants_one_active_per_encounter;

ALTER TABLE combatants DROP COLUMN active;

-- +goose Down
ALTER TABLE combatants
ADD COLUMN active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1));

UPDATE combatants
SET active = 1
WHERE EXISTS (
    SELECT 1
    FROM encounters e
    WHERE e.id = combatants.encounter_id
      AND e.turn_index = combatants.position
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_one_active_per_encounter
ON combatants(encounter_id)
WHERE active = 1;
