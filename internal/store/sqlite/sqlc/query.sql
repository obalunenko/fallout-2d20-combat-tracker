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
INSERT INTO campaigns (id, name, start_date, updated_at)
VALUES (
  sqlc.arg(id),
  sqlc.arg(name),
  sqlc.arg(start_date),
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
INSERT INTO players (id, campaign_id, name)
VALUES (
  sqlc.arg(id),
  sqlc.arg(campaign_id),
  sqlc.arg(name)
);

-- name: InsertPlayerCharacter :exec
INSERT INTO player_characters (
  id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense, torso_only, active
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
  sqlc.arg(active)
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
    updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
WHERE id = sqlc.arg(character_id);

-- name: UpsertPlayerCharacterDefenseByLocation :exec
INSERT INTO player_character_defense_by_location (
  player_character_id,
  body_location_id,
  defense,
  updated_at
)
VALUES (
  sqlc.arg(player_character_id),
  sqlc.arg(body_location_id),
  sqlc.arg(defense),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (player_character_id, body_location_id) DO UPDATE SET
  defense = excluded.defense,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

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
WITH player_character_defense AS (
  SELECT
    cdl.player_character_id,
    MAX(CASE WHEN bl.code = 'head' THEN cdl.defense END) AS defense_head,
    MAX(CASE WHEN bl.code = 'torso' THEN cdl.defense END) AS defense_torso,
    MAX(CASE WHEN bl.code = 'left_arm' THEN cdl.defense END) AS defense_left_arm,
    MAX(CASE WHEN bl.code = 'right_arm' THEN cdl.defense END) AS defense_right_arm,
    MAX(CASE WHEN bl.code = 'left_leg' THEN cdl.defense END) AS defense_left_leg,
    MAX(CASE WHEN bl.code = 'right_leg' THEN cdl.defense END) AS defense_right_leg
  FROM player_character_defense_by_location cdl
  JOIN body_locations bl ON bl.id = cdl.body_location_id
  GROUP BY cdl.player_character_id
),
player_character_resistance_global_agg AS (
  SELECT
    crg.player_character_id,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.resistance END) AS damage_resistance_physical,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.resistance END) AS damage_resistance_energy,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.resistance END) AS damage_resistance_radiation,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.resistance END) AS damage_resistance_poison,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.immune END) AS damage_resistance_physical_immune,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.immune END) AS damage_resistance_energy_immune,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.immune END) AS damage_resistance_radiation_immune,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.immune END) AS damage_resistance_poison_immune
  FROM player_character_resistance_global crg
  JOIN damage_types dt ON dt.id = crg.damage_type_id
  GROUP BY crg.player_character_id
),
player_character_resistance_by_location_agg AS (
  SELECT
    crl.player_character_id,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_physical_head,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_physical_torso,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_physical_left_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_physical_right_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_physical_left_leg,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_physical_right_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_energy_head,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_energy_torso,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_energy_left_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_energy_right_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_energy_left_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_energy_right_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_radiation_head,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_radiation_torso,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_radiation_left_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_radiation_right_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_radiation_left_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_radiation_right_leg
  FROM player_character_resistance_by_location crl
  JOIN damage_types dt ON dt.id = crl.damage_type_id
  JOIN body_locations bl ON bl.id = crl.body_location_id
  GROUP BY crl.player_character_id
)
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
  CAST(COALESCE(cdl.defense_head, 0) AS INTEGER) AS defense_head,
  CAST(COALESCE(cdl.defense_torso, 0) AS INTEGER) AS defense_torso,
  CAST(COALESCE(cdl.defense_left_arm, 0) AS INTEGER) AS defense_left_arm,
  CAST(COALESCE(cdl.defense_right_arm, 0) AS INTEGER) AS defense_right_arm,
  CAST(COALESCE(cdl.defense_left_leg, 0) AS INTEGER) AS defense_left_leg,
  CAST(COALESCE(cdl.defense_right_leg, 0) AS INTEGER) AS defense_right_leg,
  CAST(COALESCE(crl.damage_resistance_physical_head, 0) AS INTEGER) AS damage_resistance_physical_head,
  CAST(COALESCE(crl.damage_resistance_physical_torso, 0) AS INTEGER) AS damage_resistance_physical_torso,
  CAST(COALESCE(crl.damage_resistance_physical_left_arm, 0) AS INTEGER) AS damage_resistance_physical_left_arm,
  CAST(COALESCE(crl.damage_resistance_physical_right_arm, 0) AS INTEGER) AS damage_resistance_physical_right_arm,
  CAST(COALESCE(crl.damage_resistance_physical_left_leg, 0) AS INTEGER) AS damage_resistance_physical_left_leg,
  CAST(COALESCE(crl.damage_resistance_physical_right_leg, 0) AS INTEGER) AS damage_resistance_physical_right_leg,
  CAST(COALESCE(crg.damage_resistance_physical, 0) AS INTEGER) AS damage_resistance_physical,
  CAST(COALESCE(crg.damage_resistance_energy, 0) AS INTEGER) AS damage_resistance_energy,
  CAST(COALESCE(crg.damage_resistance_radiation, 0) AS INTEGER) AS damage_resistance_radiation,
  CAST(COALESCE(crg.damage_resistance_poison, 0) AS INTEGER) AS damage_resistance_poison,
  CAST(COALESCE(crl.damage_resistance_energy_head, 0) AS INTEGER) AS damage_resistance_energy_head,
  CAST(COALESCE(crl.damage_resistance_energy_torso, 0) AS INTEGER) AS damage_resistance_energy_torso,
  CAST(COALESCE(crl.damage_resistance_energy_left_arm, 0) AS INTEGER) AS damage_resistance_energy_left_arm,
  CAST(COALESCE(crl.damage_resistance_energy_right_arm, 0) AS INTEGER) AS damage_resistance_energy_right_arm,
  CAST(COALESCE(crl.damage_resistance_energy_left_leg, 0) AS INTEGER) AS damage_resistance_energy_left_leg,
  CAST(COALESCE(crl.damage_resistance_energy_right_leg, 0) AS INTEGER) AS damage_resistance_energy_right_leg,
  CAST(COALESCE(crl.damage_resistance_radiation_head, 0) AS INTEGER) AS damage_resistance_radiation_head,
  CAST(COALESCE(crl.damage_resistance_radiation_torso, 0) AS INTEGER) AS damage_resistance_radiation_torso,
  CAST(COALESCE(crl.damage_resistance_radiation_left_arm, 0) AS INTEGER) AS damage_resistance_radiation_left_arm,
  CAST(COALESCE(crl.damage_resistance_radiation_right_arm, 0) AS INTEGER) AS damage_resistance_radiation_right_arm,
  CAST(COALESCE(crl.damage_resistance_radiation_left_leg, 0) AS INTEGER) AS damage_resistance_radiation_left_leg,
  CAST(COALESCE(crl.damage_resistance_radiation_right_leg, 0) AS INTEGER) AS damage_resistance_radiation_right_leg,
  CAST(COALESCE(crg.damage_resistance_physical_immune, 0) AS INTEGER) AS damage_resistance_physical_immune,
  CAST(COALESCE(crg.damage_resistance_energy_immune, 0) AS INTEGER) AS damage_resistance_energy_immune,
  CAST(COALESCE(crg.damage_resistance_radiation_immune, 0) AS INTEGER) AS damage_resistance_radiation_immune,
  CAST(COALESCE(crg.damage_resistance_poison_immune, 0) AS INTEGER) AS damage_resistance_poison_immune
