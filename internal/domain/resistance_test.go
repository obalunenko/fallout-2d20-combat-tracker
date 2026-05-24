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

func TestCombatantProfileRoundTrip(t *testing.T) {
	t.Parallel()

	profile := CombatantProfile{
		Stats: CombatantStats{
			TorsoOnly:  true,
			Level:      3,
			XP:         50,
			Initiative: 7,
			HP:         9,
			MaxHP:      10,
			Defense:    2,
		},
		Resistance: ResistanceProfile{
			Global: map[DamageType]Resistance{
				DamagePhysical: {Immune: true},
				DamagePoison:   {Value: 4},
			},
			ByLocation: map[DamageType]map[BodyLocation]int{
				DamagePhysical: {
					BodyTorso: 6,
				},
			},
		},
	}

	var combatant Combatant
	combatant.SetProfile(profile)
	actual := combatant.Profile()

	assert.Equal(t, profile.Stats, actual.Stats)
	resistance, immune, err := actual.Resistance.GlobalResistance(DamagePhysical)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)
	assert.True(t, immune)
	resistance, _, err = actual.Resistance.GlobalResistance(DamagePoison)
	require.NoError(t, err)
	assert.Equal(t, 4, resistance)
	locationResistance, err := actual.Resistance.LocationResistance(DamagePhysical, BodyTorso)
	require.NoError(t, err)
	assert.Equal(t, 6, locationResistance)
}

func TestResistanceProfileSettersInitializeMapsAndClone(t *testing.T) {
	t.Parallel()

	profile := NewResistanceProfile()
	require.NoError(t, profile.SetGlobalResistance(DamagePhysical, Resistance{Value: 99, Immune: true}))
	require.NoError(t, profile.SetGlobalResistance(DamagePoison, Resistance{Value: 4, Immune: true}))
	require.NoError(t, profile.SetLocationResistance(DamageEnergy, BodyLeftArm, 7))

	resistance, immune, err := profile.GlobalResistance(DamagePhysical)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)
	assert.True(t, immune)

	resistance, immune, err = profile.GlobalResistance(DamagePoison)
	require.NoError(t, err)
	assert.Equal(t, 4, resistance)
	assert.True(t, immune)

	locationResistance, err := profile.LocationResistance(DamageEnergy, BodyLeftArm)
	require.NoError(t, err)
	assert.Equal(t, 7, locationResistance)

	clone := profile.Clone()
	require.NoError(t, clone.SetLocationResistance(DamageEnergy, BodyLeftArm, 9))

	locationResistance, err = profile.LocationResistance(DamageEnergy, BodyLeftArm)
	require.NoError(t, err)
	assert.Equal(t, 7, locationResistance)
	locationResistance, err = clone.LocationResistance(DamageEnergy, BodyLeftArm)
	require.NoError(t, err)
	assert.Equal(t, 9, locationResistance)
}

func TestResistanceProfileSettersValidateKeys(t *testing.T) {
	t.Parallel()

	profile := NewResistanceProfile()

	require.Error(t, profile.SetGlobalResistance(DamageType("fire"), Resistance{}))
	require.Error(t, profile.SetLocationResistance(DamageEnergy, BodyLocation("wing"), 1))
	require.Error(t, profile.SetLocationResistance(DamagePoison, BodyTorso, 1))
}

func TestResistanceProfileEffectiveResistance(t *testing.T) {
	t.Parallel()

	profile := NewResistanceProfile()
	require.NoError(t, profile.SetGlobalResistance(DamagePhysical, Resistance{Immune: true}))
	require.NoError(t, profile.SetGlobalResistance(DamagePoison, Resistance{Value: 3}))
	require.NoError(t, profile.SetLocationResistance(DamagePhysical, BodyHead, 2))
	require.NoError(t, profile.SetLocationResistance(DamagePhysical, BodyTorso, 5))

	resistance, immune, err := profile.EffectiveResistance(DamagePhysical, BodyHead, false)
	require.NoError(t, err)
	assert.Equal(t, 2, resistance)
	assert.True(t, immune)

	resistance, immune, err = profile.EffectiveResistance(DamagePhysical, BodyHead, true)
	require.NoError(t, err)
	assert.Equal(t, 5, resistance)
	assert.True(t, immune)

	resistance, immune, err = profile.EffectiveResistance(DamagePoison, BodyHead, false)
	require.NoError(t, err)
	assert.Equal(t, 3, resistance)
	assert.False(t, immune)
}

func TestCombatantResistanceProfileAdapters(t *testing.T) {
	t.Parallel()

	var combatant Combatant
	require.NoError(t, combatant.SetGlobalResistance(DamageEnergy, 12, true))
	require.NoError(t, combatant.SetGlobalResistance(DamagePoison, 4, false))
	require.NoError(t, combatant.SetLocationResistance(DamageEnergy, BodyRightLeg, 8))

	resistance, immune, err := combatant.GlobalResistance(DamageEnergy)
	require.NoError(t, err)
	assert.Equal(t, 0, resistance)
	assert.True(t, immune)

	resistance, immune, err = combatant.GlobalResistance(DamagePoison)
	require.NoError(t, err)
	assert.Equal(t, 4, resistance)
	assert.False(t, immune)

	locationResistance, err := combatant.LocationResistance(DamageEnergy, BodyRightLeg)
	require.NoError(t, err)
	assert.Equal(t, 8, locationResistance)

	assert.Equal(t, 0, combatant.ResistEnergy)
	assert.True(t, combatant.ImmuneEnergy)
	assert.Equal(t, 4, combatant.ResistPoison)
	assert.Equal(t, 8, combatant.ResistEnergyRightLeg)
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

func TestValidateResistanceProfileChecksMeaningfulResistanceValues(t *testing.T) {
	t.Parallel()

	err := ValidateResistanceProfile(ResistanceProfile{
		Global: map[DamageType]Resistance{
			DamagePoison: {Value: -1},
		},
	})
	require.Error(t, err)

	err = ValidateResistanceProfile(ResistanceProfile{
		Global: map[DamageType]Resistance{
			DamagePhysical: {Value: -1},
		},
		ByLocation: map[DamageType]map[BodyLocation]int{
			DamagePhysical: {
				BodyTorso: 1,
			},
		},
	})
	require.NoError(t, err)
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
