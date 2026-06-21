package fyneui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestCollectDraftDifficultyFromRowsUsesPartyLevelsAndMonsterQuantity(t *testing.T) {
	test.NewTempApp(t)
	rows := []*combatantInputRow{
		newCombatantInputRow("party", func(*combatantInputRow) {}, nil),
		newCombatantInputRow("party", func(*combatantInputRow) {}, nil),
		newCombatantInputRow("npc", func(*combatantInputRow) {}, nil),
	}
	fillCombatantInputRow(rows[0], domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 2}, domain.SideParty, 1)
	fillCombatantInputRow(rows[1], domain.Combatant{ID: "char-2", Name: "Companion", Level: 3}, domain.SideParty, 1)
	fillCombatantInputRow(rows[2], domain.Combatant{Name: "Raider", XP: 40}, domain.SideNPC, 3)

	metrics := collectDraftDifficultyFromRows(rows)

	assert.Equal(t, domain.EncounterDifficultyHard, metrics.Label)
	assert.Equal(t, 2, metrics.PartyCount)
	assert.Equal(t, 3, metrics.AveragePCLevel)
	assert.Equal(t, 120, metrics.TotalMonsterXP)
	assert.Equal(t, 60.0, metrics.XPBaseline)
	assert.Equal(t, 5, metrics.EncounterLevel)
	assert.Equal(t, 2, metrics.Difference)
}

func TestCollectDraftDifficultyFromRowsCalculatesWithPlayersAndNoMonsters(t *testing.T) {
	test.NewTempApp(t)
	rows := []*combatantInputRow{
		newCombatantInputRow("party", func(*combatantInputRow) {}, nil),
		newCombatantInputRow("party", func(*combatantInputRow) {}, nil),
	}
	fillCombatantInputRow(rows[0], domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 5}, domain.SideParty, 1)
	fillCombatantInputRow(rows[1], domain.Combatant{ID: "char-2", Name: "Companion", Level: 5}, domain.SideParty, 1)

	metrics := collectDraftDifficultyFromRows(rows)

	assert.Equal(t, domain.EncounterDifficultyTrivial, metrics.Label)
	assert.Equal(t, 0, metrics.TotalMonsterXP)
	assert.Equal(t, 1, metrics.EncounterLevel)
	assert.Equal(t, -4, metrics.Difference)
}

func TestCollectDraftDifficultyFromRowsReturnsUnavailableForInvalidDraftInput(t *testing.T) {
	test.NewTempApp(t)
	party := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(party, domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 2}, domain.SideParty, 1)
	monster := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(monster, domain.Combatant{Name: "Raider", XP: 40}, domain.SideNPC, 1)
	monster.number.SetText("nope")

	metrics := collectDraftDifficultyFromRows([]*combatantInputRow{party, monster})

	assert.Equal(t, domain.EncounterDifficultyUnknown, metrics.Label)
	assert.Contains(t, metrics.UnavailableReason, "invalid quantity")
}

func TestCollectDraftDifficultyUnavailableDoesNotReplaceSaveValidation(t *testing.T) {
	test.NewTempApp(t)
	party := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(party, domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 2, Initiative: 10, HP: 8, MaxHP: 8}, domain.SideParty, 1)
	monster := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(monster, domain.Combatant{Name: "Raider", Level: 1, XP: 40, Initiative: 8, HP: 6, MaxHP: 6}, domain.SideNPC, 1)
	monster.xp.SetText("bad")

	metrics := collectDraftDifficultyFromRows([]*combatantInputRow{party, monster})
	_, saveErr := collectCombatantsFromRows([]*combatantInputRow{party, monster})

	assert.Equal(t, domain.EncounterDifficultyUnknown, metrics.Label)
	assert.Contains(t, metrics.UnavailableReason, "invalid XP")
	require.Error(t, saveErr)
	assert.Contains(t, saveErr.Error(), "invalid XP")
}

