package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLCSchemaSeedsEnumDictionaries(t *testing.T) {
	schema, err := os.ReadFile("sqlc/schema.sql")
	require.NoError(t, err)

	db, err := sql.Open("sqlite", sqliteDSN(filepath.Join(t.TempDir(), "schema.db")))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	_, err = db.Exec(string(schema))
	require.NoError(t, err)

	assert.Equal(
		t,
		"0:global,1:head,2:torso,3:left_arm,4:right_arm,5:left_leg,6:right_leg",
		queryString(t, db, `SELECT GROUP_CONCAT(id || ':' || code, ',') FROM (SELECT id, code FROM body_locations ORDER BY id)`),
	)
	assert.Equal(
		t,
		"1:physical,2:energy,3:radiation,4:poison",
		queryString(t, db, `SELECT GROUP_CONCAT(id || ':' || code, ',') FROM (SELECT id, code FROM damage_types ORDER BY id)`),
	)
}

func TestOpenAndMigrateAddsPlayerCharacterDetails(t *testing.T) {
	store := newTestStore(t)

	assert.Contains(t, queryColumnNames(t, store.db, "player_characters"), "notes")
	assert.Equal(t, "", queryString(t, store.db, `SELECT notes FROM player_characters WHERE id = ?`, "repo-char-1"))
	assert.Equal(t, int64(7), queryInt64(t, store.db, `SELECT COUNT(*) FROM special_attributes`))
	assert.Equal(t, int64(7), queryInt64(t, store.db, `SELECT COUNT(*) FROM player_character_special_attributes WHERE player_character_id = ?`, "repo-char-1"))
	assert.Equal(t, int64(7), queryInt64(t, store.db, `SELECT COUNT(*) FROM player_character_special_attributes WHERE player_character_id = ? AND value = 1`, "repo-char-1"))

	_, err := store.db.Exec(`UPDATE player_character_special_attributes SET value = 0 WHERE player_character_id = ?`, "repo-char-1")
	require.Error(t, err)
}

func TestOpenAndMigrateEnablesForeignKeysAndCascadeOnAllConnections(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repo-fk-cascade.db")
	db, err := OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	store := NewEncounterStore(db)
	_, err = store.CreateCampaign(t.Context(), "fk-campaign", "FK Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player A",
			Character: domain.Combatant{
				ID:         "fk-char-1",
				Name:       "Scout",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 7,
				HP:         6,
				MaxHP:      6,
			},
		},
	})
	require.NoError(t, err)
	assert.NotContains(t, queryColumnNames(t, store.db, "player_characters"), "campaign_id")
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "fk-encounter",
		Name: "Cascade Check",
		Combatants: []domain.Combatant{{
			ID:                  "fk-npc-1",
			Name:                "Raider",
			Side:                domain.SideNPC,
			Initiative:          8,
			HP:                  6,
			MaxHP:               6,
			ResistPoison:        2,
			ResistPhysicalTorso: 1,
		}},
	}))
	assert.Equal(t, int64(1), queryInt64(t, db, `SELECT COUNT(*) FROM combatants WHERE encounter_id = ?`, "fk-encounter"))
	assert.Equal(t, int64(1), queryInt64(t, db, `
		SELECT COUNT(*)
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')
	`, statProfileID(statProfileCombatantKind, "fk-npc-1")))

	conn1, err := db.Conn(t.Context())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn1.Close())
	}()
	conn2, err := db.Conn(t.Context())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn2.Close())
	}()

	var foreignKeysConn1, foreignKeysConn2 int64
	require.NoError(t, conn1.QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&foreignKeysConn1))
	require.NoError(t, conn2.QueryRowContext(t.Context(), `PRAGMA foreign_keys`).Scan(&foreignKeysConn2))
	assert.Equal(t, int64(1), foreignKeysConn1)
	assert.Equal(t, int64(1), foreignKeysConn2)

	_, err = conn2.ExecContext(t.Context(), `DELETE FROM encounters WHERE id = ?`, "fk-encounter")
	require.NoError(t, err)

	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM combatants WHERE encounter_id = ?`, "fk-encounter"))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM stat_profiles WHERE id = ?`, statProfileID(statProfileCombatantKind, "fk-npc-1")))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM stat_profile_resistance_by_location WHERE stat_profile_id = ?`, statProfileID(statProfileCombatantKind, "fk-npc-1")))
}

func TestOpenAndMigrateCreatesCriticalEncounterIndexes(t *testing.T) {
	store := newTestStore(t)

	assertIndexColumns(t, store.db, "idx_encounters_campaign_deleted_updated", []indexColumn{
		{name: "campaign_id"},
		{name: "deleted_at"},
		{name: "updated_at", desc: true},
		{name: "id", desc: true},
	})
}

func TestOpenAndMigrateEnforcesCampaignRelationships(t *testing.T) {
	store := newTestStore(t)

	_, err := store.CreateCampaign(t.Context(), "other-campaign", "Other Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Other Player",
			Character: domain.Combatant{
				ID:         "other-char",
				Name:       "Other Character",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 5,
				HP:         5,
				MaxHP:      5,
			},
		},
	})
	require.NoError(t, err)

	_, err = store.db.Exec(
		`INSERT INTO encounters (id, campaign_id, name, round, turn_index)
         VALUES (?, ?, ?, ?, ?)`,
		"null-campaign-encounter",
		nil,
		"Null Campaign",
		1,
		0,
	)
	require.Error(t, err)

	_, err = store.db.Exec(
		`INSERT INTO encounters (id, campaign_id, name, round, turn_index)
         VALUES (?, ?, ?, ?, ?)`,
		"missing-campaign-encounter",
		"missing-campaign",
		"Missing Campaign",
		1,
		0,
	)
	require.Error(t, err)

	_, err = store.db.Exec(
		`INSERT INTO encounters (id, campaign_id, name, round, turn_index)
         VALUES (?, ?, ?, ?, ?)`,
		"campaign-rel-encounter",
		"repo-test-campaign",
		"Relationship Check",
		1,
		0,
	)
	require.NoError(t, err)
	_, err = store.db.Exec(`INSERT INTO stat_profiles (id) VALUES (?)`, statProfileID(statProfileCombatantKind, "cross-campaign-combatant"))
	require.NoError(t, err)
	_, err = store.db.Exec(
		`INSERT INTO combatants (
            id, encounter_id, stat_profile_id, player_character_id, name, side, position
         ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"cross-campaign-combatant",
		"campaign-rel-encounter",
		statProfileID(statProfileCombatantKind, "cross-campaign-combatant"),
		"other-char",
		"Cross Campaign Character",
		string(domain.SideParty),
		0,
	)
	require.Error(t, err)
}