FROM player_characters pc
JOIN players p ON p.id = pc.player_id
LEFT JOIN player_character_defense cdl ON cdl.player_character_id = pc.id
LEFT JOIN player_character_resistance_global_agg crg ON crg.player_character_id = pc.id
LEFT JOIN player_character_resistance_by_location_agg crl ON crl.player_character_id = pc.id
WHERE pc.campaign_id = sqlc.arg(campaign_id) AND pc.active = 1
ORDER BY p.name COLLATE NOCASE ASC, pc.name COLLATE NOCASE ASC;

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
WITH combatant_defense AS (
  SELECT
    cdl.combatant_id,
    MAX(CASE WHEN bl.code = 'head' THEN cdl.defense END) AS defense_head,
    MAX(CASE WHEN bl.code = 'torso' THEN cdl.defense END) AS defense_torso,
    MAX(CASE WHEN bl.code = 'left_arm' THEN cdl.defense END) AS defense_left_arm,
    MAX(CASE WHEN bl.code = 'right_arm' THEN cdl.defense END) AS defense_right_arm,
    MAX(CASE WHEN bl.code = 'left_leg' THEN cdl.defense END) AS defense_left_leg,
    MAX(CASE WHEN bl.code = 'right_leg' THEN cdl.defense END) AS defense_right_leg
  FROM combatant_defense_by_location cdl
  JOIN body_locations bl ON bl.id = cdl.body_location_id
  GROUP BY cdl.combatant_id
),
combatant_resistance_global_agg AS (
  SELECT
    crg.combatant_id,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.resistance END) AS damage_resistance_physical,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.resistance END) AS damage_resistance_energy,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.resistance END) AS damage_resistance_radiation,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.resistance END) AS damage_resistance_poison,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.immune END) AS damage_resistance_physical_immune,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.immune END) AS damage_resistance_energy_immune,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.immune END) AS damage_resistance_radiation_immune,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.immune END) AS damage_resistance_poison_immune
  FROM combatant_resistance_global crg
  JOIN damage_types dt ON dt.id = crg.damage_type_id
  GROUP BY crg.combatant_id
),
combatant_resistance_by_location_agg AS (
  SELECT
    crl.combatant_id,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_physical_head,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_physical_torso,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_physical_left_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_physical_right_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_physical_left_leg,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_physical_right_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_energy_head,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_energy_torso,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_energy_left_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_energy_right_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_energy_left_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_energy_right_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_radiation_head,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_radiation_torso,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_radiation_left_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_radiation_right_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_radiation_left_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_radiation_right_leg
  FROM combatant_resistance_by_location crl
  JOIN damage_types dt ON dt.id = crl.damage_type_id
  JOIN body_locations bl ON bl.id = crl.body_location_id
  GROUP BY crl.combatant_id
)
SELECT
  c.id,
  c.name,
  c.side,
  c.level,
  c.xp,
  c.initiative,
  c.hp,
  c.max_hp,
  c.defense,
  c.torso_only,
  CAST(COALESCE(cdl.defense_head, 0) AS INTEGER) AS defense_head,
  CAST(COALESCE(cdl.defense_torso, 0) AS INTEGER) AS defense_torso,
  CAST(COALESCE(cdl.defense_left_arm, 0) AS INTEGER) AS defense_left_arm,
  CAST(COALESCE(cdl.defense_right_arm, 0) AS INTEGER) AS defense_right_arm,
  CAST(COALESCE(cdl.defense_left_leg, 0) AS INTEGER) AS defense_left_leg,
  CAST(COALESCE(cdl.defense_right_leg, 0) AS INTEGER) AS defense_right_leg,
  CAST(COALESCE(crl.damage_resistance_physical_head, 0) AS INTEGER) AS damage_resistance_physical_head,
  CAST(COALESCE(crl.damage_resistance_physical_torso, 0) AS INTEGER) AS damage_resistance_physical_torso,
  CAST(COALESCE(crl.damage_resistance_physical_left_arm, 0) AS INTEGER) AS damage_resistance_physical_left_arm,
  CAST(COALESCE(crl.damage_resistance_physical_right_arm, 0) AS INTEGER) AS damage_resistance_physical_right_arm,
  CAST(COALESCE(crl.damage_resistance_physical_left_leg, 0) AS INTEGER) AS damage_resistance_physical_left_leg,
  CAST(COALESCE(crl.damage_resistance_physical_right_leg, 0) AS INTEGER) AS damage_resistance_physical_right_leg,
  CAST(COALESCE(crg.damage_resistance_physical, 0) AS INTEGER) AS damage_resistance_physical,
  CAST(COALESCE(crg.damage_resistance_energy, 0) AS INTEGER) AS damage_resistance_energy,
  CAST(COALESCE(crg.damage_resistance_radiation, 0) AS INTEGER) AS damage_resistance_radiation,
  CAST(COALESCE(crg.damage_resistance_poison, 0) AS INTEGER) AS damage_resistance_poison,
  CAST(COALESCE(crl.damage_resistance_energy_head, 0) AS INTEGER) AS damage_resistance_energy_head,
  CAST(COALESCE(crl.damage_resistance_energy_torso, 0) AS INTEGER) AS damage_resistance_energy_torso,
  CAST(COALESCE(crl.damage_resistance_energy_left_arm, 0) AS INTEGER) AS damage_resistance_energy_left_arm,
  CAST(COALESCE(crl.damage_resistance_energy_right_arm, 0) AS INTEGER) AS damage_resistance_energy_right_arm,
  CAST(COALESCE(crl.damage_resistance_energy_left_leg, 0) AS INTEGER) AS damage_resistance_energy_left_leg,
  CAST(COALESCE(crl.damage_resistance_energy_right_leg, 0) AS INTEGER) AS damage_resistance_energy_right_leg,
  CAST(COALESCE(crl.damage_resistance_radiation_head, 0) AS INTEGER) AS damage_resistance_radiation_head,
  CAST(COALESCE(crl.damage_resistance_radiation_torso, 0) AS INTEGER) AS damage_resistance_radiation_torso,
  CAST(COALESCE(crl.damage_resistance_radiation_left_arm, 0) AS INTEGER) AS damage_resistance_radiation_left_arm,
  CAST(COALESCE(crl.damage_resistance_radiation_right_arm, 0) AS INTEGER) AS damage_resistance_radiation_right_arm,
  CAST(COALESCE(crl.damage_resistance_radiation_left_leg, 0) AS INTEGER) AS damage_resistance_radiation_left_leg,
  CAST(COALESCE(crl.damage_resistance_radiation_right_leg, 0) AS INTEGER) AS damage_resistance_radiation_right_leg,
  CAST(COALESCE(crg.damage_resistance_physical_immune, 0) AS INTEGER) AS damage_resistance_physical_immune,
  CAST(COALESCE(crg.damage_resistance_energy_immune, 0) AS INTEGER) AS damage_resistance_energy_immune,
  CAST(COALESCE(crg.damage_resistance_radiation_immune, 0) AS INTEGER) AS damage_resistance_radiation_immune,
  CAST(COALESCE(crg.damage_resistance_poison_immune, 0) AS INTEGER) AS damage_resistance_poison_immune,
  c.active,
  c.defeated
