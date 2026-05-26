package app

import (
	"testing"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEncounterRejectsEmptyCombatants(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "test", nil)
	require.Error(t, err)
}

func TestCreateEncounterRejectsInvalidCombatantValues(t *testing.T) {
	tests := []struct {
		name      string
		combatant domain.Combatant
		want      string
	}{
		{
			name:      "invalid side",
			combatant: domain.Combatant{Name: "Raider", Side: domain.Side("other"), Initiative: 8, HP: 6, MaxHP: 6},
			want:      "invalid side",
		},
		{
			name:      "negative hp",
			combatant: domain.Combatant{Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: -1, MaxHP: 6},
			want:      "invalid HP",
		},
		{
			name:      "hp exceeds max hp",
			combatant: domain.Combatant{Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: 7, MaxHP: 6},
			want:      "current HP cannot exceed max HP",
		},
		{
			name:      "negative resistance",
			combatant: domain.Combatant{Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: 6, MaxHP: 6, ResistEnergyTorso: -1},
			want:      "invalid resistance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newSQLiteService(t)

			_, err := svc.CreateEncounter(t.Context(), "enc-invalid", "Invalid", []domain.Combatant{tt.combatant})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCreateEncounterPersistsAndGetReturnsSorted(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "test", []domain.Combatant{
		{
			ID: "c2", Name: "Beta", Level: 4, XP: 0, Initiative: 7, Side: domain.SideParty, HP: 12, Defense: 1,
			ResistPhysical: 0, ResistEnergy: 1, ResistRadiation: 2, ResistPoison: 3,
			ImmunePoison: true,
		},
		{
			ID: "c1", Name: "Alpha", Level: 6, XP: 120, Initiative: 9, Side: domain.SideNPC, HP: 20, Defense: 2,
			ResistPhysical: 1, ResistEnergy: 2, ResistRadiation: 3, ResistPoison: 4,
			ImmuneEnergy: true,
		},
	})
	require.NoError(t, err)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)

	require.Len(t, enc.Combatants, 2)
	assert.Equal(t, "c1", enc.Combatants[0].ID)
	assert.Equal(t, 20, enc.Combatants[0].HP)
	assert.Equal(t, 2, enc.Combatants[0].Defense)
	assert.Equal(t, 6, enc.Combatants[0].Level)
	assert.Equal(t, 120, enc.Combatants[0].XP)
	assert.Equal(t, 0, enc.Combatants[0].ResistPhysical)
	assert.Equal(t, 0, enc.Combatants[0].ResistEnergy)
	assert.Equal(t, 0, enc.Combatants[0].ResistRadiation)
	assert.Equal(t, 4, enc.Combatants[0].ResistPoison)
	assert.False(t, enc.Combatants[0].ImmunePhysical)
	assert.True(t, enc.Combatants[0].ImmuneEnergy)
	assert.False(t, enc.Combatants[0].ImmuneRadiation)
	assert.False(t, enc.Combatants[0].ImmunePoison)
}

func TestCreateEncounterGeneratesUUIDsWhenIDsAreMissing(t *testing.T) {
	svc := newSQLiteService(t)
	created, err := svc.CreateEncounter(t.Context(), "", "test", []domain.Combatant{
		{Name: "Alpha", Initiative: 10, Side: domain.SideParty, HP: 10},
		{Name: "Beta", Initiative: 8, Side: domain.SideNPC, HP: 8},
	})
	require.NoError(t, err)

	_, err = uuid.Parse(created.ID)
	require.NoError(t, err)
	require.Len(t, created.Combatants, 2)
	for _, c := range created.Combatants {
		_, parseErr := uuid.Parse(c.ID)
		require.NoError(t, parseErr)
	}
}

func TestAdvanceTurnPersistsState(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "test", []domain.Combatant{
		{ID: "c1", Name: "Alpha", Initiative: 10},
		{ID: "c2", Name: "Beta", Initiative: 8},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, enc.TurnIndex)
	assert.True(t, enc.Combatants[1].Active)
}

func TestResourceCommands(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "test", []domain.Combatant{{ID: "c1", Name: "Alpha", Initiative: 10}})
	require.NoError(t, err)

	_, err = svc.AddPartyAP(t.Context(), 2)
	require.NoError(t, err)
	_, err = svc.SpendPartyAP(t.Context(), 1)
	require.NoError(t, err)
	_, err = svc.SpendPartyAP(t.Context(), 10)
	require.Error(t, err)

	_, err = svc.AddThreat(t.Context(), 2)
	require.NoError(t, err)
	_, err = svc.SpendThreat(t.Context(), 1)
	require.NoError(t, err)
	_, err = svc.SpendThreat(t.Context(), 10)
	require.Error(t, err)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, enc.Resources.PartyAP)
	assert.Equal(t, 1, enc.Resources.GMThreat)

	next, err := svc.CreateEncounter(t.Context(), "enc-2", "next", []domain.Combatant{{ID: "c2", Name: "Beta", Initiative: 8}})
	require.NoError(t, err)
	assert.Equal(t, 1, next.Resources.PartyAP)
	assert.Equal(t, 1, next.Resources.GMThreat)

	first, err := svc.ActivateEncounter(t.Context(), "enc-1")
	require.NoError(t, err)
	assert.Equal(t, 1, first.Resources.PartyAP)
	assert.Equal(t, 1, first.Resources.GMThreat)

	capped, err := svc.AddPartyAP(t.Context(), 20)
	require.NoError(t, err)
	assert.Equal(t, domain.MaxPartyAP, capped.Resources.PartyAP)

	uncappedThreat, err := svc.AddThreat(t.Context(), 100)
	require.NoError(t, err)
	assert.Equal(t, 101, uncappedThreat.Resources.GMThreat)
}