func TestOpenAndMigrateEnforcesCoreRelationshipInvariants(t *testing.T) {
	store := newTestStore(t)

	_, err := store.CreateCampaign(t.Context(), "relationship-other-campaign", "Relationship Other Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Other Player",
			Character: domain.Combatant{
				ID:         "relationship-other-char",
				Name:       "Other Character",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 5,
				HP:         5,
				MaxHP:      5,
			},
		},
	})
	require.NoError(t, err)

	playerID := queryString(t, store.db, `SELECT id FROM players WHERE campaign_id = ? AND name = ?`, "repo-test-campaign", "Player 1")
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "relationship-encounter",
		Name: "Relationship Invariants",
		Combatants: []domain.Combatant{{
			ID:         "relationship-combatant",
			Name:       "Raider",
			Side:       domain.SideNPC,
			Initiative: 8,
			HP:         6,
			MaxHP:      6,
		}},
	}))
	_, err = store.db.Exec(`INSERT INTO stat_profiles (id) VALUES (?)`, statProfileID(statProfileCombatantKind, "relationship-linked-combatant"))
	require.NoError(t, err)
	_, err = store.db.Exec(
		`INSERT INTO combatants (
            id, encounter_id, stat_profile_id, player_character_id, name, side, position
         ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"relationship-linked-combatant",
		"relationship-encounter",
		statProfileID(statProfileCombatantKind, "relationship-linked-combatant"),
		"repo-char-1",
		"Linked Character",
		string(domain.SideParty),
		1,
	)
	require.NoError(t, err)

	t.Run("player character requires existing player", func(t *testing.T) {
		_, err := store.db.Exec(`INSERT INTO stat_profiles (id) VALUES (?)`, statProfileID(statProfilePlayerCharacterKind, "relationship-missing-player-char"))
		require.NoError(t, err)

		_, err = store.db.Exec(
			`INSERT INTO player_characters (
                id, player_id, stat_profile_id, name, active, availability_status
             ) VALUES (?, ?, ?, ?, ?, ?)`,
			"relationship-missing-player-char",
			"missing-player",
			statProfileID(statProfilePlayerCharacterKind, "relationship-missing-player-char"),
			"Missing Player Character",
			0,
			playerCharacterAvailabilityActive,
		)
		require.Error(t, err)
	})

	t.Run("player campaign update keeps linked combatants consistent", func(t *testing.T) {
		_, err := store.db.Exec(`UPDATE players SET campaign_id = ? WHERE id = ?`, "relationship-other-campaign", playerID)
		require.Error(t, err)
	})

	t.Run("player has at most one active character", func(t *testing.T) {
		_, err := store.db.Exec(`INSERT INTO stat_profiles (id) VALUES (?)`, statProfileID(statProfilePlayerCharacterKind, "relationship-second-active-char"))
		require.NoError(t, err)

		_, err = store.db.Exec(
			`INSERT INTO player_characters (
                id, player_id, stat_profile_id, name, active, availability_status
             ) VALUES (?, ?, ?, ?, ?, ?)`,
			"relationship-second-active-char",
			playerID,
			statProfileID(statProfilePlayerCharacterKind, "relationship-second-active-char"),
			"Second Active Character",
			1,
			playerCharacterAvailabilityActive,
		)
		require.Error(t, err)
	})

	t.Run("combatant requires existing encounter", func(t *testing.T) {
		_, err := store.db.Exec(`INSERT INTO stat_profiles (id) VALUES (?)`, statProfileID(statProfileCombatantKind, "relationship-missing-encounter-combatant"))
		require.NoError(t, err)

		_, err = store.db.Exec(
			`INSERT INTO combatants (
                id, encounter_id, stat_profile_id, name, side, position
             ) VALUES (?, ?, ?, ?, ?, ?)`,
			"relationship-missing-encounter-combatant",
			"missing-encounter",
			statProfileID(statProfileCombatantKind, "relationship-missing-encounter-combatant"),
			"Missing Encounter Combatant",
			string(domain.SideNPC),
			0,
		)
		require.Error(t, err)
	})

	t.Run("combatant requires existing stat profile", func(t *testing.T) {
		_, err := store.db.Exec(
			`INSERT INTO combatants (
                id, encounter_id, stat_profile_id, name, side, position
             ) VALUES (?, ?, ?, ?, ?, ?)`,
			"relationship-missing-profile-combatant",
			"relationship-encounter",
			statProfileID(statProfileCombatantKind, "relationship-missing-profile-combatant"),
			"Missing Profile Combatant",
			string(domain.SideNPC),
			1,
		)
		require.Error(t, err)
	})
}

func TestOpenAndMigrateDerivesActiveCombatantFromTurnIndex(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:        "single-active",
		Name:      "Single Active",
		TurnIndex: 1,
		Combatants: []domain.Combatant{
			{
				ID:         "active-1",
				Name:       "Active One",
				Side:       domain.SideNPC,
				Initiative: 10,
				HP:         6,
				MaxHP:      6,
				Active:     true,
			},
			{
				ID:         "inactive-1",
				Name:       "Inactive One",
				Side:       domain.SideNPC,
				Initiative: 9,
				HP:         5,
				MaxHP:      5,
				Active:     false,
			},
		},
	}))

	assert.NotContains(t, queryColumnNames(t, store.db, "combatants"), "active")

	encounter, err := store.GetEncounterByID(t.Context(), "single-active")
	require.NoError(t, err)
	require.Len(t, encounter.Combatants, 2)
	assert.False(t, encounter.Combatants[0].Active)
	assert.True(t, encounter.Combatants[1].Active)
}

func TestOpenAndMigrateCascadesOwnedStatProfiles(t *testing.T) {
	store := newTestStore(t)

	playerCharacterProfileID := statProfileID(statProfilePlayerCharacterKind, "repo-char-1")
	_, err := store.db.Exec(`DELETE FROM stat_profiles WHERE id = ?`, playerCharacterProfileID)
	require.Error(t, err)

	_, err = store.db.Exec(`DELETE FROM player_characters WHERE id = ?`, "repo-char-1")
	require.NoError(t, err)
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profiles WHERE id = ?`, playerCharacterProfileID))
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profile_resistance_by_location WHERE stat_profile_id = ?`, playerCharacterProfileID))

	monster, err := store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		ID:                  "cascade-monster",
		Name:                "Cascade Monster",
		Level:               1,
		XP:                  20,
		Initiative:          4,
		HP:                  5,
		MaxHP:               5,
		ResistPoison:        2,
		ResistPhysicalTorso: 1,
	})
	require.NoError(t, err)

	monsterProfileID := statProfileID(statProfileMonsterTemplateKind, monster.ID)
	_, err = store.db.Exec(`DELETE FROM stat_profiles WHERE id = ?`, monsterProfileID)
	require.Error(t, err)

	_, err = store.db.Exec(`DELETE FROM monster_templates WHERE id = ?`, monster.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profiles WHERE id = ?`, monsterProfileID))
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profile_resistance_by_location WHERE stat_profile_id = ?`, monsterProfileID))
}

func TestMigration35BackfillsNullEncounterCampaigns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-35-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 34))

	_, err = db.Exec(
		`INSERT INTO encounters (id, campaign_id, name, round, turn_index)
         VALUES (?, ?, ?, ?, ?)`,
		"legacy-null-campaign",
		nil,
		"Legacy Null Campaign",
		1,
		0,
	)
	require.NoError(t, err)

	require.NoError(t, goose.Up(db, "migrations"))
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", queryString(t, db, `SELECT campaign_id FROM encounters WHERE id = ?`, "legacy-null-campaign"))

	_, err = db.Exec(
		`INSERT INTO encounters (id, campaign_id, name, round, turn_index)
         VALUES (?, ?, ?, ?, ?)`,
		"still-null-campaign",
		nil,
		"Still Null Campaign",
		1,
		0,
	)
	require.Error(t, err)
}

func TestMigration36BackfillsDuplicateActiveCombatants(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-36-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 35))

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, start_date)
		VALUES ('active-campaign', 'Active Campaign', '2026-01-01 00:00:00');
		INSERT OR IGNORE INTO app_state (id, active_campaign_id)
		VALUES (1, 'active-campaign');
		INSERT INTO encounters (id, campaign_id, name, round, turn_index)
		VALUES ('legacy-active-encounter', 'active-campaign', 'Legacy Active', 1, 1);
		INSERT INTO stat_profiles (id)
		VALUES ('combatant:active-a'), ('combatant:active-b');
		INSERT INTO combatants (
			id, encounter_id, stat_profile_id, name, side, active, defeated, position
		) VALUES
			('active-a', 'legacy-active-encounter', 'combatant:active-a', 'Active A', 'npc', 1, 0, 0),
			('active-b', 'legacy-active-encounter', 'combatant:active-b', 'Active B', 'npc', 1, 0, 1);
	`)
	require.NoError(t, err)

	require.NoError(t, goose.Up(db, "migrations"))

	assert.NotContains(t, queryColumnNames(t, db, "combatants"), "active")
	assert.Equal(t, "active-b", queryString(t, db, `
		SELECT c.id
		FROM combatants c
		JOIN encounters e ON e.id = c.encounter_id
		WHERE c.encounter_id = ? AND c.position = e.turn_index
	`, "legacy-active-encounter"))
}

