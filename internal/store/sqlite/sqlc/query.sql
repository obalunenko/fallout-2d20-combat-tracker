-- name: GetLatestEncounter :one
SELECT id, name, round, turn_index, party_ap, gm_threat
FROM encounters
WHERE deleted_at IS NULL
ORDER BY updated_at DESC, id DESC
LIMIT 1;

-- name: ListCombatantsByEncounterID :many
SELECT id, name, side, level, xp, initiative, hp, defense,
       damage_resistance_physical, damage_resistance_energy, damage_resistance_radiation, damage_resistance_poison,
       damage_resistance_physical_immune, damage_resistance_energy_immune, damage_resistance_radiation_immune, damage_resistance_poison_immune,
       active, defeated
FROM combatants
WHERE encounter_id = sqlc.arg(encounter_id)
ORDER BY position ASC;

-- name: UpsertEncounter :exec
INSERT INTO encounters (id, name, round, turn_index, party_ap, gm_threat, updated_at, deleted_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(name),
  sqlc.arg(round),
  sqlc.arg(turn_index),
  sqlc.arg(party_ap),
  sqlc.arg(gm_threat),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  NULL
)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	round = excluded.round,
	turn_index = excluded.turn_index,
	party_ap = excluded.party_ap,
	gm_threat = excluded.gm_threat,
	deleted_at = NULL,
	updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: DeleteCombatantsByEncounterID :exec
DELETE FROM combatants
WHERE encounter_id = sqlc.arg(encounter_id);

-- name: InsertCombatant :exec
INSERT INTO combatants (
	id, encounter_id, name, side, level, xp, initiative, hp, defense,
	damage_resistance, damage_resistance_physical, damage_resistance_energy, damage_resistance_radiation, damage_resistance_poison,
	damage_resistance_physical_immune, damage_resistance_energy_immune, damage_resistance_radiation_immune, damage_resistance_poison_immune,
	active, defeated, position
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
  sqlc.arg(name),
  sqlc.arg(side),
  sqlc.arg(level),
  sqlc.arg(xp),
  sqlc.arg(initiative),
  sqlc.arg(hp),
  sqlc.arg(defense),
  sqlc.arg(damage_resistance),
  sqlc.arg(damage_resistance_physical),
  sqlc.arg(damage_resistance_energy),
  sqlc.arg(damage_resistance_radiation),
  sqlc.arg(damage_resistance_poison),
  sqlc.arg(damage_resistance_physical_immune),
  sqlc.arg(damage_resistance_energy_immune),
  sqlc.arg(damage_resistance_radiation_immune),
  sqlc.arg(damage_resistance_poison_immune),
  sqlc.arg(active),
  sqlc.arg(defeated),
  sqlc.arg(position)
);

-- name: ListEncounterSummaries :many
SELECT e.id, e.name, e.round, COUNT(c.id) AS combatants, e.updated_at
FROM encounters e
LEFT JOIN combatants c ON c.encounter_id = e.id
WHERE e.deleted_at IS NULL
GROUP BY e.id, e.name, e.round, e.updated_at
ORDER BY e.updated_at DESC, e.id DESC;

-- name: ActivateEncounter :execrows
UPDATE encounters
SET updated_at = CASE
  WHEN STRFTIME('%Y-%m-%d %H:%M:%f', 'now') <= COALESCE((SELECT MAX(updated_at) FROM encounters WHERE deleted_at IS NULL), '')
    THEN STRFTIME(
      '%Y-%m-%d %H:%M:%f',
      COALESCE((SELECT MAX(updated_at) FROM encounters WHERE deleted_at IS NULL), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
      '+0.001 seconds'
    )
  ELSE STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
END
WHERE encounters.id = sqlc.arg(encounter_id) AND deleted_at IS NULL;

-- name: SoftDeleteEncounter :execrows
UPDATE encounters
SET deleted_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE encounters.id = sqlc.arg(encounter_id) AND deleted_at IS NULL;

-- name: InsertEncounterLog :exec
INSERT INTO encounter_logs (id, encounter_id, round, message, created_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
  sqlc.arg(round),
  sqlc.arg(message),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: ListEncounterLogsByEncounterID :many
SELECT round, message, created_at
FROM encounter_logs
WHERE encounter_id = sqlc.arg(encounter_id)
ORDER BY created_at DESC, rowid DESC;

-- name: ListPartyTemplates :many
WITH latest_party AS (
  SELECT
    c.name,
    c.level,
    c.xp,
    c.initiative,
    c.hp,
    c.defense,
    c.damage_resistance_physical,
    c.damage_resistance_energy,
    c.damage_resistance_radiation,
    c.damage_resistance_poison,
    c.damage_resistance_physical_immune,
    c.damage_resistance_energy_immune,
    c.damage_resistance_radiation_immune,
    c.damage_resistance_poison_immune,
    ROW_NUMBER() OVER (
      PARTITION BY LOWER(TRIM(c.name))
      ORDER BY e.updated_at DESC, e.id DESC, c.position ASC
    ) AS rn
  FROM combatants c
  JOIN encounters e ON e.id = c.encounter_id
  WHERE c.side = 'party' AND e.deleted_at IS NULL
)
SELECT
  name,
  level,
  xp,
  initiative,
  hp,
  defense,
  damage_resistance_physical,
  damage_resistance_energy,
  damage_resistance_radiation,
  damage_resistance_poison,
  damage_resistance_physical_immune,
  damage_resistance_energy_immune,
  damage_resistance_radiation_immune,
  damage_resistance_poison_immune
FROM latest_party
WHERE rn = 1
ORDER BY name COLLATE NOCASE ASC;
