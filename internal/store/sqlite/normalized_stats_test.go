package sqlite

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalResistanceStatsIncludesImmunityFlags(t *testing.T) {
	ids := testDictionaryIDs()
	actual, err := globalResistanceStats(ids, domain.ResistanceProfile{
		Global: map[domain.DamageType]domain.Resistance{
			domain.DamagePhysical:  {Value: 1, Immune: true},
			domain.DamageEnergy:    {Value: 2},
			domain.DamageRadiation: {Value: 3, Immune: true},
			domain.DamagePoison:    {Value: 4},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, []resistanceGlobalStat{
		{ids.damageTypes[domain.DamagePhysical], 0, 1},
		{ids.damageTypes[domain.DamageEnergy], 0, 0},
		{ids.damageTypes[domain.DamageRadiation], 0, 1},
		{ids.damageTypes[domain.DamagePoison], 4, 0},
	}, actual)
}

func TestResistanceStatsByLocationPreservesDamageAndBodyLocationOrder(t *testing.T) {
	ids := testDictionaryIDs()
	actual, err := resistanceStatsByLocation(ids, domain.ResistanceProfile{
		ByLocation: map[domain.DamageType]map[domain.BodyLocation]int{
			domain.DamagePhysical: {
				domain.BodyHead:     1,
				domain.BodyTorso:    2,
				domain.BodyLeftArm:  3,
				domain.BodyRightArm: 4,
				domain.BodyLeftLeg:  5,
				domain.BodyRightLeg: 6,
			},
			domain.DamageEnergy: {
				domain.BodyHead:     7,
				domain.BodyTorso:    8,
				domain.BodyLeftArm:  9,
				domain.BodyRightArm: 10,
				domain.BodyLeftLeg:  11,
				domain.BodyRightLeg: 12,
			},
			domain.DamageRadiation: {
				domain.BodyHead:     13,
				domain.BodyTorso:    14,
				domain.BodyLeftArm:  15,
				domain.BodyRightArm: 16,
				domain.BodyLeftLeg:  17,
				domain.BodyRightLeg: 18,
			},
		},
	})
	require.NoError(t, err)

	require.Len(t, actual, 18)
	assert.Equal(t, []resistanceByLocationStat{
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyHead], 1},
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyTorso], 2},
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyLeftArm], 3},
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyRightArm], 4},
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyLeftLeg], 5},
		{ids.damageTypes[domain.DamagePhysical], ids.bodyLocations[domain.BodyRightLeg], 6},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyHead], 7},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyTorso], 8},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyLeftArm], 9},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyRightArm], 10},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyLeftLeg], 11},
		{ids.damageTypes[domain.DamageEnergy], ids.bodyLocations[domain.BodyRightLeg], 12},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyHead], 13},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyTorso], 14},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyLeftArm], 15},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyRightArm], 16},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyLeftLeg], 17},
		{ids.damageTypes[domain.DamageRadiation], ids.bodyLocations[domain.BodyRightLeg], 18},
	}, actual)
}

func TestNormalizedStatsFailsWhenDictionaryIDsAreMissing(t *testing.T) {
	ids := testDictionaryIDs()
	delete(ids.damageTypes, domain.DamagePoison)

	_, err := globalResistanceStats(ids, domain.ResistanceProfile{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown damage type id")
}

func testDictionaryIDs() dictionaryIDs {
	return dictionaryIDs{
		damageTypes: map[domain.DamageType]int64{
			domain.DamagePhysical:  101,
			domain.DamageEnergy:    102,
			domain.DamageRadiation: 103,
			domain.DamagePoison:    104,
		},
		bodyLocations: map[domain.BodyLocation]int64{
			domain.BodyHead:     201,
			domain.BodyTorso:    202,
			domain.BodyLeftArm:  203,
			domain.BodyRightArm: 204,
			domain.BodyLeftLeg:  205,
			domain.BodyRightLeg: 206,
		},
	}
}