func TestListAndActivateEncounter(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)
	_, err = svc.CreateEncounter(t.Context(), "enc-2", "Bravo", []domain.Combatant{{ID: "c2", Name: "Two", Initiative: 8}})
	require.NoError(t, err)

	summaries, err := svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "enc-2", summaries[0].ID)

	active, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "enc-2", active.ID)

	active, err = svc.ActivateEncounter(t.Context(), "enc-1")
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)
}

func TestUpdateEncounterPreservesTurnIndexAndActiveCombatant(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "c1", Name: "One", Initiative: 10, Side: domain.SideParty, HP: 10, MaxHP: 10},
		{ID: "c2", Name: "Two", Initiative: 8, Side: domain.SideNPC, HP: 10, MaxHP: 10},
		{ID: "c3", Name: "Three", Initiative: 6, Side: domain.SideNPC, HP: 10, MaxHP: 10},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	_, err = svc.AddPartyAP(t.Context(), 2)
	require.NoError(t, err)

	beforeUpdate, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.GreaterOrEqual(t, beforeUpdate.TurnIndex, 0)
	require.Less(t, beforeUpdate.TurnIndex, len(beforeUpdate.Combatants))
	beforeActiveID := beforeUpdate.Combatants[beforeUpdate.TurnIndex].ID
	beforeCombatants := append([]domain.Combatant(nil), beforeUpdate.Combatants...)

	updated, err := svc.UpdateEncounter(t.Context(), beforeUpdate.ID, "Alpha Updated", beforeCombatants)
	require.NoError(t, err)
	assert.Equal(t, beforeUpdate.Round, updated.Round)
	assert.Equal(t, beforeUpdate.TurnIndex, updated.TurnIndex)
	require.GreaterOrEqual(t, updated.TurnIndex, 0)
	require.Less(t, updated.TurnIndex, len(updated.Combatants))
	assert.Equal(t, beforeActiveID, updated.Combatants[updated.TurnIndex].ID)

	persisted, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, beforeUpdate.TurnIndex, persisted.TurnIndex)
	require.GreaterOrEqual(t, persisted.TurnIndex, 0)
	require.Less(t, persisted.TurnIndex, len(persisted.Combatants))
	assert.Equal(t, beforeActiveID, persisted.Combatants[persisted.TurnIndex].ID)
}

func TestUpdateEncounterRejectsInvalidCombatantValues(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: 6, MaxHP: 6},
	})
	require.NoError(t, err)

	_, err = svc.UpdateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: 6, MaxHP: 6, Defense: -1},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid defense")
}

func TestListEncountersIncludesDifficultyMetrics(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Difficulty Check", []domain.Combatant{
		{ID: "p1", Name: "Vault Dweller", Side: domain.SideParty, Level: 2, Initiative: 10, HP: 8, Defense: 1},
		{ID: "p2", Name: "Companion", Side: domain.SideParty, Level: 2, Initiative: 9, HP: 7, Defense: 1},
		{ID: "n1", Name: "Raider", Side: domain.SideNPC, Level: 2, XP: 60, Initiative: 8, HP: 6, Defense: 0},
		{ID: "n2", Name: "Raider", Side: domain.SideNPC, Level: 2, XP: 60, Initiative: 7, HP: 6, Defense: 0},
	})
	require.NoError(t, err)

	summaries, err := svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 1)

	assert.Equal(t, "Difficulty Check", summaries[0].Name)
	assert.Equal(t, "Hard", summaries[0].Difficulty)
	assert.Equal(t, 2, summaries[0].PartyCount)
	assert.Equal(t, 2.0, summaries[0].PartyAvgLevel)
	assert.Equal(t, 60, summaries[0].PartyXPBudget)
	assert.Equal(t, 2, summaries[0].EnemyCount)
	assert.Equal(t, 2.0, summaries[0].EnemyAvgLevel)
	assert.Equal(t, 120, summaries[0].EnemyTotalXP)
	assert.Equal(t, 2.0, summaries[0].DifficultyScore)
}

