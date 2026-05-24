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
  id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense, torso_only, active, availability_status, created_at, updated_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(player_id),
  sqlc.arg(campaign_id),
  sqlc.arg(name),
  sqlc.arg(level),
  sqlc.arg(initiative),
  sqlc.arg(hp),
  sqlc.arg(max_hp),
  sqlc.arg(defense),
  sqlc.arg(torso_only),
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
SET campaign_id = sqlc.arg(campaign_id),
    name = sqlc.arg(name),
    level = sqlc.arg(level),
    initiative = sqlc.arg(initiative),
    hp = sqlc.arg(hp),
    max_hp = sqlc.arg(max_hp),
    defense = sqlc.arg(defense),
    torso_only = sqlc.arg(torso_only),
    active = 1,
    availability_status = sqlc.arg(availability_status),
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE id = sqlc.arg(character_id);

-- name: ListInactiveCurrentPlayerCharacterIDsByCampaignID :many
SELECT id
FROM player_characters
WHERE campaign_id = sqlc.arg(campaign_id)
  AND active = 1
  AND availability_status = 'inactive';

-- name: UpsertPlayerCharacterResistanceGlobal :exec
INSERT INTO player_character_resistance_global (
  player_character_id,
  damage_type_id,
  resistance,
  immune,
  updated_at
)
VALUES (
  sqlc.arg(player_character_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(resistance),
  sqlc.arg(immune),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (player_character_id, damage_type_id) DO UPDATE SET
  resistance = excluded.resistance,
  immune = excluded.immune,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: UpsertPlayerCharacterResistanceByLocation :exec
INSERT INTO player_character_resistance_by_location (
  player_character_id,
  damage_type_id,
  body_location_id,
  resistance,
  updated_at
)
VALUES (
  sqlc.arg(player_character_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(body_location_id),
  sqlc.arg(resistance),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (player_character_id, damage_type_id, body_location_id) DO UPDATE SET
  resistance = excluded.resistance,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: ListActivePartyCharactersByCampaignID :many
SELECT
  pc.id,
  p.name AS player_name,
  pc.name AS character_name,
  pc.level,
  pc.initiative,
  pc.hp,
  pc.max_hp,
  pc.defense,
  pc.torso_only,
  pc.availability_status
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
WHERE pc.campaign_id = sqlc.arg(campaign_id) AND pc.active = 1
ORDER BY p.name COLLATE NOCASE ASC, pc.name COLLATE NOCASE ASC;

-- name: ListActivePlayerCharacterResistanceGlobalByCampaignID :many
SELECT
  crg.player_character_id,
  dt.code AS damage_type,
  crg.resistance,
  crg.immune
FROM player_character_resistance_global crg
JOIN player_characters pc ON pc.id = crg.player_character_id
JOIN damage_types dt ON dt.id = crg.damage_type_id
WHERE pc.campaign_id = sqlc.arg(campaign_id)
  AND pc.active = 1
ORDER BY pc.name COLLATE NOCASE ASC, pc.id DESC, dt.id ASC;

-- name: ListActivePlayerCharacterResistanceByLocationByCampaignID :many
SELECT
  crl.player_character_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  crl.resistance
FROM player_character_resistance_by_location crl
JOIN player_characters pc ON pc.id = crl.player_character_id
JOIN damage_types dt ON dt.id = crl.damage_type_id
JOIN body_locations bl ON bl.id = crl.body_location_id
WHERE pc.campaign_id = sqlc.arg(campaign_id)
  AND pc.active = 1
ORDER BY pc.name COLLATE NOCASE ASC, pc.id DESC, dt.id ASC, bl.id ASC;

-- name: GetLatestEncounterByCampaignID :one
SELECT id, campaign_id, name, round, turn_index, party_ap, gm_threat,
       difficulty_label, difficulty_score,
       party_count, party_avg_level, party_xp_budget,
       enemy_count, enemy_avg_level, enemy_total_xp
FROM encounters
WHERE deleted_at IS NULL AND campaign_id = sqlc.arg(campaign_id)
ORDER BY updated_at DESC, id DESC
LIMIT 1;

-- name: GetEncounterByIDByCampaignID :one
SELECT id, campaign_id, name, round, turn_index, party_ap, gm_threat,
       difficulty_label, difficulty_score,
       party_count, party_avg_level, party_xp_budget,
       enemy_count, enemy_avg_level, enemy_total_xp
FROM encounters
WHERE deleted_at IS NULL
  AND campaign_id = sqlc.arg(campaign_id)
  AND id = sqlc.arg(encounter_id);

-- name: ListCombatantsByEncounterID :many
SELECT
  c.id,
  c.player_character_id,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.name ELSE c.name END AS TEXT) AS name,
  c.side,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.level ELSE c.level END AS INTEGER) AS level,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN 0 ELSE c.xp END AS INTEGER) AS xp,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.initiative ELSE c.initiative END AS INTEGER) AS initiative,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.hp ELSE c.hp END AS INTEGER) AS hp,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.max_hp ELSE c.max_hp END AS INTEGER) AS max_hp,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.defense ELSE c.defense END AS INTEGER) AS defense,
  CAST(CASE WHEN c.side = 'party' AND pc.id IS NOT NULL THEN pc.torso_only ELSE c.torso_only END AS INTEGER) AS torso_only,
  c.active,
  CAST(CASE
    WHEN c.side = 'party' AND pc.id IS NOT NULL THEN CASE WHEN pc.hp <= 0 THEN 1 ELSE 0 END
    ELSE c.defeated
  END AS INTEGER) AS defeated