func TestCollectDraftDifficultyFromRowsRecalculatesTabletopScaleQuickly(t *testing.T) {
	test.NewTempApp(t)
	rows := make([]*combatantInputRow, 0, 12)
	for i := 0; i < 4; i++ {
		row := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
		fillCombatantInputRow(row, domain.Combatant{ID: "char", Name: "Party", Level: 3}, domain.SideParty, 1)
		rows = append(rows, row)
	}
	for i := 0; i < 8; i++ {
		row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
		fillCombatantInputRow(row, domain.Combatant{Name: "Raider", XP: 20}, domain.SideNPC, 1)
		rows = append(rows, row)
	}

	start := time.Now()
	metrics := collectDraftDifficultyFromRows(rows)
	elapsed := time.Since(start)

	assert.Equal(t, domain.EncounterDifficultyAverage, metrics.Label)
	assert.Less(t, elapsed, 100*time.Millisecond)
}

func TestCollectCombatantsFromRowsExpandsNPCCount(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	template := domain.Combatant{
		Name:       "Raider",
		Level:      3,
		XP:         40,
		Initiative: 9,
		HP:         6,
		MaxHP:      8,
		Defense:    1,
	}
	setTestLocationResistance(t, &template, domain.DamagePhysical, domain.BodyHead, 4)
	setTestLocationResistance(t, &template, domain.DamageEnergy, domain.BodyTorso, 5)
	setTestLocationResistance(t, &template, domain.DamageRadiation, domain.BodyRightLeg, 6)
	setTestGlobalResistance(t, &template, domain.DamagePoison, 7, false)
	fillCombatantInputRow(row, template, domain.SideNPC, 2)

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
		assertLocationResistance(t, combatant, domain.DamagePhysical, domain.BodyHead, 4)
		assertLocationResistance(t, combatant, domain.DamageEnergy, domain.BodyTorso, 5)
		assertLocationResistance(t, combatant, domain.DamageRadiation, domain.BodyRightLeg, 6)
		assertGlobalResistance(t, combatant, domain.DamagePoison, 7, false)
	}
}

func TestCollectCombatantsFromRowsForcesPartyCountAndXP(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		ID:         "char-1",
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
	assert.Empty(t, combatants[0].ID)
	assert.Equal(t, "char-1", combatants[0].PlayerCharacterID)
	assert.Equal(t, domain.SideParty, combatants[0].Side)
	assert.Equal(t, 0, combatants[0].XP)
}

func TestCollectCombatantsFromRowsRejectsManualPartyMember(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		Name:       "Vault Dweller",
		Level:      2,
		Initiative: 10,
		HP:         8,
		MaxHP:      8,
	}, domain.SideParty, 1)

	_, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "party members must be loaded from campaign")
}

func TestFillCombatantInputRowLocksLoadedPartyStats(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("party", func(*combatantInputRow) {}, nil)
	fillCombatantInputRow(row, domain.Combatant{
		ID:         "char-1",
		Name:       "Vault Dweller",
		Level:      2,
		Initiative: 10,
		HP:         8,
		MaxHP:      8,
		Defense:    1,
	}, domain.SideParty, 1)

	assert.True(t, row.linkedParty)
	assert.True(t, row.name.Disabled())
	assert.True(t, row.side.Disabled())
	assert.True(t, row.level.Disabled())
	assert.True(t, row.hp.Disabled())
	assert.True(t, row.resistance.globalEntry(domain.DamagePoison).Disabled())
	assert.True(t, row.resistance.globalImmune(domain.DamagePoison).Disabled())
}