func TestMigration39MovesGlobalResistanceIntoLocationRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-39-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 38))

	poisonID := queryInt64(t, db, `SELECT id FROM damage_types WHERE code = 'poison'`)
	physicalID := queryInt64(t, db, `SELECT id FROM damage_types WHERE code = 'physical'`)
	energyID := queryInt64(t, db, `SELECT id FROM damage_types WHERE code = 'energy'`)
	headID := queryInt64(t, db, `SELECT id FROM body_locations WHERE code = 'head'`)

	_, err = db.Exec(`
		INSERT INTO stat_profiles (id)
		VALUES ('migration-39-profile')
	`)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO stat_profile_resistance_global (
			stat_profile_id, damage_type_id, resistance, immune
		) VALUES
			('migration-39-profile', ?, 7, 1),
			('migration-39-profile', ?, 0, 1)
	`, poisonID, physicalID)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO stat_profile_resistance_by_location (
			stat_profile_id, damage_type_id, body_location_id, resistance
		) VALUES
			('migration-39-profile', ?, ?, 3),
			('migration-39-profile', ?, ?, 9)
	`, energyID, headID, physicalID, headID)
	require.NoError(t, err)

	require.NoError(t, goose.Up(db, "migrations"))

	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name = 'stat_profile_resistance_global'`))
	assert.Contains(t, queryColumnNames(t, db, "stat_profile_resistance_by_location"), "immune")
	globalID := queryInt64(t, db, `SELECT id FROM body_locations WHERE code = 'global'`)
	assert.Equal(t, int64(7), queryInt64(t, db, `
		SELECT resistance
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND damage_type_id = ?
		  AND body_location_id = ?
	`, "migration-39-profile", poisonID, globalID))
	assert.Equal(t, int64(1), queryInt64(t, db, `
		SELECT immune
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND damage_type_id = ?
		  AND body_location_id = ?
	`, "migration-39-profile", physicalID, globalID))
	assert.Equal(t, int64(0), queryInt64(t, db, `
		SELECT COUNT(*)
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND damage_type_id = ?
		  AND body_location_id <> ?
	`, "migration-39-profile", physicalID, globalID))
	assert.Equal(t, int64(3), queryInt64(t, db, `
		SELECT resistance
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND damage_type_id = ?
		  AND body_location_id = ?
		  AND immune = 0
	`, "migration-39-profile", energyID, headID))
}

func TestMigration33BackfillsStatProfilesAndAddsResistanceCompatibilityViews(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-32-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 31))

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, start_date) VALUES ('legacy-campaign', 'Legacy Campaign', '2026-01-01 00:00:00');
		INSERT OR IGNORE INTO app_state (id, active_campaign_id) VALUES (1, 'legacy-campaign');
		INSERT INTO players (id, campaign_id, name) VALUES ('legacy-player', 'legacy-campaign', 'Legacy Player');
		INSERT INTO player_characters (
			id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense, torso_only, active, availability_status
		) VALUES (
			'legacy-character', 'legacy-player', 'legacy-campaign', 'Legacy Character', 2, 7, 6, 8, 1, 0, 1, 'active'
		);
		INSERT INTO encounters (
			id, campaign_id, name, round, turn_index, party_ap, gm_threat,
			difficulty_label, difficulty_score, party_count, party_avg_level, party_xp_budget,
			enemy_count, enemy_avg_level, enemy_total_xp
		) VALUES (
			'legacy-encounter', 'legacy-campaign', 'Legacy Encounter', 1, 0, 0, 0,
			'Unknown', 0, 0, 0, 0, 0, 0, 0
		);
		INSERT INTO combatants (
			id, encounter_id, player_character_id, name, side, torso_only, initiative,
			active, defeated, position, hp, max_hp, defense, level, xp
		) VALUES (
			'legacy-combatant', 'legacy-encounter', NULL, 'Legacy Raider', 'npc', 1, 5,
			1, 0, 0, 4, 6, 2, 3, 25
		);
		INSERT INTO monster_templates (
			id, name, name_key, torso_only, level, xp, initiative, hp, max_hp, defense
		) VALUES (
			'legacy-monster', 'Legacy Monster', 'legacy monster', 0, 4, 30, 4, 3, 5, 3
		);
	`)
	require.NoError(t, err)

	poisonID := queryInt64(t, db, `SELECT id FROM damage_types WHERE code = 'poison'`)
	energyID := queryInt64(t, db, `SELECT id FROM damage_types WHERE code = 'energy'`)
	headID := queryInt64(t, db, `SELECT id FROM body_locations WHERE code = 'head'`)
	_, err = db.Exec(
		`INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune)
         VALUES (?, ?, ?, ?)`,
		"legacy-combatant",
		poisonID,
		3,
		1,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO player_character_resistance_by_location (player_character_id, damage_type_id, body_location_id, resistance)
         VALUES (?, ?, ?, ?)`,
		"legacy-character",
		energyID,
		headID,
		4,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO monster_template_resistance_global (monster_template_id, damage_type_id, resistance, immune)
         VALUES (?, ?, ?, ?)`,
		"legacy-monster",
		poisonID,
		5,
		0,
	)
	require.NoError(t, err)

	require.NoError(t, goose.Up(db, "migrations"))

	assert.Equal(t, int64(3), queryInt64(t, db, `SELECT level FROM stat_profiles WHERE id = ?`, statProfileID(statProfileCombatantKind, "legacy-combatant")))
	assert.Equal(t, int64(25), queryInt64(t, db, `SELECT xp FROM stat_profiles WHERE id = ?`, statProfileID(statProfileCombatantKind, "legacy-combatant")))
	assert.Equal(t, int64(6), queryInt64(t, db, `SELECT hp FROM stat_profiles WHERE id = ?`, statProfileID(statProfilePlayerCharacterKind, "legacy-character")))
	assert.Equal(t, int64(30), queryInt64(t, db, `SELECT xp FROM stat_profiles WHERE id = ?`, statProfileID(statProfileMonsterTemplateKind, "legacy-monster")))
	assert.Equal(t, int64(3), queryInt64(t, db, `
		SELECT resistance
		FROM stat_profile_resistance_by_location
		WHERE stat_profile_id = ?
		  AND damage_type_id = ?
		  AND body_location_id = (SELECT id FROM body_locations WHERE code = 'global')
	`, statProfileID(statProfileCombatantKind, "legacy-combatant"), poisonID))
	assert.Equal(t, int64(4), queryInt64(t, db, `SELECT resistance FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, statProfileID(statProfilePlayerCharacterKind, "legacy-character"), energyID, headID))
	assert.Equal(t, int64(5), queryInt64(t, db, `SELECT resistance FROM monster_template_resistance_global WHERE monster_template_id = ? AND damage_type_id = ?`, "legacy-monster", poisonID))
	assert.Equal(t, "view", queryString(t, db, `SELECT type FROM sqlite_schema WHERE name = ?`, "combatant_resistance_global"))
}

func TestResistanceCompatibilityViewsWriteThroughStatProfiles(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "compat-encounter",
		Name: "Compatibility Views",
		Combatants: []domain.Combatant{{
			ID:         "compat-combatant",
			Name:       "Raider",
			Side:       domain.SideNPC,
			Initiative: 5,
			HP:         5,
			MaxHP:      5,
		}},
	}))
	monster, err := store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		ID:         "compat-monster",
		Name:       "Compat Monster",
		Level:      1,
		Initiative: 4,
		HP:         4,
		MaxHP:      4,
	})
	require.NoError(t, err)

	for _, viewName := range []string{
		"combatant_resistance_global",
		"combatant_resistance_by_location",
		"player_character_resistance_global",
		"player_character_resistance_by_location",
		"monster_template_resistance_global",
		"monster_template_resistance_by_location",
	} {
		assert.Equal(t, "view", queryString(t, store.db, `SELECT type FROM sqlite_schema WHERE name = ?`, viewName))
	}

	poisonID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = 'poison'`)
	energyID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = 'energy'`)
	globalID := queryInt64(t, store.db, `SELECT id FROM body_locations WHERE code = 'global'`)
	headID := queryInt64(t, store.db, `SELECT id FROM body_locations WHERE code = 'head'`)
	owners := []struct {
		globalView   string
		locationView string
		ownerColumn  string
		ownerID      string
		profileID    string
	}{
		{
			globalView:   "combatant_resistance_global",
			locationView: "combatant_resistance_by_location",
			ownerColumn:  "combatant_id",
			ownerID:      "compat-combatant",
			profileID:    statProfileID(statProfileCombatantKind, "compat-combatant"),
		},
		{
			globalView:   "player_character_resistance_global",
			locationView: "player_character_resistance_by_location",
			ownerColumn:  "player_character_id",
			ownerID:      "repo-char-1",
			profileID:    statProfileID(statProfilePlayerCharacterKind, "repo-char-1"),
		},
		{
			globalView:   "monster_template_resistance_global",
			locationView: "monster_template_resistance_by_location",
			ownerColumn:  "monster_template_id",
			ownerID:      monster.ID,
			profileID:    statProfileID(statProfileMonsterTemplateKind, monster.ID),
		},
	}

	for _, owner := range owners {
		t.Run(owner.globalView, func(t *testing.T) {
			_, err := store.db.Exec(
				fmt.Sprintf(`INSERT INTO %s (%s, damage_type_id, resistance, immune) VALUES (?, ?, ?, ?)`, owner.globalView, owner.ownerColumn),
				owner.ownerID,
				poisonID,
				4,
				1,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(4), queryInt64(t, store.db, `SELECT resistance FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, poisonID, globalID))
			assert.Equal(t, int64(1), queryInt64(t, store.db, `SELECT immune FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, poisonID, globalID))

			_, err = store.db.Exec(
				fmt.Sprintf(`UPDATE %s SET resistance = ?, immune = ? WHERE %s = ? AND damage_type_id = ?`, owner.globalView, owner.ownerColumn),
				6,
				0,
				owner.ownerID,
				poisonID,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(6), queryInt64(t, store.db, `SELECT resistance FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, poisonID, globalID))
			assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT immune FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, poisonID, globalID))

			_, err = store.db.Exec(
				fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND damage_type_id = ?`, owner.globalView, owner.ownerColumn),
				owner.ownerID,
				poisonID,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, poisonID, globalID))
		})

		t.Run(owner.locationView, func(t *testing.T) {
			_, err := store.db.Exec(
				fmt.Sprintf(`INSERT INTO %s (%s, damage_type_id, body_location_id, resistance) VALUES (?, ?, ?, ?)`, owner.locationView, owner.ownerColumn),
				owner.ownerID,
				energyID,
				headID,
				3,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(3), queryInt64(t, store.db, `SELECT resistance FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, energyID, headID))

			_, err = store.db.Exec(
				fmt.Sprintf(`UPDATE %s SET resistance = ? WHERE %s = ? AND damage_type_id = ? AND body_location_id = ?`, owner.locationView, owner.ownerColumn),
				8,
				owner.ownerID,
				energyID,
				headID,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(8), queryInt64(t, store.db, `SELECT resistance FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, energyID, headID))

			_, err = store.db.Exec(
				fmt.Sprintf(`DELETE FROM %s WHERE %s = ? AND damage_type_id = ? AND body_location_id = ?`, owner.locationView, owner.ownerColumn),
				owner.ownerID,
				energyID,
				headID,
			)
			require.NoError(t, err)
			assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM stat_profile_resistance_by_location WHERE stat_profile_id = ? AND damage_type_id = ? AND body_location_id = ?`, owner.profileID, energyID, headID))
		})
	}
}

