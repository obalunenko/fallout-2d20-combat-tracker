package fyneui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestCollectCombatantsFromRowsExpandsNPCCount(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		Name:                    "Raider",
		Level:                   3,
		XP:                      40,
		Initiative:              9,
		HP:                      6,
		MaxHP:                   8,
		Defense:                 1,
		ResistPhysicalHead:      4,
		ResistEnergyTorso:       5,
		ResistRadiationRightLeg: 6,
		ResistPoison:            7,
	}, domain.SideNPC, 2)

	combatants, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.NoError(t, err)
	require.Len(t, combatants, 2)
	for _, combatant := range combatants {
		assert.Equal(t, "Raider", combatant.Name)
		assert.Equal(t, domain.SideNPC, combatant.Side)
		assert.Equal(t, 3, combatant.Level)
		assert.Equal(t, 40, combatant.XP)
		assert.Equal(t, 9, combatant.Initiative)
		assert.Equal(t, 6, combatant.HP)
		assert.Equal(t, 8, combatant.MaxHP)
		assert.Equal(t, 1, combatant.Defense)
		assert.Equal(t, 4, combatant.ResistPhysicalHead)
		assert.Equal(t, 5, combatant.ResistEnergyTorso)
		assert.Equal(t, 6, combatant.ResistRadiationRightLeg)
		assert.Equal(t, 7, combatant.ResistPoison)
	}
}

func TestCollectCombatantsFromRowsForcesPartyCountAndXP(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		Name:       "Vault Dweller",
		Level:      2,
		XP:         99,
		Initiative: 10,
		HP:         8,
		MaxHP:      8,
		Defense:    1,
	}, domain.SideParty, 3)

	combatants, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.NoError(t, err)
	require.Len(t, combatants, 1)
	assert.Equal(t, domain.SideParty, combatants[0].Side)
	assert.Equal(t, 0, combatants[0].XP)
}

func TestCollectCombatantsFromRowsFlattensTorsoOnlyStats(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		Name:                   "Turret",
		TorsoOnly:              true,
		Level:                  2,
		XP:                     25,
		Initiative:             5,
		HP:                     6,
		MaxHP:                  6,
		Defense:                3,
		ResistPhysicalHead:     4,
		ResistPhysicalTorso:    5,
		ResistEnergyLeftArm:    6,
		ResistEnergyTorso:      7,
		ResistRadiationLeftLeg: 8,
		ResistRadiationTorso:   9,
	}, domain.SideNPC, 1)

	combatants, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.NoError(t, err)
	require.Len(t, combatants, 1)
	combatant := combatants[0]
	assert.True(t, combatant.TorsoOnly)
	assert.Equal(t, 0, combatant.ResistPhysicalHead)
	assert.Equal(t, 5, combatant.ResistPhysicalTorso)
	assert.Equal(t, 0, combatant.ResistEnergyLeftArm)
	assert.Equal(t, 7, combatant.ResistEnergyTorso)
	assert.Equal(t, 0, combatant.ResistRadiationLeftLeg)
	assert.Equal(t, 9, combatant.ResistRadiationTorso)
}

func TestCollectCombatantsFromRowsValidatesRequiredValues(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	row.name.SetText("Raider")

	_, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "initiative is required")
}

func TestCollectCampaignPlayersFromRowsMapsPlayerCharacter(t *testing.T) {
	test.NewTempApp(t)
	row := newCampaignPlayerInputRow(func(*campaignPlayerInputRow) {})
	row.playerName.SetText(" June ")
	row.characterName.SetText(" Vault Dweller ")
	row.level.SetText("4")
	row.initiative.SetText("11")
	row.hp.SetText("9")
	row.hpMax.SetText("12")
	row.defense.SetText("2")
	row.drPhysHead.SetText("4")
	row.drEnergyTorso.SetText("5")
	row.drRadRL.SetText("6")
	row.drPoison.SetText("imm")

	players, err := collectCampaignPlayersFromRows([]*campaignPlayerInputRow{row})

	require.NoError(t, err)
	require.Len(t, players, 1)
	assert.Equal(t, "June", players[0].PlayerName)
	character := players[0].Character
	assert.Equal(t, "Vault Dweller", character.Name)
	assert.Equal(t, domain.SideParty, character.Side)
	assert.Equal(t, 4, character.Level)
	assert.Equal(t, 11, character.Initiative)
	assert.Equal(t, 9, character.HP)
	assert.Equal(t, 12, character.MaxHP)
	assert.Equal(t, 2, character.Defense)
	assert.Equal(t, 4, character.ResistPhysicalHead)
	assert.Equal(t, 5, character.ResistEnergyTorso)
	assert.Equal(t, 6, character.ResistRadiationRightLeg)
	assert.True(t, character.ImmunePoison)
	assert.Equal(t, 0, character.ResistPoison)
}

func TestCollectCampaignPlayersFromRowsValidatesHPBounds(t *testing.T) {
	test.NewTempApp(t)
	row := newCampaignPlayerInputRow(func(*campaignPlayerInputRow) {})
	row.playerName.SetText("June")
	row.characterName.SetText("Vault Dweller")
	row.hp.SetText("12")
	row.hpMax.SetText("8")

	_, err := collectCampaignPlayersFromRows([]*campaignPlayerInputRow{row})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "current HP cannot exceed max HP")
}
