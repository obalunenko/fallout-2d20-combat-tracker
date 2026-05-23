package sqlite

import (
	"context"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreGetWhenEmpty(t *testing.T) {
	store := newTestStore(t)

	enc, err := store.Get(t.Context())
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
				ResistPoison: 0,
				ImmunePoison: true, Active: false, Defeated: false,
			},
			{
				ID: "n1", Name: "Radscorpion", Side: domain.SideNPC,
				Level: 5, XP: 80, Initiative: 9, HP: 18, MaxHP: 18, Defense: 1,
				ResistPhysicalHead: 1, ResistPhysicalTorso: 1, ResistPhysicalLeftArm: 1, ResistPhysicalRightArm: 1, ResistPhysicalLeftLeg: 1, ResistPhysicalRightLeg: 1,
				ResistPoison:   3,
				ImmunePhysical: true, Active: true, Defeated: false,
			},
		},
	}

	require.NoError(t, store.Save(t.Context(), expected))

	actual, err := store.Get(t.Context())
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
	require.NoError(t, store.Save(t.Context(), first))

	updated := &domain.Encounter{
		ID:        "enc-1",
		Name:      "Updated",
		Round:     2,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "c3", Name: "Three", Side: domain.SideNPC, Initiative: 12, Active: true},
		},
	}
	require.NoError(t, store.Save(t.Context(), updated))

	actual, err := store.Get(t.Context())
	require.NoError(t, err)
	require.Len(t, actual.Combatants, 1)
	assert.Equal(t, "c3", actual.Combatants[0].ID)
	assert.Equal(t, "Updated", actual.Name)
}

func TestEncounterStoreListActivateSoftDelete(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "A", Side: domain.SideParty, Initiative: 10, Active: true}},
	}))
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-2",
		Name:       "Bravo",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c2", Name: "B", Side: domain.SideNPC, Initiative: 8, Active: true}},
	}))

	summaries, err := store.List(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "enc-2", summaries[0].ID)

	require.NoError(t, store.Activate(t.Context(), "enc-1"))

	active, err := store.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)

	require.NoError(t, store.SoftDelete(t.Context(), "enc-1"))
	active, err = store.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "enc-2", active.ID)

	summaries, err = store.List(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "enc-2", summaries[0].ID)
}

func TestEncounterStoreMaintainsEncounterAuditFields(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-audit-1",
		Name:       "Audit Fields",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c-audit-1", Name: "Scout", Side: domain.SideParty, Initiative: 10, HP: 5, MaxHP: 5, Active: true}},
	}))
	inserted := queryAuditFields(t, store.db, "encounters", "enc-audit-1")

	assert.True(t, inserted.createdAt.Valid)
	assert.True(t, inserted.updatedAt.Valid)
	assert.False(t, inserted.deletedAt.Valid)
	assert.False(t, inserted.updatedAt.Time.Before(inserted.createdAt.Time))

	require.NoError(t, store.Activate(t.Context(), "enc-audit-1"))
	activated := queryAuditFields(t, store.db, "encounters", "enc-audit-1")

	assert.True(t, activated.createdAt.Valid)
	assert.True(t, activated.updatedAt.Valid)
	assert.False(t, activated.deletedAt.Valid)
	assert.Equal(t, inserted.createdAt.Time, activated.createdAt.Time)
	assert.False(t, activated.updatedAt.Time.Before(inserted.updatedAt.Time))

	require.NoError(t, store.SoftDelete(t.Context(), "enc-audit-1"))
	deleted := queryAuditFields(t, store.db, "encounters", "enc-audit-1")

	assert.True(t, deleted.createdAt.Valid)
	assert.True(t, deleted.updatedAt.Valid)
	assert.True(t, deleted.deletedAt.Valid)
	assert.Equal(t, inserted.createdAt.Time, deleted.createdAt.Time)
	assert.False(t, deleted.updatedAt.Time.Before(activated.updatedAt.Time))
	assert.False(t, deleted.deletedAt.Time.Before(deleted.createdAt.Time))
}

