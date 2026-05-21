package sqlite

import (
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
	"github.com/stretchr/testify/assert"
)

func TestCombatantFromRowMapsDBFields(t *testing.T) {
	actual := combatantFromRow(dbgen.ListCombatantsByEncounterIDRow{
		ID:                                "c1",
		Name:                              "Raider",
		Side:                              string(domain.SideNPC),
		Level:                             4,
		Xp:                                80,
		Initiative:                        12,
		Hp:                                17,
		MaxHp:                             22,
		Defense:                           1,
		TorsoOnly:                         1,
		DefenseHead:                       2,
		DefenseTorso:                      3,
		DefenseLeftArm:                    4,
		DefenseRightArm:                   5,
		DefenseLeftLeg:                    6,
		DefenseRightLeg:                   7,
		DamageResistancePhysicalHead:      8,
		DamageResistancePhysicalTorso:     9,
		DamageResistancePhysicalLeftArm:   10,
		DamageResistancePhysicalRightArm:  11,
		DamageResistancePhysicalLeftLeg:   12,
		DamageResistancePhysicalRightLeg:  13,
		DamageResistancePhysical:          14,
		DamageResistanceEnergy:            15,
		DamageResistanceRadiation:         16,
		DamageResistancePoison:            17,
		DamageResistanceEnergyHead:        18,
		DamageResistanceEnergyTorso:       19,
		DamageResistanceEnergyLeftArm:     20,
		DamageResistanceEnergyRightArm:    21,
		DamageResistanceEnergyLeftLeg:     22,
		DamageResistanceEnergyRightLeg:    23,
		DamageResistanceRadiationHead:     24,
		DamageResistanceRadiationTorso:    25,
		DamageResistanceRadiationLeftArm:  26,
		DamageResistanceRadiationRightArm: 27,
		DamageResistanceRadiationLeftLeg:  28,
		DamageResistanceRadiationRightLeg: 29,
		DamageResistancePhysicalImmune:    1,
		DamageResistanceEnergyImmune:      0,
		DamageResistanceRadiationImmune:   1,
		DamageResistancePoisonImmune:      0,
		Active:                            1,
		Defeated:                          1,
	})

	assert.Equal(t, domain.Combatant{
		ID:                      "c1",
		Name:                    "Raider",
		Side:                    domain.SideNPC,
		TorsoOnly:               true,
		Level:                   4,
		XP:                      80,
		Initiative:              12,
		HP:                      17,
		MaxHP:                   22,
		Defense:                 1,
		DefenseHead:             2,
		DefenseTorso:            3,
		DefenseLeftArm:          4,
		DefenseRightArm:         5,
		DefenseLeftLeg:          6,
		DefenseRightLeg:         7,
		ResistPhysicalHead:      8,
		ResistPhysicalTorso:     9,
		ResistPhysicalLeftArm:   10,
		ResistPhysicalRightArm:  11,
		ResistPhysicalLeftLeg:   12,
		ResistPhysicalRightLeg:  13,
		ResistEnergyHead:        18,
		ResistEnergyTorso:       19,
		ResistEnergyLeftArm:     20,
		ResistEnergyRightArm:    21,
		ResistEnergyLeftLeg:     22,
		ResistEnergyRightLeg:    23,
		ResistRadiationHead:     24,
		ResistRadiationTorso:    25,
		ResistRadiationLeftArm:  26,
		ResistRadiationRightArm: 27,
		ResistRadiationLeftLeg:  28,
		ResistRadiationRightLeg: 29,
		ResistPhysical:          14,
		ResistEnergy:            15,
		ResistRadiation:         16,
		ResistPoison:            17,
		ImmunePhysical:          true,
		ImmuneEnergy:            false,
		ImmuneRadiation:         true,
		ImmunePoison:            false,
		Active:                  true,
		Defeated:                true,
	}, actual)
}

