package sqlite

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalResistanceStatsIncludesImmunityFlags(t *testing.T) {
	actual := globalResistanceStats(domain.Combatant{
		ResistPhysical:  1,
		ResistEnergy:    2,
		ResistRadiation: 3,
		ResistPoison:    4,
		ImmunePhysical:  true,
		ImmuneRadiation: true,
	})

	assert.Equal(t, []resistanceGlobalStat{
		{damageTypePhysical, 1, 1},
		{damageTypeEnergy, 2, 0},
		{damageTypeRadiation, 3, 1},
		{damageTypePoison, 4, 0},
	}, actual)
}

func TestResistanceStatsByLocationPreservesDamageAndBodyLocationOrder(t *testing.T) {
	actual := resistanceStatsByLocation(domain.Combatant{
		ResistPhysicalHead:      1,
		ResistPhysicalTorso:     2,
		ResistPhysicalLeftArm:   3,
		ResistPhysicalRightArm:  4,
		ResistPhysicalLeftLeg:   5,
		ResistPhysicalRightLeg:  6,
		ResistEnergyHead:        7,
		ResistEnergyTorso:       8,
		ResistEnergyLeftArm:     9,
		ResistEnergyRightArm:    10,
		ResistEnergyLeftLeg:     11,
		ResistEnergyRightLeg:    12,
		ResistRadiationHead:     13,
		ResistRadiationTorso:    14,
		ResistRadiationLeftArm:  15,
		ResistRadiationRightArm: 16,
		ResistRadiationLeftLeg:  17,
		ResistRadiationRightLeg: 18,
	})

	require.Len(t, actual, 18)
	assert.Equal(t, []resistanceByLocationStat{
		{damageTypePhysical, bodyLocationHead, 1},
		{damageTypePhysical, bodyLocationTorso, 2},
		{damageTypePhysical, bodyLocationLeftArm, 3},
		{damageTypePhysical, bodyLocationRightArm, 4},
		{damageTypePhysical, bodyLocationLeftLeg, 5},
		{damageTypePhysical, bodyLocationRightLeg, 6},
		{damageTypeEnergy, bodyLocationHead, 7},
		{damageTypeEnergy, bodyLocationTorso, 8},
		{damageTypeEnergy, bodyLocationLeftArm, 9},
		{damageTypeEnergy, bodyLocationRightArm, 10},
		{damageTypeEnergy, bodyLocationLeftLeg, 11},
		{damageTypeEnergy, bodyLocationRightLeg, 12},
		{damageTypeRadiation, bodyLocationHead, 13},
		{damageTypeRadiation, bodyLocationTorso, 14},
		{damageTypeRadiation, bodyLocationLeftArm, 15},
		{damageTypeRadiation, bodyLocationRightArm, 16},
		{damageTypeRadiation, bodyLocationLeftLeg, 17},
		{damageTypeRadiation, bodyLocationRightLeg, 18},
	}, actual)
}
