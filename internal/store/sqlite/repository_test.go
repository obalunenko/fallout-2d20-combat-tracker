package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreGetWhenEmpty(t *testing.T) {
	store := newTestStore(t)

	enc, err := store.Get()
	require.ErrorIs(t, err, domain.ErrEncounterNotInitialized)
	assert.Nil(t, enc)
}

func TestEncounterStoreSaveAndGetRoundTrip(t *testing.T) {
	store := newTestStore(t)

	expected := &domain.Encounter{
		ID:        "enc-1",
		Name:      "Vault Entrance",
		Round:     3,
		TurnIndex: 1,
		Resources: domain.Resources{
			PartyAP:  4,
			GMThreat: 2,
		},
		Combatants: []domain.Combatant{
			{
				ID: "p1", Name: "Roland", Side: domain.SideParty,
				Level: 7, XP: 0, Initiative: 11, HP: 22, MaxHP: 22, Defense: 2,
				ResistPhysicalHead: 2, ResistPhysicalTorso: 2, ResistPhysicalLeftArm: 2, ResistPhysicalRightArm: 2, ResistPhysicalLeftLeg: 2, ResistPhysicalRightLeg: 2,
				ResistPhysical: 3, ResistEnergy: 2, ResistRadiation: 1, ResistPoison: 0,
				ImmunePoison: true, Active: false, Defeated: false,
			},
			{
				ID: "n1", Name: "Radscorpion", Side: domain.SideNPC,
				Level: 5, XP: 80, Initiative: 9, HP: 18, MaxHP: 18, Defense: 1,
				ResistPhysicalHead: 1, ResistPhysicalTorso: 1, ResistPhysicalLeftArm: 1, ResistPhysicalRightArm: 1, ResistPhysicalLeftLeg: 1, ResistPhysicalRightLeg: 1,
				ResistPhysical: 2, ResistEnergy: 1, ResistRadiation: 4, ResistPoison: 3,
				ImmunePhysical: true, Active: true, Defeated: false,
			},
		},
	}

	require.NoError(t, store.Save(expected))

	actual, err := store.Get()
	require.NoError(t, err)
	require.NotNil(t, actual)

	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Round, actual.Round)
	assert.Equal(t, expected.TurnIndex, actual.TurnIndex)
	assert.Equal(t, expected.Resources, actual.Resources)
	require.Len(t, actual.Combatants, 2)
	assert.Equal(t, expected.Combatants, actual.Combatants)
}

func TestEncounterStoreSaveReplacesCombatants(t *testing.T) {
	store := newTestStore(t)

	first := &domain.Encounter{
		ID:   "enc-1",
		Name: "Initial",
		Combatants: []domain.Combatant{
			{ID: "c1", Name: "One", Side: domain.SideParty, Initiative: 10, Active: true},
			{ID: "c2", Name: "Two", Side: domain.SideNPC, Initiative: 8, Active: false},
		},
	}
	require.NoError(t, store.Save(first))

	updated := &domain.Encounter{
		ID:        "enc-1",
		Name:      "Updated",
		Round:     2,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "c3", Name: "Three", Side: domain.SideNPC, Initiative: 12, Active: true},
		},
	}
	require.NoError(t, store.Save(updated))

	actual, err := store.Get()
	require.NoError(t, err)
	require.Len(t, actual.Combatants, 1)
	assert.Equal(t, "c3", actual.Combatants[0].ID)
	assert.Equal(t, "Updated", actual.Name)
}

func TestEncounterStoreListActivateSoftDelete(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "A", Side: domain.SideParty, Initiative: 10, Active: true}},
	}))
	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-2",
		Name:       "Bravo",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c2", Name: "B", Side: domain.SideNPC, Initiative: 8, Active: true}},
	}))

	summaries, err := store.List()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "enc-2", summaries[0].ID)

	require.NoError(t, store.Activate("enc-1"))

	active, err := store.Get()
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)

	require.NoError(t, store.SoftDelete("enc-1"))
	active, err = store.Get()
	require.NoError(t, err)
	assert.Equal(t, "enc-2", active.ID)

	summaries, err = store.List()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "enc-2", summaries[0].ID)
}