func TestActivateEncounterNotFound(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)

	_, err = svc.ActivateEncounter(t.Context(), "enc-missing")
	require.ErrorIs(t, err, domain.ErrEncounterNotFound)
}

func TestRestartEncounterResetsRoundAndPreservesCampaignResources(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "c1", Name: "One", Initiative: 10, Side: domain.SideParty, HP: 10, MaxHP: 10},
		{ID: "c2", Name: "Two", Initiative: 8, Side: domain.SideNPC, HP: 8, MaxHP: 8},
	})
	require.NoError(t, err)

	_, _, err = svc.ApplyDamage(t.Context(), "c1", domain.DamagePhysical, domain.BodyTorso, 4)
	require.NoError(t, err)
	_, _, err = svc.ApplyDamage(t.Context(), "c2", domain.DamagePhysical, domain.BodyTorso, 99)
	require.NoError(t, err)
	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	_, err = svc.AddPartyAP(t.Context(), 3)
	require.NoError(t, err)
	_, err = svc.AddThreat(t.Context(), 2)
	require.NoError(t, err)

	restarted, err := svc.RestartEncounter(t.Context(), "enc-1")
	require.NoError(t, err)

	assert.Equal(t, "enc-1", restarted.ID)
	assert.Equal(t, 1, restarted.Round)
	assert.Equal(t, 0, restarted.TurnIndex)
	assert.Equal(t, 3, restarted.Resources.PartyAP)
	assert.Equal(t, 2, restarted.Resources.GMThreat)
	require.Len(t, restarted.Combatants, 2)
	assert.True(t, restarted.Combatants[0].Active)
	assert.False(t, restarted.Combatants[1].Active)
	assert.False(t, restarted.Combatants[0].Defeated)
	assert.False(t, restarted.Combatants[1].Defeated)
	assert.Equal(t, 6, restarted.Combatants[0].HP)
	assert.Equal(t, 8, restarted.Combatants[1].HP)
	assert.Equal(t, 8, restarted.Combatants[1].MaxHP)
}

func TestSoftDeleteEncounterHidesFromListAndActivation(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)
	_, err = svc.CreateEncounter(t.Context(), "enc-2", "Bravo", []domain.Combatant{{ID: "c2", Name: "Two", Initiative: 8}})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteEncounter(t.Context(), "enc-2"))

	summaries, err := svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "enc-1", summaries[0].ID)

	active, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)

	_, err = svc.ActivateEncounter(t.Context(), "enc-2")
	require.ErrorIs(t, err, domain.ErrEncounterNotFound)
}

func TestCreateEncounterRejectsDuplicatePartyCharacter(t *testing.T) {
	svc := newSQLiteService(t)

	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{PlayerCharacterID: "char-1", Name: "Vault Dweller", Initiative: 10, Side: domain.SideParty, HP: 7, MaxHP: 7},
		{PlayerCharacterID: "char-1", Name: "Vault Dweller", Initiative: 9, Side: domain.SideParty, HP: 7, MaxHP: 7},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already in this encounter")
}

func TestListPartyMembersUsesActiveCampaignCharactersFirst(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Roland", Initiative: 9, Side: domain.SideParty, Level: 1, HP: 7, Defense: 2},
		{ID: "n1", Name: "Raider", Initiative: 7, Side: domain.SideNPC, Level: 1, XP: 30, HP: 6, Defense: 1},
	})
	require.NoError(t, err)
	_, err = svc.CreateEncounter(t.Context(), "enc-2", "Bravo", []domain.Combatant{
		{ID: "p2", Name: "Piper", Initiative: 8, Side: domain.SideParty, Level: 2, HP: 8, Defense: 3},
		{ID: "p3", Name: "Roland", Initiative: 11, Side: domain.SideParty, Level: 3, HP: 9, Defense: 4},
	})
	require.NoError(t, err)

	party, err := svc.ListPartyMembers(t.Context())
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, "Vault Dweller", party[0].Name)
	assert.Equal(t, 1, party[0].Level)
	assert.Equal(t, 9, party[0].Initiative)
}