FROM combatants c
JOIN encounters e ON e.id = c.encounter_id
LEFT JOIN player_characters pc ON pc.id = c.player_character_id
  AND pc.campaign_id = e.campaign_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC;

-- name: ListCombatantResistanceGlobalByEncounterID :many
SELECT
  crg.combatant_id,
  dt.code AS damage_type,
  crg.resistance,
  crg.immune
FROM combatant_resistance_global crg
JOIN combatants c ON c.id = crg.combatant_id
JOIN damage_types dt ON dt.id = crg.damage_type_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC, dt.id ASC;

-- name: ListCombatantResistanceByLocationByEncounterID :many
SELECT
  crl.combatant_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  crl.resistance
FROM combatant_resistance_by_location crl
JOIN combatants c ON c.id = crl.combatant_id
JOIN damage_types dt ON dt.id = crl.damage_type_id
JOIN body_locations bl ON bl.id = crl.body_location_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC, dt.id ASC, bl.id ASC;

-- name: ListLinkedPlayerCharacterResistanceGlobalByEncounterID :many
SELECT
  c.id AS combatant_id,
  pcrg.player_character_id,
  dt.code AS damage_type,
  pcrg.resistance,
  pcrg.immune
FROM combatants c
JOIN encounters e ON e.id = c.encounter_id
JOIN player_characters pc ON pc.id = c.player_character_id
  AND pc.campaign_id = e.campaign_id
JOIN player_character_resistance_global pcrg ON pcrg.player_character_id = pc.id
JOIN damage_types dt ON dt.id = pcrg.damage_type_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
  AND c.side = 'party'
ORDER BY c.position ASC, dt.id ASC;

-- name: ListLinkedPlayerCharacterResistanceByLocationByEncounterID :many
SELECT
  c.id AS combatant_id,
  pcrl.player_character_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  pcrl.resistance
FROM combatants c
JOIN encounters e ON e.id = c.encounter_id
JOIN player_characters pc ON pc.id = c.player_character_id
  AND pc.campaign_id = e.campaign_id
JOIN player_character_resistance_by_location pcrl ON pcrl.player_character_id = pc.id
JOIN damage_types dt ON dt.id = pcrl.damage_type_id
JOIN body_locations bl ON bl.id = pcrl.body_location_id
WHERE c.encounter_id = sqlc.arg(encounter_id)
  AND c.side = 'party'
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
  difficulty_label, difficulty_score,
  party_count, party_avg_level, party_xp_budget,
  enemy_count, enemy_avg_level, enemy_total_xp,
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
  sqlc.arg(difficulty_label),
  sqlc.arg(difficulty_score),
  sqlc.arg(party_count),
  sqlc.arg(party_avg_level),
  sqlc.arg(party_xp_budget),
  sqlc.arg(enemy_count),
  sqlc.arg(enemy_avg_level),
  sqlc.arg(enemy_total_xp),
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
	difficulty_label = excluded.difficulty_label,
	difficulty_score = excluded.difficulty_score,
	party_count = excluded.party_count,
	party_avg_level = excluded.party_avg_level,
	party_xp_budget = excluded.party_xp_budget,
	enemy_count = excluded.enemy_count,
	enemy_avg_level = excluded.enemy_avg_level,
	enemy_total_xp = excluded.enemy_total_xp,
	deleted_at = NULL,
	updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: DeleteCombatantsByEncounterID :exec
DELETE FROM combatants
WHERE encounter_id = sqlc.arg(encounter_id);

-- name: InsertCombatant :exec
INSERT INTO combatants (
	id, encounter_id, player_character_id, name, side, torso_only, level, xp, initiative, hp, max_hp, defense, active, defeated, position, created_at, updated_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
  sqlc.narg(player_character_id),
  sqlc.arg(name),
  sqlc.arg(side),
  sqlc.arg(torso_only),
  sqlc.arg(level),
  sqlc.arg(xp),
	  sqlc.arg(initiative),
	  sqlc.arg(hp),
	  sqlc.arg(max_hp),
	  sqlc.arg(defense),
	  sqlc.arg(active),
	  sqlc.arg(defeated),
	  sqlc.arg(position),
	  STRFTIME('%Y-%m-%d %H:%M:%f', 'now'),
	  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
	);

