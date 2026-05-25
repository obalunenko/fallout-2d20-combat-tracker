package app

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndListMonsterTemplates(t *testing.T) {
	svc := newSQLiteService(t)

	saved, err := svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{
		{
			Name: "Raider", Side: domain.SideParty, Level: 2, XP: 60, Initiative: 8, HP: 10, MaxHP: 10, Defense: 1,
			ResistPhysicalTorso: 2, ResistEnergyTorso: 1, ResistPoison: 3, ImmuneRadiation: true,
		},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.NotEmpty(t, saved[0].ID)
	assert.Equal(t, domain.SideNPC, saved[0].Side)

	_, err = svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{
		{Name: "raider", Level: 3, XP: 80, Initiative: 9, HP: 12, MaxHP: 12, Defense: 2, ResistPoison: 4},
	})
	require.NoError(t, err)

	monsters, err := svc.ListMonsterTemplates(t.Context())
	require.NoError(t, err)
	require.Len(t, monsters, 1)
	assert.Equal(t, "raider", monsters[0].Name)
	assert.Equal(t, 3, monsters[0].Level)
	assert.Equal(t, 80, monsters[0].XP)
	assert.Equal(t, 9, monsters[0].Initiative)
	assert.Equal(t, 12, monsters[0].HP)
	assert.Equal(t, 2, monsters[0].Defense)
	assert.Equal(t, 4, monsters[0].ResistPoison)
	assert.Equal(t, domain.SideNPC, monsters[0].Side)
}