func TestCampaignPlayerFromRowMapsPartyCharacter(t *testing.T) {
	actual := campaignPlayerFromRow(dbgen.ListActivePartyCharactersByCampaignIDRow{
		ID:                                "pc1",
		PlayerName:                        "June",
		CharacterName:                     "Vault Dweller",
		Level:                             3,
		Initiative:                        9,
		Hp:                                18,
		MaxHp:                             21,
		Defense:                           2,
		TorsoOnly:                         1,
		DefenseHead:                       3,
		DamageResistancePhysicalHead:      4,
		DamageResistanceEnergyTorso:       5,
		DamageResistanceRadiationRightLeg: 6,
		DamageResistancePoison:            7,
		DamageResistanceEnergyImmune:      1,
	})

	assert.Equal(t, "June", actual.PlayerName)
	assert.Equal(t, domain.Combatant{
		ID:                      "pc1",
		Name:                    "Vault Dweller",
		Side:                    domain.SideParty,
		TorsoOnly:               true,
		Level:                   3,
		Initiative:              9,
		HP:                      18,
		MaxHP:                   21,
		Defense:                 2,
		DefenseHead:             3,
		ResistPhysicalHead:      4,
		ResistEnergyTorso:       5,
		ResistRadiationRightLeg: 6,
		ResistPoison:            7,
		ImmuneEnergy:            true,
	}, actual.Character)
}

func TestEncounterFromLatestRowMapsEncounterFields(t *testing.T) {
	combatants := []domain.Combatant{{ID: "c1", Name: "Alpha", Active: true}}

	actual := encounterFromLatestRow(dbgen.GetLatestEncounterByCampaignIDRow{
		ID:         "enc-1",
		CampaignID: []byte("camp-1"),
		Name:       "Vault Ambush",
		Round:      3,
		TurnIndex:  1,
		PartyAp:    4,
		GmThreat:   5,
	}, combatants)

	assert.Equal(t, &domain.Encounter{
		ID:         "enc-1",
		CampaignID: "camp-1",
		Name:       "Vault Ambush",
		Round:      3,
		TurnIndex:  1,
		Combatants: combatants,
		Resources: domain.Resources{
			PartyAP:  4,
			GMThreat: 5,
		},
	}, actual)
}

func TestEncounterFromByIDRowMapsEncounterFields(t *testing.T) {
	combatants := []domain.Combatant{{ID: "npc-1", Name: "Raider", Active: true}}

	actual := encounterFromByIDRow(dbgen.GetEncounterByIDByCampaignIDRow{
		ID:         "enc-2",
		CampaignID: "camp-2",
		Name:       "Road Fight",
		Round:      6,
		TurnIndex:  2,
		PartyAp:    1,
		GmThreat:   9,
	}, combatants)

	assert.Equal(t, &domain.Encounter{
		ID:         "enc-2",
		CampaignID: "camp-2",
		Name:       "Road Fight",
		Round:      6,
		TurnIndex:  2,
		Combatants: combatants,
		Resources: domain.Resources{
			PartyAP:  1,
			GMThreat: 9,
		},
	}, actual)
}

func TestEncounterSummaryFromRowMapsSummaryFields(t *testing.T) {
	updatedAt := time.Date(2026, 5, 22, 12, 13, 14, 123000000, time.UTC)

	actual := encounterSummaryFromRow(dbgen.ListEncounterSummariesByCampaignIDRow{
		ID:              "enc-1",
		CampaignID:      []byte("camp-1"),
		Name:            "Vault Ambush",
		Round:           7,
		Combatants:      4,
		DifficultyLabel: "challenging",
		DifficultyScore: 1.25,
		PartyCount:      2,
		PartyAvgLevel:   3.5,
		PartyXpBudget:   120,
		EnemyCount:      2,
		EnemyAvgLevel:   4.5,
		EnemyTotalXp:    150,
		UpdatedAt:       updatedAt,
	})

	assert.Equal(t, domain.EncounterSummary{
		ID:              "enc-1",
		CampaignID:      "camp-1",
		Name:            "Vault Ambush",
		Round:           7,
		Combatants:      4,
		Difficulty:      "challenging",
		DifficultyScore: 1.25,
		PartyCount:      2,
		PartyAvgLevel:   3.5,
		PartyXPBudget:   120,
		EnemyCount:      2,
		EnemyAvgLevel:   4.5,
		EnemyTotalXP:    150,
		UpdatedAt:       updatedAt,
	}, actual)
}