func TestOpenAndMigratePreventsMixedGlobalAndLocationResistanceForSameDamageType(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "exclusive-resistance-encounter",
		Name: "Exclusive Resistance",
		Combatants: []domain.Combatant{{
			ID:         "exclusive-resistance-combatant",
			Name:       "Raider",
			Side:       domain.SideNPC,
			Initiative: 8,
			HP:         6,
			MaxHP:      6,
		}},
	}))

	physicalID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = 'physical'`)
	headID := queryInt64(t, store.db, `SELECT id FROM body_locations WHERE code = 'head'`)
	profileID := statProfileID(statProfileCombatantKind, "exclusive-resistance-combatant")
	_, err := store.db.Exec(
		`DELETE FROM stat_profile_resistance_by_location
	     WHERE stat_profile_id = ?
	       AND damage_type_id = ?`,
		profileID,
		physicalID,
	)
	require.NoError(t, err)

	_, err = store.db.Exec(
		`INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune)
	     VALUES (?, ?, ?, ?)`,
		"exclusive-resistance-combatant",
		physicalID,
		0,
		1,
	)
	require.NoError(t, err)
	_, err = store.db.Exec(
		`INSERT INTO combatant_resistance_by_location (combatant_id, damage_type_id, body_location_id, resistance)
	     VALUES (?, ?, ?, ?)`,
		"exclusive-resistance-combatant",
		physicalID,
		headID,
		3,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "location resistance conflicts with global resistance")

	_, err = store.db.Exec(
		`DELETE FROM combatant_resistance_global
	     WHERE combatant_id = ?
	       AND damage_type_id = ?`,
		"exclusive-resistance-combatant",
		physicalID,
	)
	require.NoError(t, err)
	_, err = store.db.Exec(
		`INSERT INTO combatant_resistance_by_location (combatant_id, damage_type_id, body_location_id, resistance)
	     VALUES (?, ?, ?, ?)`,
		"exclusive-resistance-combatant",
		physicalID,
		headID,
		3,
	)
	require.NoError(t, err)
	_, err = store.db.Exec(
		`INSERT INTO combatant_resistance_global (combatant_id, damage_type_id, resistance, immune)
	     VALUES (?, ?, ?, ?)`,
		"exclusive-resistance-combatant",
		physicalID,
		0,
		1,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "global resistance conflicts with location resistance")
}

func TestOpenAndMigrateDropsLegacyCombatStatsColumnsAndSyncTriggers(t *testing.T) {
	store := newTestStore(t)
	legacyColumns := []string{
		"defense_head",
		"defense_torso",
		"defense_left_arm",
		"defense_right_arm",
		"defense_left_leg",
		"defense_right_leg",
		"damage_resistance_physical_head",
		"damage_resistance_physical_torso",
		"damage_resistance_physical_left_arm",
		"damage_resistance_physical_right_arm",
		"damage_resistance_physical_left_leg",
		"damage_resistance_physical_right_leg",
		"damage_resistance_physical",
		"damage_resistance_energy",
		"damage_resistance_radiation",
		"damage_resistance_poison",
		"damage_resistance_energy_head",
		"damage_resistance_energy_torso",
		"damage_resistance_energy_left_arm",
		"damage_resistance_energy_right_arm",
		"damage_resistance_energy_left_leg",
		"damage_resistance_energy_right_leg",
		"damage_resistance_radiation_head",
		"damage_resistance_radiation_torso",
		"damage_resistance_radiation_left_arm",
		"damage_resistance_radiation_right_arm",
		"damage_resistance_radiation_left_leg",
		"damage_resistance_radiation_right_leg",
		"damage_resistance_physical_immune",
		"damage_resistance_energy_immune",
		"damage_resistance_radiation_immune",
		"damage_resistance_poison_immune",
	}

	for _, table := range []string{"combatants", "player_characters"} {
		columns := queryColumnNames(t, store.db, table)
		for _, column := range legacyColumns {
			assert.NotContains(t, columns, column, "%s.%s should be removed", table, column)
		}
	}

	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'trigger' AND name LIKE 'trg_sync_%'`))
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name IN ('combatant_defense_by_location', 'player_character_defense_by_location')`))
}

func TestOpenAndMigrateDropsEncounterDifficultyCacheColumns(t *testing.T) {
	store := newTestStore(t)
	columns := queryColumnNames(t, store.db, "encounters")
	for _, column := range []string{
		"difficulty_label",
		"difficulty_score",
		"party_count",
		"party_avg_level",
		"party_xp_budget",
		"enemy_count",
		"enemy_avg_level",
		"enemy_total_xp",
	} {
		assert.NotContains(t, columns, column)
	}
}

func TestOpenAndMigrateDropsMonsterTemplateNameKey(t *testing.T) {
	store := newTestStore(t)
	assert.NotContains(t, queryColumnNames(t, store.db, "monster_templates"), "name_key")
	assert.Equal(t, "index", queryString(t, store.db, `SELECT type FROM sqlite_schema WHERE name = ?`, "idx_monster_templates_name_normalized"))

	_, err := store.db.Exec(`
		INSERT INTO stat_profiles (id, level)
		VALUES ('monster_template:name-key-a', 1), ('monster_template:name-key-b', 1);
	`)
	require.NoError(t, err)
	_, err = store.db.Exec(`
		INSERT INTO monster_templates (id, stat_profile_id, name)
		VALUES
			('name-key-a', 'monster_template:name-key-a', 'Name Key Mutant'),
			('name-key-b', 'monster_template:name-key-b', ' name key mutant ');
	`)
	require.Error(t, err)
}

func TestOpenAndMigrateRestrictsGlobalResistanceToPoison(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:   "global-resistance-schema",
		Name: "Global Resistance Schema",
		Combatants: []domain.Combatant{{
			ID:                  "global-resistance-combatant",
			Name:                "Raider",
			Side:                domain.SideNPC,
			Initiative:          8,
			HP:                  6,
			MaxHP:               6,
			ResistPhysicalTorso: 2,
			ResistPoison:        3,
		}},
	}))
	monster, err := store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		Name:                "Schema Sentry",
		Level:               1,
		Initiative:          5,
		HP:                  4,
		MaxHP:               4,
		ResistPhysicalTorso: 2,
		ResistPoison:        3,
	})
	require.NoError(t, err)
	globalID := queryInt64(t, store.db, `SELECT id FROM body_locations WHERE code = 'global'`)
	physicalID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = ?`, string(domain.DamagePhysical))
	energyID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = ?`, string(domain.DamageEnergy))
	radiationID := queryInt64(t, store.db, `SELECT id FROM damage_types WHERE code = ?`, string(domain.DamageRadiation))

	for _, row := range []struct {
		profileID    string
		damageTypeID int64
	}{
		{statProfileID(statProfileCombatantKind, "global-resistance-combatant"), physicalID},
		{statProfileID(statProfilePlayerCharacterKind, "repo-char-1"), energyID},
		{statProfileID(statProfileMonsterTemplateKind, monster.ID), radiationID},
	} {
		_, err = store.db.Exec(
			`DELETE FROM stat_profile_resistance_by_location
		     WHERE stat_profile_id = ?
		       AND damage_type_id = ?`,
			row.profileID,
			row.damageTypeID,
		)
		require.NoError(t, err)
	}

	_, err = store.db.Exec(
		`INSERT INTO stat_profile_resistance_by_location (
	         stat_profile_id, damage_type_id, body_location_id, resistance
	     ) VALUES (?, ?, ?, 1)`,
		statProfileID(statProfileCombatantKind, "global-resistance-combatant"),
		physicalID,
		globalID,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-poison global resistance must be zero")
	_, err = store.db.Exec(
		`INSERT INTO stat_profile_resistance_by_location (
	         stat_profile_id, damage_type_id, body_location_id, resistance
	     ) VALUES (?, ?, ?, 1)`,
		statProfileID(statProfilePlayerCharacterKind, "repo-char-1"),
		energyID,
		globalID,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-poison global resistance must be zero")
	_, err = store.db.Exec(
		`INSERT INTO stat_profile_resistance_by_location (
	         stat_profile_id, damage_type_id, body_location_id, resistance
	     ) VALUES (?, ?, ?, 1)`,
		statProfileID(statProfileMonsterTemplateKind, monster.ID),
		radiationID,
		globalID,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-poison global resistance must be zero")

	_, err = store.db.Exec(
		`UPDATE stat_profile_resistance_by_location
         SET resistance = 4
         WHERE stat_profile_id = ?
           AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
           AND body_location_id = ?`,
		statProfileID(statProfileCombatantKind, "global-resistance-combatant"),
		string(domain.DamagePoison),
		globalID,
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		int64(4),
		queryInt64(
			t,
			store.db,
			`SELECT resistance
             FROM stat_profile_resistance_by_location
             WHERE stat_profile_id = ?
               AND damage_type_id = (SELECT id FROM damage_types WHERE code = ?)
               AND body_location_id = ?`,
			statProfileID(statProfileCombatantKind, "global-resistance-combatant"),
			string(domain.DamagePoison),
			globalID,
		),
	)
}

func TestOpenAndMigrateEnforcesBaseTableCheckConstraints(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:        "schema-check-encounter",
		Name:      "Schema Check",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{{
			ID:         "schema-check-combatant",
			Name:       "Raider",
			Side:       domain.SideNPC,
			Level:      1,
			Initiative: 8,
			HP:         6,
			MaxHP:      6,
		}},
	}))
	require.NoError(t, store.AppendEncounterLog(t.Context(), "schema-check-encounter", 1, "Started"))
	monster, err := store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		Name:       "Schema Mutant",
		Level:      1,
		XP:         30,
		Initiative: 5,
		HP:         4,
		MaxHP:      4,
	})
	require.NoError(t, err)

	checks := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "encounter round",
			sql:  `UPDATE encounters SET round = 0 WHERE id = ?`,
			args: []any{"schema-check-encounter"},
		},
		{
			name: "campaign resources",
			sql:  `UPDATE campaigns SET party_ap = -1 WHERE id = ?`,
			args: []any{"repo-test-campaign"},
		},
		{
			name: "campaign party AP maximum",
			sql:  `UPDATE campaigns SET party_ap = 7 WHERE id = ?`,
			args: []any{"repo-test-campaign"},
		},
		{
			name: "combatant side",
			sql:  `UPDATE combatants SET side = 'other' WHERE id = ?`,
			args: []any{"schema-check-combatant"},
		},
		{
			name: "combatant hp",
			sql:  `UPDATE stat_profiles SET hp = 7, max_hp = 6 WHERE id = ?`,
			args: []any{statProfileID(statProfileCombatantKind, "schema-check-combatant")},
		},
		{
			name: "player character level",
			sql:  `UPDATE stat_profiles SET level = 0 WHERE id = ?`,
			args: []any{statProfileID(statProfilePlayerCharacterKind, "repo-char-1")},
		},
		{
			name: "player character hp",
			sql:  `UPDATE stat_profiles SET hp = 8, max_hp = 7 WHERE id = ?`,
			args: []any{statProfileID(statProfilePlayerCharacterKind, "repo-char-1")},
		},
		{
			name: "monster level",
			sql:  `UPDATE stat_profiles SET level = 0 WHERE id = ?`,
			args: []any{statProfileID(statProfileMonsterTemplateKind, monster.ID)},
		},
		{
			name: "body location enum",
			sql:  `INSERT INTO body_locations (id, code) VALUES (?, ?)`,
			args: []any{99, "wing"},
		},
		{
			name: "damage type enum",
			sql:  `INSERT INTO damage_types (id, code) VALUES (?, ?)`,
			args: []any{99, "fire"},
		},
		{
			name: "encounter log round",
			sql:  `UPDATE encounter_logs SET round = 0 WHERE encounter_id = ?`,
			args: []any{"schema-check-encounter"},
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			_, err := store.db.Exec(check.sql, check.args...)
			require.Error(t, err)
		})
	}
}

func TestMigration42MovesLatestEncounterResourcesToCampaign(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-42-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 41))

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, start_date)
		VALUES ('legacy-campaign', 'Legacy Campaign', '2026-01-01 00:00:00');
		INSERT INTO encounters (
			id, campaign_id, name, round, turn_index, party_ap, gm_threat, updated_at
		) VALUES
			('legacy-old', 'legacy-campaign', 'Old Fight', 1, 0, 1, 2, '2026-01-01 00:00:00.000'),
			('legacy-latest', 'legacy-campaign', 'Latest Fight', 2, 0, 9, 99, '2026-02-01 00:00:00.000');
	`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(db, "migrations", 42))

	assert.Equal(t, int64(domain.MaxPartyAP), queryInt64(t, db, `SELECT party_ap FROM campaigns WHERE id = ?`, "legacy-campaign"))
	assert.Equal(t, int64(99), queryInt64(t, db, `SELECT gm_threat FROM campaigns WHERE id = ?`, "legacy-campaign"))
	assert.NotContains(t, queryColumnNames(t, db, "encounters"), "party_ap")
	assert.NotContains(t, queryColumnNames(t, db, "encounters"), "gm_threat")
}

