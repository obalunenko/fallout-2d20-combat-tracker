package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEncounterRejectsEmptyCombatants(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "test", nil)
	require.Error(t, err)
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
	assert.Equal(t, 1, enc.Combatants[0].ResistPhysical)
	assert.Equal(t, 2, enc.Combatants[0].ResistEnergy)
	assert.Equal(t, 3, enc.Combatants[0].ResistRadiation)
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

func TestRestartEncounterResetsRoundAndResources(t *testing.T) {
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
	assert.Equal(t, 0, restarted.Resources.PartyAP)
	assert.Equal(t, 0, restarted.Resources.GMThreat)
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

func TestApplyDamagePersistsHPAndDefeated(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Player", Initiative: 10, Side: domain.SideParty, HP: 12},
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 5},
	})
	require.NoError(t, err)

	_, applied, err := svc.ApplyDamage(t.Context(), "n1", domain.DamageEnergy, domain.BodyTorso, 9)
	require.NoError(t, err)
	assert.Equal(t, 9, applied)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 2)
	assert.Equal(t, 0, enc.Combatants[1].HP)
	assert.True(t, enc.Combatants[1].Defeated)
}

func TestApplyDamageAndHealPartyMemberPersistsCampaignCharacterHP(t *testing.T) {
	svc := newSQLiteService(t)
	created, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "char-1", Name: "Draft Copy", Level: 99, Initiative: 10, Side: domain.SideParty, HP: 99, MaxHP: 99, Defense: 9},
	})
	require.NoError(t, err)
	require.Len(t, created.Combatants, 1)
	assert.Equal(t, "Vault Dweller", created.Combatants[0].Name)
	assert.Equal(t, 1, created.Combatants[0].Level)
	assert.Equal(t, 9, created.Combatants[0].Initiative)
	assert.Equal(t, 7, created.Combatants[0].HP)
	assert.Equal(t, 1, created.Combatants[0].Defense)
	assert.Equal(t, "char-1", created.Combatants[0].PlayerCharacterID)
	assert.NotEqual(t, "char-1", created.Combatants[0].ID)

	combatantID := created.Combatants[0].ID
	_, applied, err := svc.ApplyDamage(t.Context(), combatantID, domain.DamagePhysical, domain.BodyTorso, 4)
	require.NoError(t, err)
	assert.Equal(t, 4, applied)

	players, err := svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	require.Len(t, players, 1)
	assert.Equal(t, 3, players[0].Character.HP)

	_, healed, err := svc.Heal(t.Context(), combatantID, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, healed)

	players, err = svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	require.Len(t, players, 1)
	assert.Equal(t, 5, players[0].Character.HP)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 1)
	assert.Equal(t, 5, enc.Combatants[0].HP)
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

func TestApplyDamageRespectsImmunity(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Ghoul", Initiative: 8, Side: domain.SideNPC, HP: 9, ImmunePoison: true},
	})
	require.NoError(t, err)

	_, applied, err := svc.ApplyDamage(t.Context(), "n1", domain.DamagePoison, domain.BodyHead, 99)
	require.NoError(t, err)
	assert.Equal(t, 0, applied)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 9, enc.Combatants[0].HP)
	assert.False(t, enc.Combatants[0].Defeated)
}

func TestHealPersistsAndCanRevive(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 2, MaxHP: 8},
	})
	require.NoError(t, err)

	_, _, err = svc.ApplyDamage(t.Context(), "n1", domain.DamagePhysical, domain.BodyTorso, 6)
	require.NoError(t, err)

	_, healed, err := svc.Heal(t.Context(), "n1", 5)
	require.NoError(t, err)
	assert.Equal(t, 5, healed)

	enc, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 5, enc.Combatants[0].HP)
	assert.False(t, enc.Combatants[0].Defeated)
}

func TestEncounterLogsArePersistentAndIncludeRound(t *testing.T) {
	svc := newSQLiteService(t)
	created, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Player", Initiative: 10, Side: domain.SideParty, HP: 10},
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 6},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	_, _, err = svc.ApplyDamage(t.Context(), "n1", domain.DamagePhysical, domain.BodyLeftArm, 3)
	require.NoError(t, err)
	_, _, err = svc.Heal(t.Context(), "n1", 2)
	require.NoError(t, err)

	logs, err := svc.ListEncounterLogs(t.Context(), created.ID)
	require.NoError(t, err)
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[0].Message, "Heal")
	assert.False(t, logs[0].CreatedAt.IsZero())

	hasTurnAdvanced := false
	for _, l := range logs {
		assert.GreaterOrEqual(t, l.Round, 1)
		if strings.Contains(l.Message, "Turn advanced") {
			hasTurnAdvanced = true
		}
	}
	assert.True(t, hasTurnAdvanced, "expected at least one turn advance log entry")
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

