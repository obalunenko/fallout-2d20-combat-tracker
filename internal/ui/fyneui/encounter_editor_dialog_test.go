package fyneui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestEncounterEditorDifficultyPreviewRefreshesFromDraftChanges(t *testing.T) {
	test.NewTempApp(t)
	var rows []*combatantInputRow
	preview := widget.NewLabel("")
	refresh := func() {
		preview.SetText(formatDifficultyPreview(collectDraftDifficultyFromRows(rows)))
	}
	addRow := func(defaultSide string) *combatantInputRow {
		row := newCombatantInputRow(defaultSide, func(*combatantInputRow) {}, refresh)
		rows = append(rows, row)
		refresh()
		return row
	}
	removeRow := func(target *combatantInputRow) {
		filtered := make([]*combatantInputRow, 0, len(rows)-1)
		for _, row := range rows {
			if row != target {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
		refresh()
	}

	party := addRow("party")
	fillCombatantInputRow(party, domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 5}, domain.SideParty, 1)
	assert.Contains(t, preview.Text, "Difficulty: Trivial")

	monster := addRow("npc")
	fillCombatantInputRow(monster, domain.Combatant{Name: "Raider", XP: 20}, domain.SideNPC, 1)
	assert.Contains(t, preview.Text, "monster XP: 20")

	monster.number.SetText("3")
	assert.Contains(t, preview.Text, "monster XP: 60")

	monster.xp.SetText("40")
	assert.Contains(t, preview.Text, "monster XP: 120")

	party.level.SetText("4")
	assert.Contains(t, preview.Text, "avg PC lvl 4")

	secondMonster := addRow("npc")
	fillCombatantInputRow(secondMonster, domain.Combatant{Name: "Mutant", XP: 30}, domain.SideNPC, 1)
	assert.Contains(t, preview.Text, "monster XP: 150")

	removeRow(secondMonster)
	assert.Contains(t, preview.Text, "monster XP: 120")
}

func TestEncounterEditorDifficultyPreviewDoesNotInvokeSubmitCallback(t *testing.T) {
	test.NewTempApp(t)
	var rows []*combatantInputRow
	preview := widget.NewLabel("")
	submitCalls := 0
	onSubmit := func(string, []domain.Combatant) error {
		submitCalls++
		return nil
	}
	save := func() error {
		combatants, err := collectCombatantsFromRows(rows)
		if err != nil {
			return err
		}
		return onSubmit("Draft", combatants)
	}
	refresh := func() {
		preview.SetText(formatDifficultyPreview(collectDraftDifficultyFromRows(rows)))
	}

	party := newCombatantInputRow("party", func(*combatantInputRow) {}, refresh)
	monster := newCombatantInputRow("npc", func(*combatantInputRow) {}, refresh)
	rows = []*combatantInputRow{party, monster}
	fillCombatantInputRow(party, domain.Combatant{ID: "char-1", Name: "Vault Dweller", Level: 2, Initiative: 10, HP: 8, MaxHP: 8}, domain.SideParty, 1)
	fillCombatantInputRow(monster, domain.Combatant{Name: "Raider", Level: 1, XP: 20, Initiative: 8, HP: 6, MaxHP: 6}, domain.SideNPC, 1)
	refresh()
	require.Contains(t, preview.Text, "Difficulty:")

	monster.number.SetText("4")
	monster.xp.SetText("30")
	refresh()

	assert.Zero(t, submitCalls)
	require.NoError(t, save())
	assert.Equal(t, 1, submitCalls)
}
