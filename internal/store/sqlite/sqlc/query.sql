-- name: EnsureAppStateRow :exec
INSERT OR IGNORE INTO app_state (id, active_campaign_id)
VALUES (1, NULL);

-- name: ListDamageTypes :many
SELECT id, code
FROM damage_types
ORDER BY code ASC;

-- name: ListBodyLocations :many
SELECT id, code
FROM body_locations
ORDER BY code ASC;

-- name: UpsertStatProfile :exec
INSERT INTO stat_profiles (
  id, torso_only, level, xp, initiative, hp, max_hp, defense, created_at, updated_at, deleted_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(torso_only),
  sqlc.arg(level),
  sqlc.arg(xp),
  sqlc.arg(initiative),
  sqlc.arg(hp),
  sqlc.arg(max_hp),
  sqlc.arg(defense),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  NULL
)
ON CONFLICT(id) DO UPDATE SET
  torso_only = excluded.torso_only,
  level = excluded.level,
  xp = excluded.xp,
  initiative = excluded.initiative,
  hp = excluded.hp,
  max_hp = excluded.max_hp,
  defense = excluded.defense,
  deleted_at = NULL,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: DeleteStatProfileResistancesByProfileID :exec
DELETE FROM stat_profile_resistance_by_location
WHERE stat_profile_id = sqlc.arg(stat_profile_id);

-- name: UpsertStatProfileResistanceGlobal :exec
INSERT INTO stat_profile_resistance_by_location (
  stat_profile_id,
  damage_type_id,
  body_location_id,
  resistance,
  immune,
  updated_at
)
VALUES (
  sqlc.arg(stat_profile_id),
  sqlc.arg(damage_type_id),
  (SELECT id FROM body_locations WHERE code = 'global'),
  sqlc.arg(resistance),
  sqlc.arg(immune),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
  resistance = excluded.resistance,
  immune = excluded.immune,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: UpsertStatProfileResistanceByLocation :exec
INSERT INTO stat_profile_resistance_by_location (
  stat_profile_id,
  damage_type_id,
  body_location_id,
  resistance,
  updated_at
)
VALUES (
  sqlc.arg(stat_profile_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(body_location_id),
  sqlc.arg(resistance),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (stat_profile_id, damage_type_id, body_location_id) DO UPDATE SET
  resistance = excluded.resistance,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

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

-- name: GetCampaignByID :one
SELECT id, name, start_date, updated_at
FROM campaigns
WHERE id = sqlc.arg(campaign_id);

-- name: UpdateCampaignByID :execrows
UPDATE campaigns
SET name = sqlc.arg(name),
    start_date = sqlc.arg(start_date),
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE id = sqlc.arg(campaign_id);

-- name: DeletePlayersByCampaignID :exec
DELETE FROM players
WHERE campaign_id = sqlc.arg(campaign_id);

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
  id, player_id, stat_profile_id, name, active, availability_status, created_at, updated_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(player_id),
  sqlc.arg(stat_profile_id),
  sqlc.arg(name),
  sqlc.arg(active),
  sqlc.arg(availability_status),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: ListPlayerIDsAndNamesByCampaignID :many
SELECT id, name
FROM players
WHERE campaign_id = sqlc.arg(campaign_id);

-- name: GetActivePlayerCharacterByPlayerID :one
SELECT id, name
FROM player_characters
WHERE player_id = sqlc.arg(player_id)
  AND active = 1
ORDER BY updated_at DESC, id DESC
LIMIT 1;

-- name: DeactivateActiveCharactersByPlayerID :exec
UPDATE player_characters
SET active = 0,
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE player_id = sqlc.arg(player_id)
  AND active = 1;

-- name: UpdateActivePlayerCharacterByID :exec
UPDATE player_characters
SET name = sqlc.arg(name),
    active = 1,
    availability_status = sqlc.arg(availability_status),
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE id = sqlc.arg(character_id);

-- name: ListInactiveCurrentPlayerCharacterIDsByCampaignID :many
SELECT pc.id
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
WHERE p.campaign_id = sqlc.arg(campaign_id)
  AND pc.active = 1
  AND pc.availability_status = 'inactive';

-- name: ListActivePartyCharactersByCampaignID :many
SELECT
  pc.id,
  p.name AS player_name,
  pc.name AS character_name,
  sp.level,
  sp.initiative,
  sp.hp,
  sp.max_hp,
  sp.defense,
  sp.torso_only,
  pc.availability_status
FROM player_characters pc
JOIN stat_profiles sp ON sp.id = pc.stat_profile_id
JOIN players p ON p.id = pc.player_id
WHERE p.campaign_id = sqlc.arg(campaign_id) AND pc.active = 1
ORDER BY p.name COLLATE NOCASE ASC, pc.name COLLATE NOCASE ASC;

-- name: ListActivePlayerCharacterResistanceGlobalByCampaignID :many
SELECT
  pc.id AS player_character_id,
  dt.code AS damage_type,
  sprg.resistance,
  sprg.immune
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
JOIN stat_profile_resistance_by_location sprg ON sprg.stat_profile_id = pc.stat_profile_id
JOIN body_locations bl ON bl.id = sprg.body_location_id
  AND bl.code = 'global'
JOIN damage_types dt ON dt.id = sprg.damage_type_id
WHERE p.campaign_id = sqlc.arg(campaign_id)
  AND pc.active = 1
ORDER BY pc.name COLLATE NOCASE ASC, pc.id DESC, dt.id ASC;

-- name: ListActivePlayerCharacterResistanceByLocationByCampaignID :many
SELECT
  pc.id AS player_character_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  sprl.resistance
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
JOIN stat_profile_resistance_by_location sprl ON sprl.stat_profile_id = pc.stat_profile_id
JOIN damage_types dt ON dt.id = sprl.damage_type_id
JOIN body_locations bl ON bl.id = sprl.body_location_id
WHERE p.campaign_id = sqlc.arg(campaign_id)
  AND pc.active = 1
  AND bl.code <> 'global'
ORDER BY pc.name COLLATE NOCASE ASC, pc.id DESC, dt.id ASC, bl.id ASC;

-- name: GetLatestEncounterByCampaignID :one
SELECT id, campaign_id, name, round, turn_index, party_ap, gm_threat
FROM encounters
WHERE deleted_at IS NULL AND campaign_id = sqlc.arg(campaign_id)
ORDER BY updated_at DESC, id DESC
LIMIT 1;

-- name: GetEncounterByIDByCampaignID :one
SELECT id, campaign_id, name, round, turn_index, party_ap, gm_threat
FROM encounters
WHERE deleted_at IS NULL
  AND campaign_id = sqlc.arg(campaign_id)
  AND id = sqlc.arg(encounter_id);

-- name: ListCombatantsByEncounterID :many
SELECT
  c.id,
  c.player_character_id,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pc.name ELSE c.name END AS TEXT) AS name,
  c.side,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.level ELSE csp.level END AS INTEGER) AS level,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN 0 ELSE csp.xp END AS INTEGER) AS xp,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.initiative ELSE csp.initiative END AS INTEGER) AS initiative,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.hp ELSE csp.hp END AS INTEGER) AS hp,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.max_hp ELSE csp.max_hp END AS INTEGER) AS max_hp,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.defense ELSE csp.defense END AS INTEGER) AS defense,
  CAST(CASE WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN pcsp.torso_only ELSE csp.torso_only END AS INTEGER) AS torso_only,
  CAST(CASE WHEN c.position = e.turn_index THEN 1 ELSE 0 END AS INTEGER) AS active,
  CAST(CASE
    WHEN c.side = 'party' AND pcp.id IS NOT NULL THEN CASE WHEN pcsp.hp <= 0 THEN 1 ELSE 0 END
    ELSE c.defeated
  END AS INTEGER) AS defeated