func TestCollectCombatantsFromRowsFlattensTorsoOnlyStats(t *testing.T) {
	test.NewTempApp(t)
	row := newCombatantInputRow("npc", func(*combatantInputRow) {}, nil)
	template := domain.Combatant{
		Name:       "Turret",
		TorsoOnly:  true,
		Level:      2,
		XP:         25,
		Initiative: 5,
		HP:         6,
		MaxHP:      6,
		Defense:    3,
	}
	setTestLocationResistance(t, &template, domain.DamagePhysical, domain.BodyHead, 4)
	setTestLocationResistance(t, &template, domain.DamagePhysical, domain.BodyTorso, 5)
	setTestLocationResistance(t, &template, domain.DamageEnergy, domain.BodyLeftArm, 6)
	setTestLocationResistance(t, &template, domain.DamageEnergy, domain.BodyTorso, 7)
	setTestLocationResistance(t, &template, domain.DamageRadiation, domain.BodyLeftLeg, 8)
	setTestLocationResistance(t, &template, domain.DamageRadiation, domain.BodyTorso, 9)
	fillCombatantInputRow(row, template, domain.SideNPC, 1)

	combatants, err := collectCombatantsFromRows([]*combatantInputRow{row})

	require.NoError(t, err)
	require.Len(t, combatants, 1)
	combatant := combatants[0]
	assert.True(t, combatant.TorsoOnly)
	assertLocationResistance(t, combatant, domain.DamagePhysical, domain.BodyHead, 0)
	assertLocationResistance(t, combatant, domain.DamagePhysical, domain.BodyTorso, 5)
	assertLocationResistance(t, combatant, domain.DamageEnergy, domain.BodyLeftArm, 0)
	assertLocationResistance(t, combatant, domain.DamageEnergy, domain.BodyTorso, 7)
	assertLocationResistance(t, combatant, domain.DamageRadiation, domain.BodyLeftLeg, 0)
	assertLocationResistance(t, combatant, domain.DamageRadiation, domain.BodyTorso, 9)
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
	row.resistance.locationEntry(domain.DamagePhysical, domain.BodyHead).SetText("4")
	row.resistance.locationEntry(domain.DamageEnergy, domain.BodyTorso).SetText("5")
	row.resistance.locationEntry(domain.DamageRadiation, domain.BodyRightLeg).SetText("6")
	row.resistance.globalEntry(domain.DamagePoison).SetText("imm")
	row.active.SetChecked(false)

	players, err := collectCampaignPlayersFromRows([]*campaignPlayerInputRow{row})

	require.NoError(t, err)
	require.Len(t, players, 1)
	assert.Equal(t, "June", players[0].PlayerName)
	assert.True(t, players[0].Inactive)
	character := players[0].Character
	assert.Equal(t, "Vault Dweller", character.Name)
	assert.Equal(t, domain.SideParty, character.Side)
	assert.Equal(t, 4, character.Level)
	assert.Equal(t, 11, character.Initiative)
	assert.Equal(t, 9, character.HP)
	assert.Equal(t, 12, character.MaxHP)
	assert.Equal(t, 2, character.Defense)
	assertLocationResistance(t, character, domain.DamagePhysical, domain.BodyHead, 4)
	assertLocationResistance(t, character, domain.DamageEnergy, domain.BodyTorso, 5)
	assertLocationResistance(t, character, domain.DamageRadiation, domain.BodyRightLeg, 6)
	assertGlobalResistance(t, character, domain.DamagePoison, 0, true)
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

func setTestGlobalResistance(t *testing.T, combatant *domain.Combatant, damageType domain.DamageType, value int, immune bool) {
	t.Helper()
	require.NoError(t, combatant.SetGlobalResistance(damageType, value, immune))
}

func setTestLocationResistance(t *testing.T, combatant *domain.Combatant, damageType domain.DamageType, location domain.BodyLocation, value int) {
	t.Helper()
	require.NoError(t, combatant.SetLocationResistance(damageType, location, value))
}

func assertGlobalResistance(t *testing.T, combatant domain.Combatant, damageType domain.DamageType, wantValue int, wantImmune bool) {
	t.Helper()
	value, immune, err := combatant.GlobalResistance(damageType)
	require.NoError(t, err)
	assert.Equal(t, wantValue, value)
	assert.Equal(t, wantImmune, immune)
}

func assertLocationResistance(t *testing.T, combatant domain.Combatant, damageType domain.DamageType, location domain.BodyLocation, want int) {
	t.Helper()
	value, err := combatant.LocationResistance(damageType, location)
	require.NoError(t, err)
	assert.Equal(t, want, value)
}
