package app

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
