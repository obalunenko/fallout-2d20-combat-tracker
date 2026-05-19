-- name: EnsureAppStateRow :exec
INSERT OR IGNORE INTO app_state (id, active_campaign_id)
VALUES (1, NULL);

-- name: GetActiveCampaign :one
SELECT c.id, c.name, c.start_date, c.updated_at
FROM campaigns c
JOIN app_state s ON s.id = 1
WHERE c.id = s.active_campaign_id;

-- name: SetActiveCampaign :execrows
UPDATE app_state
SET active_campaign_id = sqlc.arg(campaign_id)
WHERE id = 1
  AND EXISTS (
    SELECT 1
    FROM campaigns c
    WHERE c.id = sqlc.arg(campaign_id)
  );

-- name: InsertCampaign :exec
INSERT INTO campaigns (id, name, start_date, created_at, updated_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(name),
  sqlc.arg(start_date),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: TouchCampaign :exec
UPDATE campaigns
SET updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE id = sqlc.arg(campaign_id);

-- name: ListCampaigns :many
SELECT id, name, start_date, updated_at
FROM campaigns
ORDER BY updated_at DESC, id DESC;

-- name: InsertPlayer :exec
INSERT INTO players (id, campaign_id, name, created_at, updated_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(campaign_id),
  sqlc.arg(name),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: InsertPlayerCharacter :exec
INSERT INTO player_characters (
  id, player_id, campaign_id, name, level, initiative, hp, defense,
  damage_resistance_physical, damage_resistance_energy, damage_resistance_radiation, damage_resistance_poison,
  damage_resistance_physical_immune, damage_resistance_energy_immune, damage_resistance_radiation_immune, damage_resistance_poison_immune,
  active, created_at, updated_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(player_id),
  sqlc.arg(campaign_id),
  sqlc.arg(name),
  sqlc.arg(level),
  sqlc.arg(initiative),
  sqlc.arg(hp),
  sqlc.arg(defense),
  sqlc.arg(damage_resistance_physical),
  sqlc.arg(damage_resistance_energy),
  sqlc.arg(damage_resistance_radiation),
  sqlc.arg(damage_resistance_poison),
  sqlc.arg(damage_resistance_physical_immune),
  sqlc.arg(damage_resistance_energy_immune),
  sqlc.arg(damage_resistance_radiation_immune),
  sqlc.arg(damage_resistance_poison_immune),
  sqlc.arg(active),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: ListActivePartyCharactersByCampaignID :many
SELECT
  pc.id,
  p.name AS player_name,
  pc.name AS character_name,
  pc.level,
  pc.initiative,
  pc.hp,
  pc.defense,
  pc.damage_resistance_physical,
  pc.damage_resistance_energy,
  pc.damage_resistance_radiation,
  pc.damage_resistance_poison,
  pc.damage_resistance_physical_immune,
  pc.damage_resistance_energy_immune,
  pc.damage_resistance_radiation_immune,
  pc.damage_resistance_poison_immune
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
WHERE pc.campaign_id = sqlc.arg(campaign_id) AND pc.active = 1
ORDER BY p.name COLLATE NOCASE ASC, pc.name COLLATE NOCASE ASC;

-- name: GetLatestEncounterByCampaignID :one
SELECT id, campaign_id, name, round, turn_index, party_ap, gm_threat
FROM encounters
WHERE deleted_at IS NULL AND campaign_id = sqlc.arg(campaign_id)
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
INSERT INTO encounters (id, campaign_id, name, round, turn_index, party_ap, gm_threat, updated_at, deleted_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(campaign_id),
  sqlc.arg(name),
  sqlc.arg(round),
  sqlc.arg(turn_index),
  sqlc.arg(party_ap),
  sqlc.arg(gm_threat),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  NULL
)
ON CONFLICT(id) DO UPDATE SET
	campaign_id = excluded.campaign_id,
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

-- name: ListEncounterSummariesByCampaignID :many
SELECT e.id, e.campaign_id, e.name, e.round, COUNT(c.id) AS combatants, e.updated_at
FROM encounters e
LEFT JOIN combatants c ON c.encounter_id = e.id
WHERE e.deleted_at IS NULL AND e.campaign_id = sqlc.arg(campaign_id)
GROUP BY e.id, e.campaign_id, e.name, e.round, e.updated_at
ORDER BY e.updated_at DESC, e.id DESC;

-- name: ActivateEncounterByCampaign :execrows
UPDATE encounters
SET updated_at = CASE
  WHEN STRFTIME('%Y-%m-%d %H:%M:%f', 'now') <= COALESCE((SELECT MAX(e2.updated_at) FROM encounters e2 WHERE e2.deleted_at IS NULL AND e2.campaign_id = sqlc.arg(campaign_id)), '')
    THEN STRFTIME(
      '%Y-%m-%d %H:%M:%f',
      COALESCE((SELECT MAX(e3.updated_at) FROM encounters e3 WHERE e3.deleted_at IS NULL AND e3.campaign_id = sqlc.arg(campaign_id)), STRFTIME('%Y-%m-%d %H:%M:%f', 'now')),
      '+0.001 seconds'
    )
  ELSE STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
END
WHERE encounters.id = sqlc.arg(encounter_id) AND encounters.deleted_at IS NULL AND encounters.campaign_id = sqlc.arg(campaign_id);

-- name: SoftDeleteEncounterByCampaign :execrows
UPDATE encounters
SET deleted_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE encounters.id = sqlc.arg(encounter_id) AND encounters.deleted_at IS NULL AND encounters.campaign_id = sqlc.arg(campaign_id);

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

-- name: ListEncounterPartyTemplatesByCampaignID :many
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
  WHERE c.side = 'party'
    AND e.deleted_at IS NULL
    AND e.campaign_id = sqlc.arg(campaign_id)
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
