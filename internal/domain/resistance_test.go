package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombatantStatsRoundTrip(t *testing.T) {
	t.Parallel()

	stats := CombatantStats{
		TorsoOnly:  true,
		Level:      4,
		XP:         90,
		Initiative: 8,
		HP:         11,
		MaxHP:      12,
		Defense:    3,
	}
	var combatant Combatant
	combatant.SetStats(stats)

	assert.Equal(t, stats, combatant.Stats())
}

func TestResistanceProfileReadsCombatantFields(t *testing.T) {
	t.Parallel()

	combatant := Combatant{
		ResistPhysical:          2,
		ResistEnergy:            3,
		ResistRadiation:         4,
		ResistPoison:            5,
		ResistPhysicalTorso:     6,
		ResistEnergyLeftArm:     7,
		ResistRadiationRightLeg: 8,
		ImmuneEnergy:            true,
		ImmunePoison:            true,
	}

	profile := combatant.ResistanceProfile()

	resistance, immune, err := profile.GlobalResistance(DamageEnergy)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)
	assert.True(t, immune)

	resistance, immune, err = profile.GlobalResistance(DamagePoison)
	require.NoError(t, err)
	assert.Equal(t, 5, resistance)
	assert.True(t, immune)

	locationResistance, err := profile.LocationResistance(DamagePhysical, BodyTorso)
	require.NoError(t, err)
	assert.Equal(t, 6, locationResistance)

	locationResistance, err = profile.LocationResistance(DamageEnergy, BodyLeftArm)
	require.NoError(t, err)
	assert.Equal(t, 7, locationResistance)

	locationResistance, err = profile.LocationResistance(DamageRadiation, BodyRightLeg)
	require.NoError(t, err)
	assert.Equal(t, 8, locationResistance)
}

func TestResistanceProfileNormalizesNegativeValues(t *testing.T) {
	t.Parallel()

	combatant := Combatant{
		ResistPhysical:          -1,
		ResistEnergyTorso:       -2,
		ResistRadiationRightLeg: 0,
	}

	profile := combatant.ResistanceProfile()
	resistance, _, err := profile.GlobalResistance(DamagePhysical)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)

	locationResistance, err := profile.LocationResistance(DamageEnergy, BodyTorso)
	require.NoError(t, err)
	assert.Equal(t, 0, locationResistance)
}

func TestCombatantHasNegativeResistance(t *testing.T) {
	t.Parallel()

	assert.False(t, Combatant{ResistPhysical: -1, ResistEnergyTorso: 2}.HasNegativeResistance())
	assert.True(t, Combatant{ResistPoison: -1}.HasNegativeResistance())
	assert.True(t, Combatant{ResistRadiationRightLeg: -1}.HasNegativeResistance())
}

func TestSetResistanceProfileWritesCombatantFields(t *testing.T) {
	t.Parallel()

	var combatant Combatant
	combatant.SetResistanceProfile(ResistanceProfile{
		Global: map[DamageType]Resistance{
			DamagePhysical:  {Value: 2, Immune: true},
			DamageEnergy:    {Value: 3},
			DamageRadiation: {Value: 4, Immune: true},
			DamagePoison:    {Value: 5},
		},
		ByLocation: map[DamageType]map[BodyLocation]int{
			DamagePhysical: {
				BodyHead:  1,
				BodyTorso: 2,
			},
			DamageEnergy: {
				BodyLeftArm: 3,
			},
			DamageRadiation: {
				BodyRightLeg: 4,
			},
		},
	})

	assert.Equal(t, 0, combatant.ResistPhysical)
	assert.True(t, combatant.ImmunePhysical)
	assert.Equal(t, 0, combatant.ResistEnergy)
	assert.False(t, combatant.ImmuneEnergy)
	assert.Equal(t, 0, combatant.ResistRadiation)
	assert.True(t, combatant.ImmuneRadiation)
	assert.Equal(t, 5, combatant.ResistPoison)
	assert.False(t, combatant.ImmunePoison)
	assert.Equal(t, 1, combatant.ResistPhysicalHead)
	assert.Equal(t, 2, combatant.ResistPhysicalTorso)
	assert.Equal(t, 3, combatant.ResistEnergyLeftArm)
	assert.Equal(t, 4, combatant.ResistRadiationRightLeg)
}

func TestResistanceProfileValidation(t *testing.T) {
	t.Parallel()

	profile := ResistanceProfile{}

	_, _, err := profile.GlobalResistance(DamageType("invalid"))
	require.Error(t, err)

	_, err = profile.LocationResistance(DamagePhysical, BodyLocation("invalid"))
	require.Error(t, err)

	resistance, err := profile.LocationResistance(DamagePoison, BodyHead)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)
}