func TestCreateCampaignRejectsZeroStartDate(t *testing.T) {
	svc := newSQLiteService(t)

	_, err := svc.CreateCampaign(t.Context(), "camp-1", "Bad Date", time.Time{}, []domain.NewCampaignPlayer{
		{
			PlayerName: "June",
			Character: domain.Combatant{
				Name:       "Vault Dweller",
				Level:      1,
				Initiative: 5,
				HP:         8,
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign start date is required")
}

func TestAddPartyAPSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
			},
			Resources: domain.Resources{
				PartyAP:  0,
				GMThreat: 0,
			},
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, err := svc.AddPartyAP(t.Context(), 2)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 2, updated.Resources.PartyAP)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 2, persisted.Resources.PartyAP, "state change is persisted even when log write fails")
}

func TestAddPartyAPLogsWhenLogWriteFails(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
			},
			Resources: domain.Resources{
				PartyAP:  0,
				GMThreat: 0,
			},
		},
		appendErr: errors.New("append log failed"),
	}

	logEntries := make([]string, 0, 1)
	svc := NewServiceWithLogf(repo, func(format string, args ...any) {
		logEntries = append(logEntries, fmt.Sprintf(format, args...))
	})

	updated, err := svc.AddPartyAP(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotEmpty(t, logEntries)
	assert.Contains(t, logEntries[0], "non-critical side effect failed")
	assert.Contains(t, logEntries[0], "side_effect=audit.append_encounter_log")
	assert.Contains(t, logEntries[0], "failures=1")
	assert.Contains(t, logEntries[0], "encounter_id=enc-1")
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("audit.append_encounter_log"))
}

func TestRunNonCriticalSideEffectLogsErrorAndDoesNotBubbleUp(t *testing.T) {
	logEntries := make([]string, 0, 1)
	svc := NewServiceWithLogf(nil, func(format string, args ...any) {
		logEntries = append(logEntries, fmt.Sprintf(format, args...))
	})

	svc.runNonCriticalSideEffect(sideEffectCategoryTelemetry, "demo_side_effect", func() error {
		return errors.New("boom")
	})

	require.Len(t, logEntries, 1)
	assert.Contains(t, logEntries[0], "non-critical side effect failed")
	assert.Contains(t, logEntries[0], "side_effect=telemetry.demo_side_effect")
	assert.Contains(t, logEntries[0], "failures=1")
	assert.Contains(t, logEntries[0], "boom")
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("telemetry.demo_side_effect"))
}

func TestRunNonCriticalSideEffectCountsFailuresPerType(t *testing.T) {
	svc := NewServiceWithLogf(nil, func(string, ...any) {})
	for range 2 {
		svc.runNonCriticalSideEffect(sideEffectCategoryNotifications, "dispatch_update", func() error {
			return errors.New("network timeout")
		})
	}
	svc.runNonCriticalSideEffect(sideEffectCategoryAudit, "append_encounter_log", func() error {
		return errors.New("write failed")
	})

	assert.Equal(t, uint64(2), svc.sideEffectFailureCount("notifications.dispatch_update"))
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("audit.append_encounter_log"))
}

func TestAdvanceTurnSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
				{ID: "c2", Name: "Two", Initiative: 8, Active: false},
			},
			TurnIndex: 0,
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, err := svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 1, updated.TurnIndex)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 1, persisted.TurnIndex)
}

func TestApplyDamageSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "n1", Name: "Raider", Initiative: 8, HP: 10, MaxHP: 10, Side: domain.SideNPC},
			},
			TurnIndex: 0,
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, applied, err := svc.ApplyDamage(t.Context(), "n1", domain.DamagePoison, domain.BodyHead, 3)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 3, applied)
	assert.Equal(t, 7, updated.Combatants[0].HP)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 7, persisted.Combatants[0].HP)
}

