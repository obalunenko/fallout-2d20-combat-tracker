package sqlite

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreListMonsterTemplatesReadsNormalizedResistances(t *testing.T) {
	store := newTestStore(t)

	_, err := store.UpsertMonsterTemplate(t.Context(), domain.Combatant{
		Name:                    "Sentry Bot",
		Level:                   4,
		XP:                      120,
		Initiative:              8,
		HP:                      14,
		MaxHP:                   14,
		Defense:                 3,
		ResistPhysical:          2,
		ResistEnergy:            3,
		ResistRadiation:         4,
		ResistPoison:            5,
		ResistPhysicalTorso:     6,
		ResistEnergyLeftArm:     7,
		ResistRadiationRightLeg: 8,
		ImmunePhysical:          true,
		ImmunePoison:            true,
	})
	require.NoError(t, err)

	monsters, err := store.ListMonsterTemplates(t.Context())
	require.NoError(t, err)
	require.Len(t, monsters, 1)

	assert.Equal(t, "Sentry Bot", monsters[0].Name)
	assert.Equal(t, 0, monsters[0].ResistPhysical)
	assert.Equal(t, 0, monsters[0].ResistEnergy)
	assert.Equal(t, 0, monsters[0].ResistRadiation)
	assert.Equal(t, 5, monsters[0].ResistPoison)
	assert.Equal(t, 6, monsters[0].ResistPhysicalTorso)
	assert.Equal(t, 7, monsters[0].ResistEnergyLeftArm)
	assert.Equal(t, 8, monsters[0].ResistRadiationRightLeg)
	assert.True(t, monsters[0].ImmunePhysical)
	assert.False(t, monsters[0].ImmuneEnergy)
	assert.False(t, monsters[0].ImmuneRadiation)
	assert.True(t, monsters[0].ImmunePoison)
}
