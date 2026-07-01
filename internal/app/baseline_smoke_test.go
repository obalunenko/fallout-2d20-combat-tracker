package app

import (
	"path/filepath"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaselineSmokeFlowPersistsCampaignEncounterResourcesActionsAndLogs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "smoke.db")
	db, err := sqlite.OpenAndMigrate(dbPath)
	require.NoError(t, err)

	svc := NewService(sqlite.NewEncounterStore(db))
	_, err = svc.CreateCampaign(t.Context(), "smoke-campaign", "Smoke Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "June",
			Character: domain.Combatant{
				ID:         "pc-june",
				Name:       "Talia",
				Level:      2,
				Initiative: 12,
				HP:         10,
				MaxHP:      10,
				Defense:    1,
			},
		},
		{
			PlayerName: "Kai",
			Character: domain.Combatant{
				ID:         "pc-kai",
				Name:       "Mack",
				Level:      1,
				Initiative: 9,
				HP:         8,
				MaxHP:      8,
				Defense:    2,
			},
		},
	})
	require.NoError(t, err)

	savedTemplates, err := svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{
		{
			ID:         "template-raider",
			Name:       "Raider",
			Level:      1,
			XP:         30,
			Initiative: 8,
			HP:         6,
			MaxHP:      6,
			Defense:    1,
		},
		{
			ID:         "template-raider-duplicate",
			Name:       " raider ",
			Level:      3,
			XP:         90,
			Initiative: 5,
			HP:         12,
			MaxHP:      12,
			Defense:    3,
		},
	})
	require.NoError(t, err)
	require.Len(t, savedTemplates, 1)

	templates, err := svc.ListMonsterTemplates(t.Context())
	require.NoError(t, err)
	require.Len(t, templates, 1)
	assert.Equal(t, "Raider", templates[0].Name)

	created, err := svc.CreateEncounter(t.Context(), "smoke-encounter", "Roadside Ambush", []domain.Combatant{
		{
			ID:                "pc-june",
			PlayerCharacterID: "pc-june",
			Name:              "Draft Copy",
			Side:              domain.SideParty,
			Level:             99,
			Initiative:        1,
			HP:                99,
			MaxHP:             99,
			Defense:           9,
		},
		{
			ID:         "npc-raider-1",
			Name:       templates[0].Name,
			Side:       domain.SideNPC,
			Level:      templates[0].Level,
			XP:         templates[0].XP,
			Initiative: templates[0].Initiative,
			HP:         templates[0].HP,
			MaxHP:      templates[0].MaxHP,
			Defense:    templates[0].Defense,
		},
	})
	require.NoError(t, err)
	require.Len(t, created.Combatants, 2)
	assert.Equal(t, "Talia", created.Combatants[0].Name)
	assert.Equal(t, domain.SideParty, created.Combatants[0].Side)
	assert.Equal(t, "Raider", created.Combatants[1].Name)

	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	withAP, err := svc.AddPartyAP(t.Context(), 10)
	require.NoError(t, err)
	assert.Equal(t, domain.MaxPartyAP, withAP.Resources.PartyAP)

	withAP, err = svc.SpendPartyAP(t.Context(), 2)
	require.NoError(t, err)
	assert.Equal(t, 4, withAP.Resources.PartyAP)

	withThreat, err := svc.AddThreat(t.Context(), 3)
	require.NoError(t, err)
	assert.Equal(t, 3, withThreat.Resources.GMThreat)

	withThreat, err = svc.SpendThreat(t.Context(), 1)
	require.NoError(t, err)
	assert.Equal(t, 2, withThreat.Resources.GMThreat)

	_, applied, err := svc.ApplyDamage(t.Context(), "npc-raider-1", domain.DamagePhysical, domain.BodyTorso, 99)
	require.NoError(t, err)
	assert.Equal(t, 99, applied)

	_, healed, err := svc.Heal(t.Context(), "npc-raider-1", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, healed)

	logs, err := svc.ListEncounterLogs(t.Context(), created.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(logs), 7)
	assert.Contains(t, logs[0].Message, "Heal")

	require.NoError(t, db.Close())

	reopenedDB, err := sqlite.OpenAndMigrate(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, reopenedDB.Close())
	})
	reopened := NewService(sqlite.NewEncounterStore(reopenedDB))

	activeCampaign, err := reopened.GetActiveCampaign(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Smoke Campaign", activeCampaign.Name)
	assert.Equal(t, 4, activeCampaign.Resources.PartyAP)
	assert.Equal(t, 2, activeCampaign.Resources.GMThreat)

	persisted, err := reopened.GetEncounter(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Roadside Ambush", persisted.Name)
	assert.Equal(t, 4, persisted.Resources.PartyAP)
	assert.Equal(t, 2, persisted.Resources.GMThreat)

	npc := requireCombatantByID(t, persisted, "npc-raider-1")
	assert.Equal(t, 3, npc.HP)
	assert.False(t, npc.Defeated)

	reopenedLogs, err := reopened.ListEncounterLogs(t.Context(), persisted.ID)
	require.NoError(t, err)
	require.Len(t, reopenedLogs, len(logs))
	assert.Contains(t, reopenedLogs[0].Message, "Heal")
}

func requireCombatantByID(t *testing.T, enc *domain.Encounter, id string) domain.Combatant {
	t.Helper()
	require.NotNil(t, enc)
	for _, combatant := range enc.Combatants {
		if combatant.ID == id {
			return combatant
		}
	}
	require.Failf(t, "combatant not found", "id=%s", id)
	return domain.Combatant{}
}