func TestServiceAppliesDefaultOperationTimeoutWhenDeadlineIsMissing(t *testing.T) {
	repo := &contextCaptureRepo{
		logFailingRepo: &logFailingRepo{
			encounter: &domain.Encounter{ID: "enc-1", Round: 1},
		},
	}
	svc := NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 200*time.Millisecond)

	_, err := svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.True(t, repo.gotHasDeadline, "service should apply operation timeout when caller has no deadline")
}

func TestServiceKeepsCallerDeadlineWhenAlreadySet(t *testing.T) {
	repo := &contextCaptureRepo{
		logFailingRepo: &logFailingRepo{
			encounter: &domain.Encounter{ID: "enc-1", Round: 1},
		},
	}
	svc := NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 5*time.Second)

	parentCtx, cancel := context.WithTimeout(t.Context(), 120*time.Millisecond)
	defer cancel()
	parentDeadline, ok := parentCtx.Deadline()
	require.True(t, ok)

	_, err := svc.GetEncounter(parentCtx)
	require.NoError(t, err)
	require.True(t, repo.gotHasDeadline)
	require.False(t, repo.gotDeadline.After(parentDeadline), "service should not extend caller deadline")
}

func newSQLiteService(t *testing.T) *Service {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "tracker.db")
	db, err := sqlite.OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	svc := NewService(sqlite.NewEncounterStore(db))
	_, err = svc.CreateCampaign(t.Context(), "test-campaign", "Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "char-1",
				Name:       "Vault Dweller",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 9,
				HP:         7,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)
	return svc
}

func testCampaignStartDate(t *testing.T) time.Time {
	t.Helper()
	startDate, err := domain.ParseCampaignStartDate("2026-01-01")
	require.NoError(t, err)
	return startDate
}

type logFailingRepo struct {
	encounter   *domain.Encounter
	appendErr   error
	saveCalls   int
	appendCalls int
}

func (r *logFailingRepo) Get(_ context.Context) (*domain.Encounter, error) {
	return cloneEncounter(r.encounter), nil
}

func (r *logFailingRepo) Save(_ context.Context, encounter *domain.Encounter) error {
	r.saveCalls++
	r.encounter = cloneEncounter(encounter)
	return nil
}

func (r *logFailingRepo) List(_ context.Context) ([]domain.EncounterSummary, error) { return nil, nil }

func (r *logFailingRepo) GetEncounterByID(_ context.Context, _ string) (*domain.Encounter, error) {
	return cloneEncounter(r.encounter), nil
}

func (r *logFailingRepo) UpdateEncounter(_ context.Context, _, _ string, _ []domain.Combatant) (*domain.Encounter, error) {
	return nil, nil
}

func (r *logFailingRepo) ListPartyMembers(_ context.Context) ([]domain.Combatant, error) {
	return nil, nil
}

func (r *logFailingRepo) CreateCampaign(_ context.Context, _, _ string, _ time.Time, _ []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) UpdateCampaign(_ context.Context, _, _ string, _ time.Time, _ []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) GetActiveCampaign(_ context.Context) (*domain.Campaign, error) {
	return nil, nil
}

func (r *logFailingRepo) ListCampaigns(_ context.Context) ([]domain.Campaign, error) { return nil, nil }

func (r *logFailingRepo) ListCampaignPlayers(_ context.Context, _ string) ([]domain.NewCampaignPlayer, error) {
	return nil, nil
}

func (r *logFailingRepo) ActivateCampaign(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) Activate(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) SoftDelete(_ context.Context, _ string) error { return nil }

func (r *logFailingRepo) AppendEncounterLog(_ context.Context, _ string, _ int, _ string) error {
	r.appendCalls++
	return r.appendErr
}

func (r *logFailingRepo) ListEncounterLogs(_ context.Context, _ string) ([]domain.EncounterLog, error) {
	return nil, nil
}

func cloneEncounter(src *domain.Encounter) *domain.Encounter {
	if src == nil {
		return nil
	}
	cp := *src
	cp.Combatants = append([]domain.Combatant(nil), src.Combatants...)
	return &cp
}

type contextCaptureRepo struct {
	*logFailingRepo
	gotHasDeadline bool
	gotDeadline    time.Time
}

func (r *contextCaptureRepo) Get(ctx context.Context) (*domain.Encounter, error) {
	r.gotDeadline, r.gotHasDeadline = ctx.Deadline()
	return cloneEncounter(r.encounter), nil
}