-- name: UpsertCombatantResistanceGlobal :exec
INSERT INTO combatant_resistance_global (
  combatant_id,
  damage_type_id,
  resistance,
  immune,
  updated_at
)
VALUES (
  sqlc.arg(combatant_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(resistance),
  sqlc.arg(immune),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (combatant_id, damage_type_id) DO UPDATE SET
  resistance = excluded.resistance,
  immune = excluded.immune,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: UpsertCombatantResistanceByLocation :exec
INSERT INTO combatant_resistance_by_location (
  combatant_id,
  damage_type_id,
  body_location_id,
  resistance,
  updated_at
)
VALUES (
  sqlc.arg(combatant_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(body_location_id),
  sqlc.arg(resistance),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (combatant_id, damage_type_id, body_location_id) DO UPDATE SET
  resistance = excluded.resistance,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: ListEncounterSummariesByCampaignID :many
SELECT
  e.id,
  e.campaign_id,
  e.name,
  e.round,
  COUNT(c.id) AS combatants,
  e.difficulty_label,
  e.difficulty_score,
  e.party_count,
  e.party_avg_level,
  e.party_xp_budget,
  e.enemy_count,
  e.enemy_avg_level,
  e.enemy_total_xp,
  e.updated_at
FROM encounters e
LEFT JOIN combatants c ON c.encounter_id = e.id
WHERE e.deleted_at IS NULL AND e.campaign_id = sqlc.arg(campaign_id)
GROUP BY
  e.id, e.campaign_id, e.name, e.round,
  e.difficulty_label, e.difficulty_score,
  e.party_count, e.party_avg_level, e.party_xp_budget,
  e.enemy_count, e.enemy_avg_level, e.enemy_total_xp,
  e.updated_at
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
  id, name, name_key, torso_only, level, xp, initiative, hp, max_hp, defense, created_at, updated_at, deleted_at
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(name),
  sqlc.arg(name_key),
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
ON CONFLICT(name_key) DO UPDATE SET
  name = excluded.name,
  torso_only = excluded.torso_only,
  level = excluded.level,
  xp = excluded.xp,
  initiative = excluded.initiative,
  hp = excluded.hp,
  max_hp = excluded.max_hp,
  defense = excluded.defense,
  deleted_at = NULL,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: GetMonsterTemplateIDByNameKey :one
SELECT id
FROM monster_templates
WHERE name_key = sqlc.arg(name_key);

-- name: UpsertMonsterTemplateResistanceGlobal :exec
INSERT INTO monster_template_resistance_global (
  monster_template_id,
  damage_type_id,
  resistance,
  immune,
  updated_at
)
VALUES (
  sqlc.arg(monster_template_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(resistance),
  sqlc.arg(immune),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (monster_template_id, damage_type_id) DO UPDATE SET
  resistance = excluded.resistance,
  immune = excluded.immune,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: UpsertMonsterTemplateResistanceByLocation :exec
INSERT INTO monster_template_resistance_by_location (
  monster_template_id,
  damage_type_id,
  body_location_id,
  resistance,
  updated_at
)
VALUES (
  sqlc.arg(monster_template_id),
  sqlc.arg(damage_type_id),
  sqlc.arg(body_location_id),
  sqlc.arg(resistance),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (monster_template_id, damage_type_id, body_location_id) DO UPDATE SET
  resistance = excluded.resistance,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

-- name: ListMonsterTemplates :many
SELECT
  mt.id,
  mt.name,
  mt.level,
  mt.xp,
  mt.initiative,
  mt.hp,
  mt.max_hp,
  mt.defense,
  mt.torso_only
FROM monster_templates mt
WHERE mt.deleted_at IS NULL
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC;

-- name: ListMonsterTemplateResistanceGlobal :many
SELECT
  mtg.monster_template_id,
  dt.code AS damage_type,
  mtg.resistance,
  mtg.immune
FROM monster_template_resistance_global mtg
JOIN monster_templates mt ON mt.id = mtg.monster_template_id
JOIN damage_types dt ON dt.id = mtg.damage_type_id
WHERE mt.deleted_at IS NULL
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC, dt.id ASC;

-- name: ListMonsterTemplateResistanceByLocation :many
SELECT
  mtl.monster_template_id,
  dt.code AS damage_type,
  bl.code AS body_location,
  mtl.resistance
FROM monster_template_resistance_by_location mtl
JOIN monster_templates mt ON mt.id = mtl.monster_template_id
JOIN damage_types dt ON dt.id = mtl.damage_type_id
JOIN body_locations bl ON bl.id = mtl.body_location_id
WHERE mt.deleted_at IS NULL
ORDER BY mt.name COLLATE NOCASE ASC, mt.id DESC, dt.id ASC, bl.id ASC;
