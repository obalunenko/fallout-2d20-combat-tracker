package fyneui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

func newCombatantInputRow(defaultSide string, onRemove func(*combatantInputRow), onChanged func()) *combatantInputRow {
	notifyChange := func() {
		if onChanged != nil {
			onChanged()
		}
	}

	name := widget.NewEntry()
	name.SetPlaceHolder("Name")
	name.TextStyle = fyne.TextStyle{Monospace: true}
	name.OnChanged = func(string) { notifyChange() }

	sideOptions := []string{"npc"}
	if defaultSide == "party" {
		sideOptions = []string{"party"}
	}
	side := widget.NewSelect(sideOptions, nil)
	side.SetSelected(defaultSide)
	torsoOnly := widget.NewCheck("torso-only", func(bool) {
		notifyChange()
	})
	number := widget.NewEntry()
	number.SetPlaceHolder("Count")
	number.TextStyle = fyne.TextStyle{Monospace: true}
	number.SetText("1")
	number.OnChanged = func(string) { notifyChange() }
	level := widget.NewEntry()
	level.SetPlaceHolder("Level")
	level.TextStyle = fyne.TextStyle{Monospace: true}
	level.SetText("1")
	level.OnChanged = func(string) { notifyChange() }
	xp := widget.NewEntry()
	xp.SetPlaceHolder("XP")
	xp.TextStyle = fyne.TextStyle{Monospace: true}
	xp.SetText("0")
	xp.OnChanged = func(string) { notifyChange() }

	initiative := widget.NewEntry()
	initiative.SetPlaceHolder("Init")
	initiative.TextStyle = fyne.TextStyle{Monospace: true}
	initiative.OnChanged = func(string) { notifyChange() }
	hp := widget.NewEntry()
	hp.SetPlaceHolder("HP")
	hp.TextStyle = fyne.TextStyle{Monospace: true}
	hp.SetText("1")
	hp.OnChanged = func(string) { notifyChange() }
	hpMax := widget.NewEntry()
	hpMax.SetPlaceHolder("Max HP")
	hpMax.TextStyle = fyne.TextStyle{Monospace: true}
	hpMax.SetText("1")
	hpMax.OnChanged = func(string) { notifyChange() }
	defense := widget.NewEntry()
	defense.SetPlaceHolder("Defense")
	defense.TextStyle = fyne.TextStyle{Monospace: true}
	defense.SetText("0")
	defense.OnChanged = func(string) { notifyChange() }
	resistance := newUIResistanceInputs(notifyChange)

	row := &combatantInputRow{
		name:       name,
		side:       side,
		torsoOnly:  torsoOnly,
		number:     number,
		level:      level,
		xp:         xp,
		initiative: initiative,
		hp:         hp,
		hpMax:      hpMax,
		defense:    defense,
		resistance: resistance,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	bodyRow := resistance.bodyGrid()
	bodyRow.Hide()
	torsoRow := resistance.torsoGrid()
	torsoRow.Hide()
	var drToggleBtn *widget.Button
	drToggleBtn = widget.NewButton("Body ▸", func() {
		if bodyRow.Visible() {
			bodyRow.Hide()
			drToggleBtn.SetText("Body ▸")
		} else {
			bodyRow.Show()
			drToggleBtn.SetText("Body ▾")
		}
		row.root.Refresh()
	})
	drToggleBtn.Importance = widget.LowImportance
	applyTorsoOnlyMode := func(enabled bool) {
		resetEntry := func(e *widget.Entry) {
			e.SetText("0")
			if enabled {
				e.Disable()
				return
			}
			e.Enable()
		}
		for _, e := range resistance.nonTorsoEntries() {
			resetEntry(e)
		}
		if enabled {
			bodyRow.Hide()
			drToggleBtn.Hide()
			torsoRow.Show()
			drToggleBtn.SetText("Body ▸")
		} else {
			drToggleBtn.Show()
			torsoRow.Hide()
		}
	}
	torsoOnly.OnChanged = func(checked bool) {
		applyTorsoOnlyMode(checked)
		notifyChange()
	}
	side.OnChanged = func(value string) {
		if row.linkedParty {
			return
		}
		if value == "party" {
			row.number.SetText("1")
			row.number.Disable()
			row.xp.SetText("0")
			row.xp.Disable()
			torsoOnly.SetChecked(false)
			notifyChange()
			return
		}
		row.number.Enable()
		row.xp.Enable()
		if strings.TrimSpace(name.Text) == "" {
			torsoOnly.SetChecked(true)
		}
		notifyChange()
	}
	if defaultSide == "npc" {
		torsoOnly.SetChecked(true)
	}
	side.SetSelected(defaultSide)

	baseRow := container.NewGridWithColumns(
		13,
		name,
		side,
		torsoOnly,
		number,
		level,
		xp,
		initiative,
		hp,
		hpMax,
		defense,
		resistance.globalCell(domain.DamagePoison),
		drToggleBtn,
		removeBtn,
	)
	row.root = container.NewVBox(baseRow, torsoRow, bodyRow)
	return row
}

func newCampaignPlayerInputRow(onRemove func(*campaignPlayerInputRow)) *campaignPlayerInputRow {
	playerName := widget.NewEntry()
	playerName.SetPlaceHolder("Player")
	playerName.TextStyle = fyne.TextStyle{Monospace: true}

	characterName := widget.NewEntry()
	characterName.SetPlaceHolder("Character")
	characterName.TextStyle = fyne.TextStyle{Monospace: true}

	level := widget.NewEntry()
	level.SetPlaceHolder("Level")
	level.TextStyle = fyne.TextStyle{Monospace: true}
	level.SetText("1")
	initiative := widget.NewEntry()
	initiative.SetPlaceHolder("Init")
	initiative.TextStyle = fyne.TextStyle{Monospace: true}
	initiative.SetText("1")
	hp := widget.NewEntry()
	hp.SetPlaceHolder("HP")
	hp.TextStyle = fyne.TextStyle{Monospace: true}
	hp.SetText("1")
	hpMax := widget.NewEntry()
	hpMax.SetPlaceHolder("Max HP")
	hpMax.TextStyle = fyne.TextStyle{Monospace: true}
	hpMax.SetText("1")
	defense := widget.NewEntry()
	defense.SetPlaceHolder("Defense")
	defense.TextStyle = fyne.TextStyle{Monospace: true}
	defense.SetText("0")
	resistance := newUIResistanceInputs(nil)

	row := &campaignPlayerInputRow{
		playerName:    playerName,
		characterName: characterName,
		level:         level,
		initiative:    initiative,
		hp:            hp,
		hpMax:         hpMax,
		defense:       defense,
		resistance:    resistance,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	active := widget.NewCheck("", nil)
	active.SetChecked(true)
	row.active = active
	bodyRow := resistance.bodyGrid()
	bodyRow.Hide()
	var drToggleBtn *widget.Button
	drToggleBtn = widget.NewButton("Body ▸", func() {
		if bodyRow.Visible() {
			bodyRow.Hide()
			drToggleBtn.SetText("Body ▸")
		} else {
			bodyRow.Show()
			drToggleBtn.SetText("Body ▾")
		}
		row.root.Refresh()
	})
	drToggleBtn.Importance = widget.LowImportance
	baseRow := container.NewGridWithColumns(
		11,
		playerName,
		characterName,
		level,
		initiative,
		hp,
		hpMax,
		defense,
		resistance.globalCell(domain.DamagePoison),
		drToggleBtn,
		active,
		removeBtn,
	)
	row.root = container.NewVBox(baseRow, bodyRow)
	return row
}