func TestEncounterStoreMaintainsCombatantAuditFields(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-combatant-audit",
		Name:       "Combatant Audit",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "combatant-audit-1", Name: "Scout", Side: domain.SideParty, Initiative: 10, HP: 5, MaxHP: 5, Active: true}},
	}))
	fields := queryAuditFields(t, store.db, "combatants", "combatant-audit-1")

	assert.True(t, fields.createdAt.Valid)
	assert.True(t, fields.updatedAt.Valid)
	assert.False(t, fields.deletedAt.Valid)
	assert.False(t, fields.updatedAt.Time.Before(fields.createdAt.Time))
}
func TestEncounterStorePartyCombatantUsesCampaignCharacterStats(t *testing.T) {
	store := newTestStore(t)
	enc := &domain.Encounter{
		ID:        "enc-party-hp",
		Name:      "Linked Party HP",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "repo-char-1", Name: "Scout", Side: domain.SideParty, Initiative: 10, HP: 6, MaxHP: 6, Active: true},
			{
				ID:                      "npc-1",
				Name:                    "Raider",
				Side:                    domain.SideNPC,
				Initiative:              8,
				HP:                      5,
				MaxHP:                   5,
				ResistPhysical:          4,
				ResistEnergyRightArm:    6,
				ResistRadiationLeftLeg:  7,
				ResistRadiationRightLeg: 8,
				ImmunePhysical:          true,
			},
		},
	}
	require.NoError(t, store.Save(t.Context(), enc))

	_, err := store.UpdateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:                "repo-char-1",
				Name:              "Scout",
				Side:              domain.SideParty,
				Level:             3,
				Initiative:        12,
				HP:                4,
				MaxHP:             8,
				Defense:           5,
				TorsoOnly:         true,
				ResistEnergyTorso: 2,
				ImmunePoison:      true,
				ResistPoison:      1,
			},
		},
	})
	require.NoError(t, err)
	_, err = store.db.Exec(`UPDATE player_characters SET name = ? WHERE id = ?`, "Scout Prime", "repo-char-1")
	require.NoError(t, err)

	actual, err := store.Get(t.Context())
	require.NoError(t, err)
	require.Len(t, actual.Combatants, 2)
	assert.Equal(t, "Scout Prime", actual.Combatants[0].Name)
	assert.Equal(t, 3, actual.Combatants[0].Level)
	assert.Equal(t, 12, actual.Combatants[0].Initiative)
	assert.Equal(t, 4, actual.Combatants[0].HP)
	assert.Equal(t, 8, actual.Combatants[0].MaxHP)
	assert.Equal(t, 5, actual.Combatants[0].Defense)
	assert.True(t, actual.Combatants[0].TorsoOnly)
	assert.Equal(t, 2, actual.Combatants[0].ResistEnergyTorso)
	assert.Equal(t, 1, actual.Combatants[0].ResistPoison)
	assert.True(t, actual.Combatants[0].ImmunePoison)
	assert.False(t, actual.Combatants[0].Defeated)
	assert.Equal(t, 0, actual.Combatants[1].ResistPhysical)
	assert.Equal(t, 6, actual.Combatants[1].ResistEnergyRightArm)
	assert.Equal(t, 7, actual.Combatants[1].ResistRadiationLeftLeg)
	assert.Equal(t, 8, actual.Combatants[1].ResistRadiationRightLeg)
	assert.True(t, actual.Combatants[1].ImmunePhysical)

	_, err = store.db.Exec(`UPDATE player_characters SET hp = 0 WHERE id = ?`, "repo-char-1")
	require.NoError(t, err)

	actual, err = store.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 0, actual.Combatants[0].HP)
	assert.True(t, actual.Combatants[0].Defeated)
}

func TestEncounterStoreSaveUpdatesLinkedCampaignCharacterFromPartyCombatant(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:        "enc-sync-party-hp",
		Name:      "Sync Party HP",
		Round:     1,
		TurnIndex: 0,
		Combatants: []domain.Combatant{
			{ID: "repo-char-1", Name: "Scout", Side: domain.SideParty, Initiative: 10, HP: 6, MaxHP: 6, Active: true},
		},
	}))

	enc, err := store.Get(t.Context())
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 1)
	enc.Combatants[0].HP = 2
	enc.Combatants[0].Level = 4
	enc.Combatants[0].Defense = 3
	enc.Combatants[0].ResistPoison = 2

	require.NoError(t, store.Save(t.Context(), enc))

	assert.Equal(t, int64(2), queryInt64(t, store.db, `SELECT hp FROM player_characters WHERE id = ?`, "repo-char-1"))
	party, err := store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, 4, party[0].Level)
	assert.Equal(t, 3, party[0].Defense)
	assert.Equal(t, 2, party[0].ResistPoison)
}