FROM combatants c
LEFT JOIN combatant_defense cdl ON cdl.combatant_id = c.id
LEFT JOIN combatant_resistance_global_agg crg ON crg.combatant_id = c.id
LEFT JOIN combatant_resistance_by_location_agg crl ON crl.combatant_id = c.id
WHERE c.encounter_id = sqlc.arg(encounter_id)
ORDER BY c.position ASC;

-- name: UpsertEncounter :exec
INSERT INTO encounters (
  id, campaign_id, name, round, turn_index, party_ap, gm_threat,
  difficulty_label, difficulty_score,
  party_count, party_avg_level, party_xp_budget,
  enemy_count, enemy_avg_level, enemy_total_xp,
  updated_at, deleted_at
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
	id, encounter_id, name, side, torso_only, level, xp, initiative, hp, max_hp, defense, active, defeated, position
)
VALUES (
  sqlc.arg(id),
  sqlc.arg(encounter_id),
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
	  sqlc.arg(position)
	);

-- name: UpsertCombatantDefenseByLocation :exec
INSERT INTO combatant_defense_by_location (
  combatant_id,
  body_location_id,
  defense,
  updated_at
)
VALUES (
  sqlc.arg(combatant_id),
  sqlc.arg(body_location_id),
  sqlc.arg(defense),
  STRFTIME('%Y-%m-%d %H:%M:%f', 'now')
)
ON CONFLICT (combatant_id, body_location_id) DO UPDATE SET
  defense = excluded.defense,
  updated_at = STRFTIME('%Y-%m-%d %H:%M:%f', 'now');

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
WITH combatant_defense AS (
  SELECT
    cdl.combatant_id,
    MAX(CASE WHEN bl.code = 'head' THEN cdl.defense END) AS defense_head,
    MAX(CASE WHEN bl.code = 'torso' THEN cdl.defense END) AS defense_torso,
    MAX(CASE WHEN bl.code = 'left_arm' THEN cdl.defense END) AS defense_left_arm,
    MAX(CASE WHEN bl.code = 'right_arm' THEN cdl.defense END) AS defense_right_arm,
    MAX(CASE WHEN bl.code = 'left_leg' THEN cdl.defense END) AS defense_left_leg,
    MAX(CASE WHEN bl.code = 'right_leg' THEN cdl.defense END) AS defense_right_leg
  FROM combatant_defense_by_location cdl
  JOIN body_locations bl ON bl.id = cdl.body_location_id
  GROUP BY cdl.combatant_id
),
combatant_resistance_global_agg AS (
  SELECT
    crg.combatant_id,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.resistance END) AS damage_resistance_physical,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.resistance END) AS damage_resistance_energy,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.resistance END) AS damage_resistance_radiation,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.resistance END) AS damage_resistance_poison,
    MAX(CASE WHEN dt.code = 'physical' THEN crg.immune END) AS damage_resistance_physical_immune,
    MAX(CASE WHEN dt.code = 'energy' THEN crg.immune END) AS damage_resistance_energy_immune,
    MAX(CASE WHEN dt.code = 'radiation' THEN crg.immune END) AS damage_resistance_radiation_immune,
    MAX(CASE WHEN dt.code = 'poison' THEN crg.immune END) AS damage_resistance_poison_immune
  FROM combatant_resistance_global crg
  JOIN damage_types dt ON dt.id = crg.damage_type_id
  GROUP BY crg.combatant_id
),
combatant_resistance_by_location_agg AS (
  SELECT
    crl.combatant_id,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_physical_head,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_physical_torso,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_physical_left_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_physical_right_arm,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_physical_left_leg,
    MAX(CASE WHEN dt.code = 'physical' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_physical_right_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_energy_head,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_energy_torso,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_energy_left_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_energy_right_arm,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_energy_left_leg,
    MAX(CASE WHEN dt.code = 'energy' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_energy_right_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'head' THEN crl.resistance END) AS damage_resistance_radiation_head,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'torso' THEN crl.resistance END) AS damage_resistance_radiation_torso,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_arm' THEN crl.resistance END) AS damage_resistance_radiation_left_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_arm' THEN crl.resistance END) AS damage_resistance_radiation_right_arm,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'left_leg' THEN crl.resistance END) AS damage_resistance_radiation_left_leg,
    MAX(CASE WHEN dt.code = 'radiation' AND bl.code = 'right_leg' THEN crl.resistance END) AS damage_resistance_radiation_right_leg
  FROM combatant_resistance_by_location crl
  JOIN damage_types dt ON dt.id = crl.damage_type_id
  JOIN body_locations bl ON bl.id = crl.body_location_id
  GROUP BY crl.combatant_id
),
latest_party AS (
  SELECT
    c.name,
    c.level,
    c.xp,
    c.initiative,
    c.hp,
    c.max_hp,
    c.defense,
    c.torso_only,
    CAST(COALESCE(cdl.defense_head, 0) AS INTEGER) AS defense_head,
    CAST(COALESCE(cdl.defense_torso, 0) AS INTEGER) AS defense_torso,
    CAST(COALESCE(cdl.defense_left_arm, 0) AS INTEGER) AS defense_left_arm,
    CAST(COALESCE(cdl.defense_right_arm, 0) AS INTEGER) AS defense_right_arm,
    CAST(COALESCE(cdl.defense_left_leg, 0) AS INTEGER) AS defense_left_leg,
    CAST(COALESCE(cdl.defense_right_leg, 0) AS INTEGER) AS defense_right_leg,
    CAST(COALESCE(crl.damage_resistance_physical_head, 0) AS INTEGER) AS damage_resistance_physical_head,
    CAST(COALESCE(crl.damage_resistance_physical_torso, 0) AS INTEGER) AS damage_resistance_physical_torso,
    CAST(COALESCE(crl.damage_resistance_physical_left_arm, 0) AS INTEGER) AS damage_resistance_physical_left_arm,
    CAST(COALESCE(crl.damage_resistance_physical_right_arm, 0) AS INTEGER) AS damage_resistance_physical_right_arm,
    CAST(COALESCE(crl.damage_resistance_physical_left_leg, 0) AS INTEGER) AS damage_resistance_physical_left_leg,
    CAST(COALESCE(crl.damage_resistance_physical_right_leg, 0) AS INTEGER) AS damage_resistance_physical_right_leg,
    CAST(COALESCE(crg.damage_resistance_physical, 0) AS INTEGER) AS damage_resistance_physical,
    CAST(COALESCE(crg.damage_resistance_energy, 0) AS INTEGER) AS damage_resistance_energy,
    CAST(COALESCE(crg.damage_resistance_radiation, 0) AS INTEGER) AS damage_resistance_radiation,
    CAST(COALESCE(crg.damage_resistance_poison, 0) AS INTEGER) AS damage_resistance_poison,
    CAST(COALESCE(crl.damage_resistance_energy_head, 0) AS INTEGER) AS damage_resistance_energy_head,
    CAST(COALESCE(crl.damage_resistance_energy_torso, 0) AS INTEGER) AS damage_resistance_energy_torso,
    CAST(COALESCE(crl.damage_resistance_energy_left_arm, 0) AS INTEGER) AS damage_resistance_energy_left_arm,
    CAST(COALESCE(crl.damage_resistance_energy_right_arm, 0) AS INTEGER) AS damage_resistance_energy_right_arm,
    CAST(COALESCE(crl.damage_resistance_energy_left_leg, 0) AS INTEGER) AS damage_resistance_energy_left_leg,
    CAST(COALESCE(crl.damage_resistance_energy_right_leg, 0) AS INTEGER) AS damage_resistance_energy_right_leg,
    CAST(COALESCE(crl.damage_resistance_radiation_head, 0) AS INTEGER) AS damage_resistance_radiation_head,
    CAST(COALESCE(crl.damage_resistance_radiation_torso, 0) AS INTEGER) AS damage_resistance_radiation_torso,
    CAST(COALESCE(crl.damage_resistance_radiation_left_arm, 0) AS INTEGER) AS damage_resistance_radiation_left_arm,
    CAST(COALESCE(crl.damage_resistance_radiation_right_arm, 0) AS INTEGER) AS damage_resistance_radiation_right_arm,
    CAST(COALESCE(crl.damage_resistance_radiation_left_leg, 0) AS INTEGER) AS damage_resistance_radiation_left_leg,
    CAST(COALESCE(crl.damage_resistance_radiation_right_leg, 0) AS INTEGER) AS damage_resistance_radiation_right_leg,
    CAST(COALESCE(crg.damage_resistance_physical_immune, 0) AS INTEGER) AS damage_resistance_physical_immune,
    CAST(COALESCE(crg.damage_resistance_energy_immune, 0) AS INTEGER) AS damage_resistance_energy_immune,
    CAST(COALESCE(crg.damage_resistance_radiation_immune, 0) AS INTEGER) AS damage_resistance_radiation_immune,
    CAST(COALESCE(crg.damage_resistance_poison_immune, 0) AS INTEGER) AS damage_resistance_poison_immune,
    ROW_NUMBER() OVER (
      PARTITION BY LOWER(TRIM(c.name))
      ORDER BY e.updated_at DESC, e.id DESC, c.position ASC
    ) AS rn
  FROM combatants c
  JOIN encounters e ON e.id = c.encounter_id
  LEFT JOIN combatant_defense cdl ON cdl.combatant_id = c.id
  LEFT JOIN combatant_resistance_global_agg crg ON crg.combatant_id = c.id
  LEFT JOIN combatant_resistance_by_location_agg crl ON crl.combatant_id = c.id
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
  max_hp,
  defense,
  torso_only,
  defense_head,
  defense_torso,
  defense_left_arm,
  defense_right_arm,
  defense_left_leg,
  defense_right_leg,
  damage_resistance_physical_head,
  damage_resistance_physical_torso,
  damage_resistance_physical_left_arm,
  damage_resistance_physical_right_arm,
  damage_resistance_physical_left_leg,
  damage_resistance_physical_right_leg,
  damage_resistance_physical,
  damage_resistance_energy,
  damage_resistance_radiation,
  damage_resistance_poison,
  damage_resistance_energy_head,
  damage_resistance_energy_torso,
  damage_resistance_energy_left_arm,
  damage_resistance_energy_right_arm,
  damage_resistance_energy_left_leg,
  damage_resistance_energy_right_leg,
  damage_resistance_radiation_head,
  damage_resistance_radiation_torso,
  damage_resistance_radiation_left_arm,
  damage_resistance_radiation_right_arm,
  damage_resistance_radiation_left_leg,
  damage_resistance_radiation_right_leg,
  damage_resistance_physical_immune,
  damage_resistance_energy_immune,
  damage_resistance_radiation_immune,
  damage_resistance_poison_immune
FROM latest_party
WHERE rn = 1
ORDER BY name COLLATE NOCASE ASC;
