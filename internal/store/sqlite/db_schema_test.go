package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Equal(t, int64(4), queryInt64(t, db, `SELECT COUNT(*) FROM combatant_resistance_global WHERE combatant_id = ?`, "fk-npc-1"))

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
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM combatant_resistance_global WHERE combatant_id = ?`, "fk-npc-1"))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM combatant_resistance_by_location WHERE combatant_id = ?`, "fk-npc-1"))
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

	_, err = store.db.Exec(`UPDATE combatant_resistance_global SET resistance = 1 WHERE combatant_id = ? AND damage_type_id = 1`, "global-resistance-combatant")
	require.Error(t, err)
	_, err = store.db.Exec(`UPDATE player_character_resistance_global SET resistance = 1 WHERE player_character_id = ? AND damage_type_id = 2`, "repo-char-1")
	require.Error(t, err)
	_, err = store.db.Exec(`UPDATE monster_template_resistance_global SET resistance = 1 WHERE monster_template_id = ? AND damage_type_id = 3`, monster.ID)
	require.Error(t, err)

	_, err = store.db.Exec(`UPDATE combatant_resistance_global SET resistance = 4 WHERE combatant_id = ? AND damage_type_id = 4`, "global-resistance-combatant")
	require.NoError(t, err)
	assert.Equal(t, int64(4), queryInt64(t, store.db, `SELECT resistance FROM combatant_resistance_global WHERE combatant_id = ? AND damage_type_id = 4`, "global-resistance-combatant"))
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
			name: "encounter resources",
			sql:  `UPDATE encounters SET party_ap = -1 WHERE id = ?`,
			args: []any{"schema-check-encounter"},
		},
		{
			name: "combatant side",
			sql:  `UPDATE combatants SET side = 'other' WHERE id = ?`,
			args: []any{"schema-check-combatant"},
		},
		{
			name: "combatant hp",
			sql:  `UPDATE combatants SET hp = 7, max_hp = 6 WHERE id = ?`,
			args: []any{"schema-check-combatant"},
		},
		{
			name: "combatant bool",
			sql:  `UPDATE combatants SET active = 2 WHERE id = ?`,
			args: []any{"schema-check-combatant"},
		},
		{
			name: "player character level",
			sql:  `UPDATE player_characters SET level = 0 WHERE id = ?`,
			args: []any{"repo-char-1"},
		},
		{
			name: "player character hp",
			sql:  `UPDATE player_characters SET hp = 8, max_hp = 7 WHERE id = ?`,
			args: []any{"repo-char-1"},
		},
		{
			name: "monster level",
			sql:  `UPDATE monster_templates SET level = 0 WHERE id = ?`,
			args: []any{monster.ID},
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

	store := NewEncounterStore(db)
	_, err = store.CreateCampaign(t.Context(), "legacy-campaign", "Legacy Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{{
		PlayerName: "Legacy Player",
		Character: domain.Combatant{
			ID:         "legacy-character",
			Name:       "Legacy Character",
			Side:       domain.SideParty,
			Level:      1,
			Initiative: 7,
			HP:         6,
		},
	}})
	require.NoError(t, err)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:        "legacy-encounter",
		Name:      "Legacy Encounter",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{{
			ID:         "legacy-combatant",
			Name:       "Legacy Raider",
			Side:       domain.SideNPC,
			Level:      1,
			Initiative: 5,
			HP:         4,
		}},
	}))
	require.NoError(t, store.AppendEncounterLog(t.Context(), "legacy-encounter", 1, "Legacy log"))
	_, err = store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		Name:       "Legacy Monster",
		Level:      1,
		XP:         10,
		Initiative: 4,
		HP:         3,
	})
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