func TestEncounterStorePersistsDifficultyMetrics(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(&domain.Encounter{
		ID:        "enc-diff-1",
		Name:      "Difficulty Persist",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "p1", Name: "P1", Side: domain.SideParty, Level: 2, Initiative: 10, HP: 8, Active: true},
			{ID: "p2", Name: "P2", Side: domain.SideParty, Level: 2, Initiative: 9, HP: 8},
			{ID: "n1", Name: "N1", Side: domain.SideNPC, Level: 2, XP: 60, Initiative: 8, HP: 6},
			{ID: "n2", Name: "N2", Side: domain.SideNPC, Level: 2, XP: 60, Initiative: 7, HP: 6},
		},
	}))

	summaries, err := store.List()
	require.NoError(t, err)
	require.NotEmpty(t, summaries)

	var summary *domain.EncounterSummary
	for i := range summaries {
		if summaries[i].ID == "enc-diff-1" {
			summary = &summaries[i]
			break
		}
	}
	require.NotNil(t, summary)
	assert.Equal(t, "Hard", summary.Difficulty)
	assert.Equal(t, 2.0, summary.DifficultyScore)
	assert.Equal(t, 2, summary.PartyCount)
	assert.Equal(t, 2.0, summary.PartyAvgLevel)
	assert.Equal(t, 60, summary.PartyXPBudget)
	assert.Equal(t, 2, summary.EnemyCount)
	assert.Equal(t, 2.0, summary.EnemyAvgLevel)
	assert.Equal(t, 120, summary.EnemyTotalXP)
}

func TestEncounterStoreNotFoundOperations(t *testing.T) {
	store := newTestStore(t)

	require.ErrorIs(t, store.Activate("missing"), domain.ErrEncounterNotFound)
	require.ErrorIs(t, store.SoftDelete("missing"), domain.ErrEncounterNotFound)
}

func TestEncounterStoreSaveNilEncounter(t *testing.T) {
	store := newTestStore(t)
	require.Error(t, store.Save(nil))
}

func TestEncounterStoreStopsQueriesWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := newTestStoreWithContext(t, ctx)

	cancel()

	_, err := store.Get()
	require.ErrorIs(t, err, context.Canceled)

	err = store.Save(&domain.Encounter{
		ID:         "enc-1",
		Name:       "Canceled",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "One", Side: domain.SideParty, Initiative: 10, Active: true}},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestEncounterStoreAppendAndListLogs(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "A", Side: domain.SideParty, Initiative: 10, Active: true}},
	}))

	require.NoError(t, store.AppendEncounterLog("enc-1", 1, "Encounter created"))
	require.NoError(t, store.AppendEncounterLog("enc-1", 2, "Turn advanced -> A"))

	logs, err := store.ListEncounterLogs("enc-1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, 2, logs[0].Round)
	assert.Contains(t, logs[0].Message, "Turn advanced")
	assert.NotEmpty(t, logs[0].CreatedAt)
	assert.Equal(t, 1, logs[1].Round)
}

func TestEncounterStoreListPartyMembersUsesActiveCampaignCharactersFirst(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-1",
		Name:       "Old",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p1", Name: "Roland", Side: domain.SideParty, Level: 1, Initiative: 8, HP: 7, Defense: 2}, {ID: "n1", Name: "Raider", Side: domain.SideNPC, XP: 30, Initiative: 7, HP: 6}},
	}))
	require.NoError(t, store.Save(&domain.Encounter{
		ID:        "enc-2",
		Name:      "New",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "p2", Name: "Piper", Side: domain.SideParty, Level: 3, Initiative: 10, HP: 9, Defense: 3, ResistEnergy: 1},
			{ID: "p3", Name: "Roland", Side: domain.SideParty, Level: 2, Initiative: 11, HP: 8, Defense: 4, ResistPhysical: 2, ImmunePoison: true},
			{ID: "n2", Name: "Molerat", Side: domain.SideNPC, XP: 25, Initiative: 6, HP: 5},
		},
	}))

	party, err := store.ListPartyMembers()
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, "Scout", party[0].Name)
	assert.Equal(t, 1, party[0].Level)
	assert.Equal(t, 7, party[0].Initiative)
}