FROM combatants c
JOIN stat_profiles csp ON csp.id = c.stat_profile_id
JOIN encounters e ON e.id = c.encounter_id
LEFT JOIN player_characters pc ON pc.id = c.player_character_id
LEFT JOIN players pcp ON pcp.id = pc.player_id
  AND pcp.campaign_id = e.campaign_id
LEFT JOIN stat_profiles pcsp ON pcsp.id = pc.stat_profile_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC;

-- name: ListCombatantResistanceGlobalByEncounterID :many
SELECT
  c.id AS combatant_id,
  dt.code AS damage_type,
  sprg.resistance,
  sprg.immune
FROM combatants c
JOIN stat_profile_resistance_by_location sprg ON sprg.stat_profile_id = c.stat_profile_id
JOIN body_locations bl ON bl.id = sprg.body_location_id
  AND bl.code = 'global'
JOIN damage_types dt ON dt.id = sprg.damage_type_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC, dt.id ASC;

-- name: ListCombatantResistanceByLocationByEncounterID :many
SELECT
  c.id AS combatant_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  sprl.resistance
FROM combatants c
JOIN stat_profile_resistance_by_location sprl ON sprl.stat_profile_id = c.stat_profile_id
JOIN damage_types dt ON dt.id = sprl.damage_type_id
JOIN body_locations bl ON bl.id = sprl.body_location_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
  AND bl.code <> 'global'
ORDER BY c.position ASC, dt.id ASC, bl.id ASC;

-- name: ListLinkedPlayerCharacterResistanceGlobalByEncounterID :many
SELECT
  c.id AS combatant_id,
  pc.id AS player_character_id,
  dt.code AS damage_type,
  sprg.resistance,
  sprg.immune
FROM combatants c
JOIN encounters e ON e.id = c.encounter_id
JOIN player_characters pc ON pc.id = c.player_character_id
JOIN players p ON p.id = pc.player_id
  AND p.campaign_id = e.campaign_id
JOIN stat_profile_resistance_by_location sprg ON sprg.stat_profile_id = pc.stat_profile_id
JOIN body_locations bl ON bl.id = sprg.body_location_id
  AND bl.code = 'global'