func TestInsertCombatantParamsMapsDomainCombatant(t *testing.T) {
	actual := insertCombatantParams("enc-1", 2, domain.Combatant{
		ID:         "c1",
		Name:       "Raider",
		Side:       domain.SideNPC,
		TorsoOnly:  true,
		Level:      4,
		XP:         80,
		Initiative: 12,
		HP:         17,
		MaxHP:      22,
		Defense:    1,
		Active:     true,
		Defeated:   true,
	})

	assert.Equal(t, dbgen.InsertCombatantParams{
		ID:          "c1",
		EncounterID: "enc-1",
		Name:        "Raider",
		Side:        string(domain.SideNPC),
		TorsoOnly:   1,
		Level:       4,
		Xp:          80,
		Initiative:  12,
		Hp:          17,
		MaxHp:       22,
		Defense:     1,
		Active:      1,
		Defeated:    1,
		Position:    2,
	}, actual)
}

func TestPlayerCharacterParamsMapAndTrimDomainCombatant(t *testing.T) {
	combatant := domain.Combatant{
		Name:       "  Vault Dweller  ",
		Level:      3,
		Initiative: 9,
		HP:         18,
		MaxHP:      21,
		Defense:    2,
		TorsoOnly:  true,
	}

	insertParams := insertPlayerCharacterParams("pc-1", "player-1", "camp-1", combatant)
	assert.Equal(t, dbgen.InsertPlayerCharacterParams{
		ID:         "pc-1",
		PlayerID:   "player-1",
		CampaignID: "camp-1",
		Name:       "Vault Dweller",
		Level:      3,
		Initiative: 9,
		Hp:         18,
		MaxHp:      21,
		Defense:    2,
		TorsoOnly:  1,
		Active:     1,
	}, insertParams)

	updateParams := updateActivePlayerCharacterParams("pc-1", "camp-1", combatant)
	assert.Equal(t, dbgen.UpdateActivePlayerCharacterByIDParams{
		CharacterID: "pc-1",
		CampaignID:  "camp-1",
		Name:        "Vault Dweller",
		Level:       3,
		Initiative:  9,
		Hp:          18,
		MaxHp:       21,
		Defense:     2,
		TorsoOnly:   1,
	}, updateParams)
}

func TestCampaignFromListRowFormatsUpdatedAt(t *testing.T) {
	startDate := time.Date(2287, 10, 23, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 22, 11, 12, 13, 456000000, time.UTC)

	actual := campaignFromListRow(dbgen.ListCampaignsRow{
		ID:        "camp-1",
		Name:      "Capital Wasteland",
		StartDate: "2287-10-23",
		UpdatedAt: updatedAt,
	})

	assert.Equal(t, domain.Campaign{
		ID:        "camp-1",
		Name:      "Capital Wasteland",
		StartDate: startDate,
		UpdatedAt: updatedAt,
	}, actual)
}

func TestEncounterLogFromRowFormatsCreatedAt(t *testing.T) {
	createdAt := time.Date(2026, 5, 22, 14, 15, 16, 789000000, time.UTC)

	actual := encounterLogFromRow(dbgen.ListEncounterLogsByEncounterIDRow{
		Round:     3,
		Message:   "Turn advanced",
		CreatedAt: createdAt,
	})

	assert.Equal(t, domain.EncounterLog{
		Round:     3,
		Message:   "Turn advanced",
		CreatedAt: createdAt,
	}, actual)
}