func TestEncounterStoreListPartyMembersReturnsEmptyWhenNoActiveCharacters(t *testing.T) {
	store := newTestStore(t)
	_, err := store.db.Exec(`UPDATE player_characters SET active = 0 WHERE campaign_id = ?`, "repo-test-campaign")
	require.NoError(t, err)

	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p1", Name: "Roland", Side: domain.SideParty, Level: 2, Initiative: 9, HP: 7, Defense: 2}},
	}))
	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "enc-2",
		Name:       "Bravo",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p2", Name: "Roland", Side: domain.SideParty, Level: 4, Initiative: 12, HP: 10, Defense: 5}},
	}))
	require.NoError(t, store.SoftDelete("enc-2"))

	party, err := store.ListPartyMembers()
	require.NoError(t, err)
	require.Empty(t, party)
}

func TestEncounterStoreUpdateCampaignKeepsInactiveCharacterHistory(t *testing.T) {
	store := newTestStore(t)

	_, err := store.UpdateCampaign("repo-test-campaign", "Repo Test Campaign", "2026-01-01", []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "repo-char-2",
				Name:       "Ranger",
				Side:       domain.SideParty,
				Level:      2,
				Initiative: 8,
				HP:         8,
				MaxHP:      8,
				Defense:    2,
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(2), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))`,
		"repo-test-campaign",
		"Player 1",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?)) AND pc.active = 1`,
		"repo-test-campaign",
		"Player 1",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))
           AND lower(trim(pc.name)) = lower(trim(?)) AND pc.active = 0`,
		"repo-test-campaign",
		"Player 1",
		"Scout",
	))
	assert.Equal(t, int64(1), queryInt64(
		t,
		store.db,
		`SELECT COUNT(*) FROM player_characters pc
         JOIN players p ON p.id = pc.player_id
         WHERE p.campaign_id = ? AND lower(trim(p.name)) = lower(trim(?))
           AND lower(trim(pc.name)) = lower(trim(?)) AND pc.active = 1`,
		"repo-test-campaign",
		"Player 1",
		"Ranger",
	))

	party, err := store.ListPartyMembers()
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, "Ranger", party[0].Name)
}

func TestEncounterStoreSaveWritesNormalizedStatsWithoutTriggers(t *testing.T) {
	store := newTestStore(t)
	dropNormalizedSyncTriggers(t, store.db)

	combatant := domain.Combatant{
		ID:             "norm-c1",
		Name:           "Sentry",
		Side:           domain.SideNPC,
		Initiative:     8,
		HP:             10,
		MaxHP:          10,
		DefenseHead:    3,
		DefenseTorso:   4,
		DefenseLeftArm: 5,
		DefenseRightArm: 6,
		DefenseLeftLeg: 7,
		DefenseRightLeg: 8,
		ResistPhysical: 2,
		ResistEnergy:   3,
		ResistRadiation: 4,
		ResistPoison:    5,
		ResistPhysicalHead:      1,
		ResistPhysicalTorso:     2,
		ResistPhysicalLeftArm:   3,
		ResistPhysicalRightArm:  4,
		ResistPhysicalLeftLeg:   5,
		ResistPhysicalRightLeg:  6,
		ResistEnergyHead:        2,
		ResistEnergyTorso:       3,
		ResistEnergyLeftArm:     4,
		ResistEnergyRightArm:    5,
		ResistEnergyLeftLeg:     6,
		ResistEnergyRightLeg:    7,
		ResistRadiationHead:     3,
		ResistRadiationTorso:    4,
		ResistRadiationLeftArm:  5,
		ResistRadiationRightArm: 6,
		ResistRadiationLeftLeg:  7,
		ResistRadiationRightLeg: 8,
		ImmunePhysical:          true,
	}
	require.NoError(t, store.Save(&domain.Encounter{
		ID:         "norm-enc-1",
		Name:       "Normalized Save",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{combatant},
	}))

	assert.Equal(t, int64(6), queryInt64(t, store.db, `SELECT COUNT(*) FROM combatant_defense_by_location WHERE combatant_id = ?`, combatant.ID))
	assert.Equal(t, int64(4), queryInt64(t, store.db, `SELECT COUNT(*) FROM combatant_resistance_global WHERE combatant_id = ?`, combatant.ID))
	assert.Equal(t, int64(18), queryInt64(t, store.db, `SELECT COUNT(*) FROM combatant_resistance_by_location WHERE combatant_id = ?`, combatant.ID))
	assert.Equal(t, int64(3), queryInt64(t, store.db, `SELECT defense FROM combatant_defense_by_location WHERE combatant_id = ? AND body_location_id = 1`, combatant.ID))
	assert.Equal(t, int64(1), queryInt64(t, store.db, `SELECT immune FROM combatant_resistance_global WHERE combatant_id = ? AND damage_type_id = 1`, combatant.ID))
}

func TestEncounterStoreCreateCampaignWritesNormalizedStatsWithoutTriggers(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "repo-norm-campaign.db")
	db, err := OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})
	dropNormalizedSyncTriggers(t, db)

	store := NewEncounterStore(db)
	characterID := "norm-char-1"
	_, err = store.CreateCampaign("norm-campaign-1", "Normalized Campaign", "2026-01-01", []domain.NewCampaignPlayer{
		{
			PlayerName: "Player A",
			Character: domain.Combatant{
				ID:                      characterID,
				Name:                    "Vera",
				Side:                    domain.SideParty,
				Level:                   2,
				Initiative:              9,
				HP:                      11,
				MaxHP:                   11,
				DefenseHead:             2,
				DefenseTorso:            2,
				DefenseLeftArm:          2,
				DefenseRightArm:         2,
				DefenseLeftLeg:          2,
				DefenseRightLeg:         2,
				ResistEnergy:            3,
				ResistEnergyHead:        1,
				ResistEnergyTorso:       2,
				ResistEnergyLeftArm:     3,
				ResistEnergyRightArm:    4,
				ResistEnergyLeftLeg:     5,
				ResistEnergyRightLeg:    6,
				ImmuneRadiation:         true,
				ResistRadiation:         7,
				ResistRadiationHead:     1,
				ResistRadiationTorso:    1,
				ResistRadiationLeftArm:  1,
				ResistRadiationRightArm: 1,
				ResistRadiationLeftLeg:  1,
				ResistRadiationRightLeg: 1,
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, int64(6), queryInt64(t, db, `SELECT COUNT(*) FROM player_character_defense_by_location WHERE player_character_id = ?`, characterID))
	assert.Equal(t, int64(4), queryInt64(t, db, `SELECT COUNT(*) FROM player_character_resistance_global WHERE player_character_id = ?`, characterID))
	assert.Equal(t, int64(18), queryInt64(t, db, `SELECT COUNT(*) FROM player_character_resistance_by_location WHERE player_character_id = ?`, characterID))
	assert.Equal(t, int64(3), queryInt64(t, db, `SELECT resistance FROM player_character_resistance_global WHERE player_character_id = ? AND damage_type_id = 2`, characterID))
	assert.Equal(t, int64(1), queryInt64(t, db, `SELECT immune FROM player_character_resistance_global WHERE player_character_id = ? AND damage_type_id = 3`, characterID))
}

func newTestStore(t *testing.T) *EncounterStore {
	t.Helper()
	return newTestStoreWithContext(t, context.Background())
}

func newTestStoreWithContext(t *testing.T, ctx context.Context) *EncounterStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "repo-test.db")
	db, err := OpenAndMigrate(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	store := NewEncounterStoreWithContext(db, ctx)
	_, err = store.CreateCampaign("repo-test-campaign", "Repo Test Campaign", "2026-01-01", []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "repo-char-1",
				Name:       "Scout",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 7,
				HP:         6,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)
	return store
}

func dropNormalizedSyncTriggers(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_update;
		DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_insert;
		DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_update;
		DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_insert;
	`)
	require.NoError(t, err)
}

func queryInt64(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var v int64
	require.NoError(t, db.QueryRow(query, args...).Scan(&v))
	return v
}
