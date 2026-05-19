package app

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEncounterRejectsEmptyCombatants(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "test", nil)
	require.Error(t, err)
}

func TestCreateEncounterPersistsAndGetReturnsSorted(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "test", []domain.Combatant{
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

	enc, err := svc.GetEncounter()
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
	created, err := svc.CreateEncounter("", "test", []domain.Combatant{
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
	_, err := svc.CreateEncounter("enc-1", "test", []domain.Combatant{
		{ID: "c1", Name: "Alpha", Initiative: 10},
		{ID: "c2", Name: "Beta", Initiative: 8},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn()
	require.NoError(t, err)

	enc, err := svc.GetEncounter()
	require.NoError(t, err)

	assert.Equal(t, 1, enc.TurnIndex)
	assert.True(t, enc.Combatants[1].Active)
}

func TestResourceCommands(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "test", []domain.Combatant{{ID: "c1", Name: "Alpha", Initiative: 10}})
	require.NoError(t, err)

	_, err = svc.AddPartyAP(2)
	require.NoError(t, err)
	_, err = svc.SpendPartyAP(1)
	require.NoError(t, err)
	_, err = svc.SpendPartyAP(10)
	require.Error(t, err)

	_, err = svc.AddThreat(2)
	require.NoError(t, err)
	_, err = svc.SpendThreat(1)
	require.NoError(t, err)
	_, err = svc.SpendThreat(10)
	require.Error(t, err)

	enc, err := svc.GetEncounter()
	require.NoError(t, err)
	assert.Equal(t, 1, enc.Resources.PartyAP)
	assert.Equal(t, 1, enc.Resources.GMThreat)
}

func TestListAndActivateEncounter(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)
	_, err = svc.CreateEncounter("enc-2", "Bravo", []domain.Combatant{{ID: "c2", Name: "Two", Initiative: 8}})
	require.NoError(t, err)

	summaries, err := svc.ListEncounters()
	require.NoError(t, err)
	require.Len(t, summaries, 2)
	assert.Equal(t, "enc-2", summaries[0].ID)

	active, err := svc.GetEncounter()
	require.NoError(t, err)
	assert.Equal(t, "enc-2", active.ID)

	active, err = svc.ActivateEncounter("enc-1")
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)
}

func TestActivateEncounterNotFound(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)

	_, err = svc.ActivateEncounter("enc-missing")
	require.ErrorIs(t, err, domain.ErrEncounterNotFound)
}

func TestRestartEncounterResetsRoundAndResources(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "c1", Name: "One", Initiative: 10},
		{ID: "c2", Name: "Two", Initiative: 8},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn()
	require.NoError(t, err)
	_, err = svc.AddPartyAP(3)
	require.NoError(t, err)
	_, err = svc.AddThreat(2)
	require.NoError(t, err)

	restarted, err := svc.RestartEncounter("enc-1")
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
}

func TestSoftDeleteEncounterHidesFromListAndActivation(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{{ID: "c1", Name: "One", Initiative: 10}})
	require.NoError(t, err)
	_, err = svc.CreateEncounter("enc-2", "Bravo", []domain.Combatant{{ID: "c2", Name: "Two", Initiative: 8}})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteEncounter("enc-2"))

	summaries, err := svc.ListEncounters()
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "enc-1", summaries[0].ID)

	active, err := svc.GetEncounter()
	require.NoError(t, err)
	assert.Equal(t, "enc-1", active.ID)

	_, err = svc.ActivateEncounter("enc-2")
	require.ErrorIs(t, err, domain.ErrEncounterNotFound)
}

func TestApplyDamagePersistsHPAndDefeated(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Player", Initiative: 10, Side: domain.SideParty, HP: 12, ResistPhysical: 2},
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 5, ResistEnergy: 1},
	})
	require.NoError(t, err)

	_, applied, err := svc.ApplyDamage("n1", domain.DamageEnergy, 9)
	require.NoError(t, err)
	assert.Equal(t, 8, applied)

	enc, err := svc.GetEncounter()
	require.NoError(t, err)
	require.Len(t, enc.Combatants, 2)
	assert.Equal(t, -3, enc.Combatants[1].HP)
	assert.True(t, enc.Combatants[1].Defeated)
}

func TestApplyDamageRespectsImmunity(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Ghoul", Initiative: 8, Side: domain.SideNPC, HP: 9, ImmuneRadiation: true},
	})
	require.NoError(t, err)

	_, applied, err := svc.ApplyDamage("n1", domain.DamageRadiation, 99)
	require.NoError(t, err)
	assert.Equal(t, 0, applied)

	enc, err := svc.GetEncounter()
	require.NoError(t, err)
	assert.Equal(t, 9, enc.Combatants[0].HP)
	assert.False(t, enc.Combatants[0].Defeated)
}

func TestHealPersistsAndCanRevive(t *testing.T) {
	svc := newSQLiteService(t)
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 2},
	})
	require.NoError(t, err)

	_, _, err = svc.ApplyDamage("n1", domain.DamagePhysical, 6)
	require.NoError(t, err)

	_, healed, err := svc.Heal("n1", 5)
	require.NoError(t, err)
	assert.Equal(t, 5, healed)

	enc, err := svc.GetEncounter()
	require.NoError(t, err)
	assert.Equal(t, 1, enc.Combatants[0].HP)
	assert.False(t, enc.Combatants[0].Defeated)
}

func TestEncounterLogsArePersistentAndIncludeRound(t *testing.T) {
	svc := newSQLiteService(t)
	created, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Player", Initiative: 10, Side: domain.SideParty, HP: 10},
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 6},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn()
	require.NoError(t, err)
	_, _, err = svc.ApplyDamage("n1", domain.DamagePhysical, 3)
	require.NoError(t, err)
	_, _, err = svc.Heal("n1", 2)
	require.NoError(t, err)

	logs, err := svc.ListEncounterLogs(created.ID)
	require.NoError(t, err)
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[0].Message, "Heal")
	assert.NotEmpty(t, logs[0].CreatedAt)

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
	_, err := svc.CreateEncounter("enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Roland", Initiative: 9, Side: domain.SideParty, Level: 1, HP: 7, Defense: 2},
		{ID: "n1", Name: "Raider", Initiative: 7, Side: domain.SideNPC, Level: 1, XP: 30, HP: 6, Defense: 1},
	})
	require.NoError(t, err)
	_, err = svc.CreateEncounter("enc-2", "Bravo", []domain.Combatant{
		{ID: "p2", Name: "Piper", Initiative: 8, Side: domain.SideParty, Level: 2, HP: 8, Defense: 3},
		{ID: "p3", Name: "Roland", Initiative: 11, Side: domain.SideParty, Level: 3, HP: 9, Defense: 4},
	})
	require.NoError(t, err)

	party, err := svc.ListPartyMembers()
	require.NoError(t, err)
	require.Len(t, party, 1)
	assert.Equal(t, "Vault Dweller", party[0].Name)
	assert.Equal(t, 1, party[0].Level)
	assert.Equal(t, 9, party[0].Initiative)
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
	_, err = svc.CreateCampaign("test-campaign", "Test Campaign", "2026-01-01", []domain.NewCampaignPlayer{
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
