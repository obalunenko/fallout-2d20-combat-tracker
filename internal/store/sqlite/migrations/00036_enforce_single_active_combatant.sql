-- +goose Up
UPDATE combatants
SET active = 0
WHERE active = 1
  AND id NOT IN (
    SELECT id
    FROM (
        SELECT
            c.id,
            ROW_NUMBER() OVER (
                PARTITION BY c.encounter_id
                ORDER BY
                    CASE WHEN c.position = e.turn_index THEN 0 ELSE 1 END,
                    c.position ASC,
                    c.id ASC
            ) AS rn
        FROM combatants c
        JOIN encounters e ON e.id = c.encounter_id
        WHERE c.active = 1
    ) AS ranked
    WHERE rn = 1
  );

CREATE UNIQUE INDEX IF NOT EXISTS idx_combatants_one_active_per_encounter
ON combatants(encounter_id)
WHERE active = 1;

-- +goose Down
DROP INDEX IF EXISTS idx_combatants_one_active_per_encounter;