JOIN damage_types dt ON dt.id = sprg.damage_type_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
  AND c.side = 'party'
ORDER BY c.position ASC, dt.id ASC;

-- name: ListLinkedPlayerCharacterResistanceByLocationByEncounterID :many
SELECT
  c.id AS combatant_id,
  pc.id AS player_character_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  sprl.resistance
FROM combatants c
JOIN encounters e ON e.id = c.encounter_id
JOIN player_characters pc ON pc.id = c.player_character_id
JOIN players p ON p.id = pc.player_id
  AND p.campaign_id = e.campaign_id
JOIN stat_profile_resistance_by_location sprl ON sprl.stat_profile_id = pc.stat_profile_id
JOIN damage_types dt ON dt.id = sprl.damage_type_id
JOIN body_locations bl ON bl.id = sprl.body_location_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
  AND c.side = 'party'
  AND bl.code <> 'global'
ORDER BY c.position ASC, dt.id ASC, bl.id ASC;

-- name: ListCombatantIDsByEncounterID :many
SELECT id
FROM combatants
WHERE encounter_id = sqlc.arg(encounter_id);

-- name: ListEncounterIDsByCampaignID :many
SELECT id
FROM encounters
WHERE deleted_at IS NULL AND campaign_id = sqlc.arg(campaign_id)
ORDER BY updated_at DESC, id DESC;

-- name: UpsertEncounter :exec
INSERT INTO encounters (
  id, campaign_id, name, round, turn_index, party_ap, gm_threat,
  created_at, updated_at, deleted_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(campaign_id),
  sqlc.arg(name),
  sqlc.arg(round),
  sqlc.arg(turn_index),
  sqlc.arg(party_ap),
  sqlc.arg(gm_threat),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
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
	id, encounter_id, stat_profile_id, player_character_id, name, side, defeated, position, created_at, updated_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
  sqlc.arg(stat_profile_id),
  sqlc.narg(player_character_id),
  sqlc.arg(name),
  sqlc.arg(side),
  sqlc.arg(defeated),
  sqlc.arg(position),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: ListEncounterSummariesByCampaignID :many
SELECT
  e.id,
  e.campaign_id,
  e.name,
  e.round,
  COUNT(c.id) AS combatants,
  e.updated_at
FROM encounters e
LEFT JOIN combatants c ON c.encounter_id = e.id
WHERE e.deleted_at IS NULL AND e.campaign_id = sqlc.arg(campaign_id)
GROUP BY
  e.id, e.campaign_id, e.name, e.round, e.updated_at
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
INSERT INTO encounter_logs (id, encounter_id, round, message, created_at, updated_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
  sqlc.arg(round),
  sqlc.arg(message),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
);

-- name: ListEncounterLogsByEncounterID :many
SELECT round, message, created_at
FROM encounter_logs
WHERE encounter_id = sqlc.arg(encounter_id)
ORDER BY created_at DESC, rowid DESC;

-- name: UpsertMonsterTemplate :exec
INSERT INTO monster_templates (
  id, stat_profile_id, name, created_at, updated_at, deleted_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(stat_profile_id),
  sqlc.arg(name),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
  NULL
)
ON CONFLICT(id) DO UPDATE SET
  stat_profile_id = excluded.stat_profile_id,
  name = excluded.name,
  deleted_at = NULL,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: GetMonsterTemplateIDByName :one
SELECT id
FROM monster_templates
WHERE lower(trim(name)) = lower(trim(sqlc.arg(name)));

-- name: ListMonsterTemplates :many
SELECT
  mt.id,
  mt.name,
  sp.level,
  sp.xp,
  sp.initiative,
  sp.hp,
  sp.max_hp,
  sp.defense,
  sp.torso_only
FROM monster_templates mt
JOIN stat_profiles sp ON sp.id = mt.stat_profile_id
WHERE mt.deleted_at IS NULL
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC;

-- name: ListMonsterTemplateResistanceGlobal :many
SELECT
  mt.id AS monster_template_id,
  dt.code AS damage_type,
  sprg.resistance,
  sprg.immune
FROM monster_templates mt
JOIN stat_profile_resistance_by_location sprg ON sprg.stat_profile_id = mt.stat_profile_id
JOIN body_locations bl ON bl.id = sprg.body_location_id
  AND bl.code = 'global'
JOIN damage_types dt ON dt.id = sprg.damage_type_id
WHERE mt.deleted_at IS NULL
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC, dt.id ASC;

-- name: ListMonsterTemplateResistanceByLocation :many
SELECT
  mt.id AS monster_template_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  sprl.resistance
FROM monster_templates mt
JOIN stat_profile_resistance_by_location sprl ON sprl.stat_profile_id = mt.stat_profile_id
JOIN damage_types dt ON dt.id = sprl.damage_type_id
JOIN body_locations bl ON bl.id = sprl.body_location_id
WHERE mt.deleted_at IS NULL
  AND bl.code <> 'global'
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC, dt.id ASC, bl.id ASC;