func TestEncounterStorePersistsDifficultyMetrics(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
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

	summaries, err := store.List(t.Context())
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

func TestEncounterStoreUpdateEncounterPreservesTurnIndexAndActiveCombatant(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:        "enc-update-1",
		Name:      "Before Update",
		Round:     3,
		TurnIndex: 1,
		Resources: domain.Resources{
			PartyAP:  2,
			GMThreat: 1,
		},
		Combatants: []domain.Combatant{
			{ID: "c1", Name: "One", Side: domain.SideParty, Initiative: 10, HP: 9, MaxHP: 9, Active: false},
			{ID: "c2", Name: "Two", Side: domain.SideNPC, Initiative: 8, HP: 7, MaxHP: 7, Active: true},
		},
	}))

	beforeUpdate, err := store.Get(t.Context())
	require.NoError(t, err)
	require.GreaterOrEqual(t, beforeUpdate.TurnIndex, 0)
	require.Less(t, beforeUpdate.TurnIndex, len(beforeUpdate.Combatants))
	beforeActiveID := beforeUpdate.Combatants[beforeUpdate.TurnIndex].ID
	beforeCombatants := append([]domain.Combatant(nil), beforeUpdate.Combatants...)

	updated, err := store.UpdateEncounter(t.Context(), beforeUpdate.ID, "After Update", beforeCombatants)
	require.NoError(t, err)
	assert.Equal(t, beforeUpdate.Round, updated.Round)
	assert.Equal(t, beforeUpdate.TurnIndex, updated.TurnIndex)
	require.GreaterOrEqual(t, updated.TurnIndex, 0)
	require.Less(t, updated.TurnIndex, len(updated.Combatants))
	assert.Equal(t, beforeActiveID, updated.Combatants[updated.TurnIndex].ID)

	persisted, err := store.Get(t.Context())
	require.NoError(t, err)
	assert.Equal(t, beforeUpdate.TurnIndex, persisted.TurnIndex)
	require.GreaterOrEqual(t, persisted.TurnIndex, 0)
	require.Less(t, persisted.TurnIndex, len(persisted.Combatants))
	assert.Equal(t, beforeActiveID, persisted.Combatants[persisted.TurnIndex].ID)
}

func TestEncounterStoreNotFoundOperations(t *testing.T) {
	store := newTestStore(t)

	require.ErrorIs(t, store.Activate(t.Context(), "missing"), domain.ErrEncounterNotFound)
	require.ErrorIs(t, store.SoftDelete(t.Context(), "missing"), domain.ErrEncounterNotFound)
}

func TestEncounterStoreSaveNilEncounter(t *testing.T) {
	store := newTestStore(t)
	require.Error(t, store.Save(t.Context(), nil))
}

func TestEncounterStoreStopsQueriesWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	store := newTestStore(t)

	cancel()

	_, err := store.Get(ctx)
	require.ErrorIs(t, err, context.Canceled)

	err = store.Save(ctx, &domain.Encounter{
		ID:         "enc-1",
		Name:       "Canceled",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "One", Side: domain.SideParty, Initiative: 10, Active: true}},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestEncounterStoreListPartyMembersUsesActiveCampaignCharactersFirst(t *testing.T) {
	store := newTestStore(t)

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-1",
		Name:       "Old",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p1", Name: "Roland", Side: domain.SideParty, Level: 1, Initiative: 8, HP: 7, Defense: 2}, {ID: "n1", Name: "Raider", Side: domain.SideNPC, XP: 30, Initiative: 7, HP: 6}},
	}))
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
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

	party, err := store.ListPartyMembers(t.Context())
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

	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p1", Name: "Roland", Side: domain.SideParty, Level: 2, Initiative: 9, HP: 7, Defense: 2}},
	}))
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-2",
		Name:       "Bravo",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "p2", Name: "Roland", Side: domain.SideParty, Level: 4, Initiative: 12, HP: 10, Defense: 5}},
	}))
	require.NoError(t, store.SoftDelete(t.Context(), "enc-2"))

	party, err := store.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Empty(t, party)
}
