package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestEncounterOrderViewRebuildUsesScannableTableRows(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	selectedIndex := 0
	expandedCombatantID := "pc-1"
	selectedLabel := widget.NewLabel("")
	view := newEncounterOrderView(
		&enc,
		&selectedIndex,
		&expandedCombatantID,
		selectedLabel,
		func(int) {},
		func(int) {},
	)

	view.Rebuild()

	orderBox := view.OrderBox().(*fyne.Container)
	require.Len(t, orderBox.Objects, len(enc.Combatants)+1)

	headerLabels := collectLabels(orderBox.Objects[0])
	assert.Contains(t, headerLabels, "Name")
	assert.Contains(t, headerLabels, "Side")
	assert.Contains(t, headerLabels, "Init")
	assert.Contains(t, headerLabels, "HP")
	assert.Contains(t, headerLabels, "Status")

	expandedRow := orderBox.Objects[1].(*fyne.Container)
	require.Len(t, expandedRow.Objects, 2)
	details := expandedRow.Objects[1].(*widget.Label)
	assert.Contains(t, details.Text, "Participant Details")
	assert.Contains(t, details.Text, "Field")
	assert.Contains(t, details.Text, "Value")
	assert.Contains(t, details.Text, "Body Damage Resistance")

	buttons := collectButtons(expandedRow)
	assert.Contains(t, buttonTexts(buttons), "Alpha")
	assert.Contains(t, buttonTexts(buttons), "DMG")
	assert.Contains(t, buttonTexts(buttons), "HEAL")

	labels := collectLabels(expandedRow)
	assert.Contains(t, labels, ">>")
	assert.Contains(t, labels, "PARTY")
	assert.Contains(t, labels, "12")
	assert.Contains(t, labels, "8/8")
	assert.Contains(t, labels, "Active")
	assert.Len(t, collectProgressBars(expandedRow), 1)

	collapsedRow := orderBox.Objects[2].(*fyne.Container)
	require.Len(t, collapsedRow.Objects, 1)
}

func TestEncounterOrderViewRebuildDisablesDefeatedRows(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	enc.Combatants[1].HP = 0
	enc.Combatants[1].Defeated = true
	selectedIndex := 0
	expandedCombatantID := ""
	selectedLabel := widget.NewLabel("")
	view := newEncounterOrderView(
		&enc,
		&selectedIndex,
		&expandedCombatantID,
		selectedLabel,
		func(int) {},
		func(int) {},
	)

	view.Rebuild()

	orderBox := view.OrderBox().(*fyne.Container)
	require.Len(t, orderBox.Objects, len(enc.Combatants)+1)

	row := orderBox.Objects[2].(*fyne.Container)
	buttons := collectButtons(row)
	nameBtn := requireButtonWithText(t, buttons, "Raider")
	assert.True(t, nameBtn.Disabled())

	damageBtn := requireButtonWithText(t, buttons, "DMG")
	healBtn := requireButtonWithText(t, buttons, "HEAL")
	assert.True(t, damageBtn.Disabled())
	assert.False(t, healBtn.Disabled())

	labels := collectLabels(row)
	assert.Contains(t, labels, "xx")
	assert.Contains(t, labels, "Defeated")
	assert.Contains(t, labels, "0/6")
}

func TestEncounterOrderViewRebuildMarksSelectedRowSeparatelyFromActiveTurn(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	selectedIndex := 1
	expandedCombatantID := ""
	selectedLabel := widget.NewLabel("")
	view := newEncounterOrderView(
		&enc,
		&selectedIndex,
		&expandedCombatantID,
		selectedLabel,
		func(int) {},
		func(int) {},
	)

	view.Rebuild()

	orderBox := view.OrderBox().(*fyne.Container)
	activeLabels := collectLabels(orderBox.Objects[1])
	selectedLabels := collectLabels(orderBox.Objects[2])
	assert.Contains(t, activeLabels, ">>")
	assert.Contains(t, selectedLabels, ">")
	assert.NotContains(t, selectedLabels, ">>")
}

func TestEncounterOrderViewRebuildShowsLongRussianCriticalImmuneState(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	enc.Combatants[0].Name = "Сверхдлинное Имя Персонажа Из Пустоши"
	enc.Combatants[0].HP = 2
	enc.Combatants[0].MaxHP = 8
	enc.Combatants[0].ImmunePoison = true
	selectedIndex := 0
	expandedCombatantID := ""
	selectedLabel := widget.NewLabel("")
	view := newEncounterOrderView(
		&enc,
		&selectedIndex,
		&expandedCombatantID,
		selectedLabel,
		func(int) {},
		func(int) {},
	)

	view.Rebuild()

	orderBox := view.OrderBox().(*fyne.Container)
	row := orderBox.Objects[1].(*fyne.Container)
	buttons := collectButtons(row)
	nameBtn := requireButtonWithText(t, buttons, "Сверхдлинное Имя Персонажа Из Пустоши")
	assert.False(t, nameBtn.Disabled())
	labels := collectLabels(row)
	assert.Contains(t, labels, "2/8")
	assert.Contains(t, labels, "IMM")
	assert.Contains(t, labels, "Active, Critical")

	hpLabels := collectLabelsWithText(row, "2/8")
	require.NotEmpty(t, hpLabels)
	assert.Equal(t, widget.WarningImportance, hpLabels[0].Importance)
	poisonLabels := collectLabelsWithText(row, formatCombatantGlobalResistance(enc.Combatants[0], domain.DamagePoison))
	require.NotEmpty(t, poisonLabels)
	assert.Equal(t, widget.SuccessImportance, poisonLabels[0].Importance)
}

func collectButtons(obj fyne.CanvasObject) []*widget.Button {
	var buttons []*widget.Button
	if btn, ok := obj.(*widget.Button); ok {
		return append(buttons, btn)
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			buttons = append(buttons, collectButtons(child)...)
		}
	}
	return buttons
}

func collectLabels(obj fyne.CanvasObject) []string {
	var labels []string
	if label, ok := obj.(*widget.Label); ok {
		return append(labels, label.Text)
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			labels = append(labels, collectLabels(child)...)
		}
	}
	return labels
}

func collectLabelsWithText(obj fyne.CanvasObject, text string) []*widget.Label {
	var labels []*widget.Label
	if label, ok := obj.(*widget.Label); ok && label.Text == text {
		return append(labels, label)
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			labels = append(labels, collectLabelsWithText(child, text)...)
		}
	}
	return labels
}

func collectProgressBars(obj fyne.CanvasObject) []*widget.ProgressBar {
	var bars []*widget.ProgressBar
	if bar, ok := obj.(*widget.ProgressBar); ok {
		return append(bars, bar)
	}
	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			bars = append(bars, collectProgressBars(child)...)
		}
	}
	return bars
}

func buttonTexts(buttons []*widget.Button) []string {
	texts := make([]string, 0, len(buttons))
	for _, button := range buttons {
		texts = append(texts, button.Text)
	}
	return texts
}

func requireButtonWithText(t *testing.T, buttons []*widget.Button, text string) *widget.Button {
	t.Helper()
	for _, button := range buttons {
		if button.Text == text {
			return button
		}
	}
	require.Failf(t, "button not found", "missing button %q in %v", text, buttonTexts(buttons))
	return nil
}