func TestMigration30BackfillsNullAuditFieldsAndDropsLeftoverTempTables(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-30-legacy.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 29))

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, start_date) VALUES ('legacy-campaign', 'Legacy Campaign', '2026-01-01 00:00:00');
		INSERT OR IGNORE INTO app_state (id, active_campaign_id) VALUES (1, 'legacy-campaign');
		INSERT INTO players (id, campaign_id, name) VALUES ('legacy-player', 'legacy-campaign', 'Legacy Player');
		INSERT INTO player_characters (
			id, player_id, campaign_id, name, level, initiative, hp, max_hp, defense, torso_only, active, availability_status
		) VALUES (
			'legacy-character', 'legacy-player', 'legacy-campaign', 'Legacy Character', 1, 7, 6, 6, 0, 0, 1, 'active'
		);
		INSERT INTO encounters (
			id, campaign_id, name, round, turn_index, party_ap, gm_threat,
			difficulty_label, difficulty_score, party_count, party_avg_level, party_xp_budget,
			enemy_count, enemy_avg_level, enemy_total_xp
		) VALUES (
			'legacy-encounter', 'legacy-campaign', 'Legacy Encounter', 1, 0, 0, 0,
			'Unknown', 0, 0, 0, 0, 0, 0, 0
		);
		INSERT INTO combatants (
			id, encounter_id, player_character_id, name, side, torso_only, initiative,
			active, defeated, position, hp, max_hp, defense, level, xp
		) VALUES (
			'legacy-combatant', 'legacy-encounter', NULL, 'Legacy Raider', 'npc', 0, 5,
			1, 0, 0, 4, 4, 0, 1, 0
		);
		INSERT INTO encounter_logs (id, encounter_id, round, message)
		VALUES ('legacy-log', 'legacy-encounter', 1, 'Legacy log');
		INSERT INTO monster_templates (
			id, name, name_key, torso_only, level, xp, initiative, hp, max_hp, defense
		) VALUES (
			'legacy-monster', 'Legacy Monster', 'legacy monster', 1, 1, 10, 4, 3, 3, 0
		);
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		UPDATE encounters SET created_at = NULL WHERE id = 'legacy-encounter';
		UPDATE combatants SET created_at = NULL, updated_at = NULL WHERE id = 'legacy-combatant';
		UPDATE encounter_logs SET updated_at = NULL WHERE encounter_id = 'legacy-encounter';
		UPDATE app_state SET updated_at = NULL WHERE id = 1;
		CREATE TABLE encounters_v3 (id TEXT);
	`)
	require.NoError(t, err)

	require.NoError(t, goose.Up(db, "migrations"))

	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'table' AND name = 'encounters_v3'`))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM encounters WHERE created_at IS NULL OR updated_at IS NULL`))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM combatants WHERE created_at IS NULL OR updated_at IS NULL`))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM encounter_logs WHERE created_at IS NULL OR updated_at IS NULL`))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM app_state WHERE updated_at IS NULL`))
}
