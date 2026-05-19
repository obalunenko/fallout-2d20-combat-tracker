package sqlite

import (
	"context"
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
				ResistPhysical: 3, ResistEnergy: 2, ResistRadiation: 1, ResistPoison: 0,
				ImmunePoison: true, Active: false, Defeated: false,
			},
			{
				ID: "n1", Name: "Radscorpion", Side: domain.SideNPC,
				Level: 5, XP: 80, Initiative: 9, HP: 18, MaxHP: 18, Defense: 1,
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

func TestEncounterStoreListPartyMembersFallbacksToEncounterTemplatesWhenNoActiveCharacters(t *testing.T) {
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
	require.Len(t, party, 1)
	assert.Equal(t, "Roland", party[0].Name)
	assert.Equal(t, 2, party[0].Level)
	assert.Equal(t, 9, party[0].Initiative)
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
