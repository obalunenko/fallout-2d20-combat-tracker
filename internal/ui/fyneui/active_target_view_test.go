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

func TestActiveTargetViewShowsEmptyStateAndDisablesActions(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	damageBtn := widget.NewButton("DMG", func() {})
	healBtn := widget.NewButton("HEAL", func() {})

	view := newActiveTargetView(widget.NewLabel(""), damageBtn, healBtn)

	assert.Equal(t, "No active target", view.nameLabel.Text)
	assert.Equal(t, "No combatants", view.detailsLabel.Text)
	assert.True(t, damageBtn.Disabled())
	assert.True(t, healBtn.Disabled())
	assert.Equal(t, float64(0), view.hpBar.Value)
	assert.Equal(t, float64(1), view.hpBar.Max)
}

func TestActiveTargetViewShowsSummaryAndKeepsDetailsCollapsible(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	damageBtn := widget.NewButton("DMG", func() {})
	healBtn := widget.NewButton("HEAL", func() {})

	view := newActiveTargetView(widget.NewLabel(""), damageBtn, healBtn)
	view.SetTarget(enc, 0)

	require.NotNil(t, view.accordion)
	require.Len(t, view.accordion.Items, 1)
	assert.Equal(t, "TARGET DETAILS", view.accordion.Items[0].Title)
	assert.False(t, view.accordion.Items[0].Open)
	assert.Equal(t, "Alpha", view.nameLabel.Text)
	assert.Equal(t, "PARTY", view.sideLabel.Text)
	assert.Equal(t, "0", view.levelLabel.Text)
	assert.Equal(t, "0", view.xpLabel.Text)
	assert.Equal(t, "Active", view.statusLabel.Text)
	assert.Equal(t, "8/8", view.hpLabel.Text)
	assert.Equal(t, float64(8), view.hpBar.Value)
	assert.Equal(t, float64(8), view.hpBar.Max)
	assert.Equal(t, "12", view.initLabel.Text)
	assert.Equal(t, "0", view.defenseLabel.Text)
	assert.Equal(t, "0", view.poisonLabel.Text)
	assert.Contains(t, view.detailsLabel.Text, "Participant Details")
	assert.Contains(t, view.detailsLabel.Text, "Body Damage Resistance")
	assert.False(t, damageBtn.Disabled())
	assert.False(t, healBtn.Disabled())
}

func TestActiveTargetViewDisablesDamageForDefeatedTarget(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	enc.Combatants[1].HP = 0
	enc.Combatants[1].Defeated = true
	damageBtn := widget.NewButton("DMG", func() {})
	healBtn := widget.NewButton("HEAL", func() {})

	view := newActiveTargetView(widget.NewLabel(""), damageBtn, healBtn)
	view.SetTarget(enc, 1)

	assert.Equal(t, "Raider", view.nameLabel.Text)
	assert.Equal(t, "Defeated", view.statusLabel.Text)
	assert.Equal(t, "0/6", view.hpLabel.Text)
	assert.Equal(t, float64(0), view.hpBar.Value)
	assert.True(t, damageBtn.Disabled())
	assert.False(t, healBtn.Disabled())
}

func TestActiveTargetViewShowsLongRussianCriticalImmuneTarget(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	enc := testEncounter()
	enc.Combatants[0].Name = "Сверхдлинное Имя Персонажа Из Пустоши"
	enc.Combatants[0].HP = 2
	enc.Combatants[0].MaxHP = 8
	enc.Combatants[0].ImmunePoison = true
	damageBtn := widget.NewButton("DMG", func() {})
	healBtn := widget.NewButton("HEAL", func() {})

	view := newActiveTargetView(widget.NewLabel(""), damageBtn, healBtn)
	view.SetTarget(enc, 0)

	assert.Equal(t, "Сверхдлинное Имя Персонажа Из Пустоши", view.nameLabel.Text)
	assert.Equal(t, fyne.TextWrapOff, view.nameLabel.Wrapping)
	assert.Equal(t, fyne.TextTruncateClip, view.nameLabel.Truncation)
	assert.Equal(t, "Active, Critical", view.statusLabel.Text)
	assert.Equal(t, widget.HighImportance, view.statusLabel.Importance)
	assert.Equal(t, "2/8", view.hpLabel.Text)
	assert.Equal(t, widget.WarningImportance, view.hpLabel.Importance)
	assert.Equal(t, float64(2), view.hpBar.Value)
	assert.Equal(t, "IMM", view.poisonLabel.Text)
	assert.Equal(t, widget.SuccessImportance, view.poisonLabel.Importance)
	assert.Contains(t, view.detailsLabel.Text, "Сверхдлинное Имя Персонажа")
	assert.Contains(t, view.detailsLabel.Text, formatCombatantGlobalResistance(enc.Combatants[0], domain.DamagePoison))
	assert.False(t, damageBtn.Disabled())
	assert.False(t, healBtn.Disabled())
}
