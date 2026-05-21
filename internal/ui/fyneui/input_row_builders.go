package fyneui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
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

	side := widget.NewSelect([]string{"party", "npc"}, nil)
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
	drEnergyHead := widget.NewEntry()
	drEnergyHead.SetPlaceHolder("DRE H")
	drEnergyHead.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyHead.SetText("0")
	drEnergyHead.OnChanged = func(string) { notifyChange() }
	drEnergyTorso := widget.NewEntry()
	drEnergyTorso.SetPlaceHolder("DRE T")
	drEnergyTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyTorso.SetText("0")
	drEnergyTorso.OnChanged = func(string) { notifyChange() }
	drEnergyLA := widget.NewEntry()
	drEnergyLA.SetPlaceHolder("DRE LA")
	drEnergyLA.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyLA.SetText("0")
	drEnergyLA.OnChanged = func(string) { notifyChange() }
	drEnergyRA := widget.NewEntry()
	drEnergyRA.SetPlaceHolder("DRE RA")
	drEnergyRA.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyRA.SetText("0")
	drEnergyRA.OnChanged = func(string) { notifyChange() }
	drEnergyLL := widget.NewEntry()
	drEnergyLL.SetPlaceHolder("DRE LL")
	drEnergyLL.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyLL.SetText("0")
	drEnergyLL.OnChanged = func(string) { notifyChange() }
	drEnergyRL := widget.NewEntry()
	drEnergyRL.SetPlaceHolder("DRE RL")
	drEnergyRL.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyRL.SetText("0")
	drEnergyRL.OnChanged = func(string) { notifyChange() }
	drRadHead := widget.NewEntry()
	drRadHead.SetPlaceHolder("DRR H")
	drRadHead.TextStyle = fyne.TextStyle{Monospace: true}
	drRadHead.SetText("0")
	drRadHead.OnChanged = func(string) { notifyChange() }
	drRadTorso := widget.NewEntry()
	drRadTorso.SetPlaceHolder("DRR T")
	drRadTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drRadTorso.SetText("0")
	drRadTorso.OnChanged = func(string) { notifyChange() }
	drRadLA := widget.NewEntry()
	drRadLA.SetPlaceHolder("DRR LA")
	drRadLA.TextStyle = fyne.TextStyle{Monospace: true}
	drRadLA.SetText("0")
	drRadLA.OnChanged = func(string) { notifyChange() }
	drRadRA := widget.NewEntry()
	drRadRA.SetPlaceHolder("DRR RA")
	drRadRA.TextStyle = fyne.TextStyle{Monospace: true}
	drRadRA.SetText("0")
	drRadRA.OnChanged = func(string) { notifyChange() }
	drRadLL := widget.NewEntry()
	drRadLL.SetPlaceHolder("DRR LL")
	drRadLL.TextStyle = fyne.TextStyle{Monospace: true}
	drRadLL.SetText("0")
	drRadLL.OnChanged = func(string) { notifyChange() }
	drRadRL := widget.NewEntry()
	drRadRL.SetPlaceHolder("DRR RL")
	drRadRL.TextStyle = fyne.TextStyle{Monospace: true}
	drRadRL.SetText("0")
	drRadRL.OnChanged = func(string) { notifyChange() }
	drPhysHead := widget.NewEntry()
	drPhysHead.SetPlaceHolder("DRP H")
	drPhysHead.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysHead.SetText("0")
	drPhysHead.OnChanged = func(string) { notifyChange() }
	drPhysTorso := widget.NewEntry()
	drPhysTorso.SetPlaceHolder("DRP T")
	drPhysTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysTorso.SetText("0")
	drPhysTorso.OnChanged = func(string) { notifyChange() }
	drPhysLA := widget.NewEntry()
	drPhysLA.SetPlaceHolder("DRP LA")
	drPhysLA.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysLA.SetText("0")
	drPhysLA.OnChanged = func(string) { notifyChange() }
	drPhysRA := widget.NewEntry()
	drPhysRA.SetPlaceHolder("DRP RA")
	drPhysRA.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysRA.SetText("0")
	drPhysRA.OnChanged = func(string) { notifyChange() }
	drPhysLL := widget.NewEntry()
	drPhysLL.SetPlaceHolder("DRP LL")
	drPhysLL.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysLL.SetText("0")
	drPhysLL.OnChanged = func(string) { notifyChange() }
	drPhysRL := widget.NewEntry()
	drPhysRL.SetPlaceHolder("DRP RL")
	drPhysRL.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysRL.SetText("0")
	drPhysRL.OnChanged = func(string) { notifyChange() }
	drPoison := widget.NewEntry()
	drPoison.SetPlaceHolder("DR Poison")
	drPoison.TextStyle = fyne.TextStyle{Monospace: true}
	drPoison.SetText("0")
	drPoison.OnChanged = func(string) { notifyChange() }

	immPhysical := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drPhysHead, drPhysTorso, drPhysLA, drPhysRA, drPhysLL, drPhysRL},
		notifyChange,
	)
	drPoisonCell, immPoison := newResistanceInputCell(drPoison, notifyChange)
	immEnergy := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drEnergyHead, drEnergyTorso, drEnergyLA, drEnergyRA, drEnergyLL, drEnergyRL},
		notifyChange,
	)
	immRadiation := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drRadHead, drRadTorso, drRadLA, drRadRA, drRadLL, drRadRL},
		notifyChange,
	)

	row := &combatantInputRow{
		name:          name,
		side:          side,
		torsoOnly:     torsoOnly,
		number:        number,
		level:         level,
		xp:            xp,
		initiative:    initiative,
		hp:            hp,
		hpMax:         hpMax,
		defense:       defense,
		drEnergyHead:  drEnergyHead,
		drEnergyTorso: drEnergyTorso,
		drEnergyLA:    drEnergyLA,
		drEnergyRA:    drEnergyRA,
		drEnergyLL:    drEnergyLL,
		drEnergyRL:    drEnergyRL,
		drRadHead:     drRadHead,
		drRadTorso:    drRadTorso,
		drRadLA:       drRadLA,
		drRadRA:       drRadRA,
		drRadLL:       drRadLL,
		drRadRL:       drRadRL,
		drPhysHead:    drPhysHead,
		drPhysTorso:   drPhysTorso,
		drPhysLA:      drPhysLA,
		drPhysRA:      drPhysRA,
		drPhysLL:      drPhysLL,
		drPhysRL:      drPhysRL,
		drPoison:      drPoison,
		immPhysical:   immPhysical,
		immEnergy:     immEnergy,
		immRadiation:  immRadiation,
		immPoison:     immPoison,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	drPartLabel := func(text string) *widget.Label {
		l := widget.NewLabel(text)
		l.TextStyle = fyne.TextStyle{Monospace: true}
		return l
	}
	bodyRow := container.NewVBox(
		container.NewGridWithColumns(
			4,
			newTableHeaderLabel("Body Part"),
			newTableHeaderLabel("Physical"),
			newTableHeaderLabel("Energy"),
			newTableHeaderLabel("Radiation"),
		),
		container.NewGridWithColumns(4, drPartLabel("Immune"), immPhysical, immEnergy, immRadiation),
		container.NewGridWithColumns(4, drPartLabel("Head"), drPhysHead, drEnergyHead, drRadHead),
		container.NewGridWithColumns(4, drPartLabel("Torso"), drPhysTorso, drEnergyTorso, drRadTorso),
		container.NewGridWithColumns(4, drPartLabel("Left Arm"), drPhysLA, drEnergyLA, drRadLA),
		container.NewGridWithColumns(4, drPartLabel("Right Arm"), drPhysRA, drEnergyRA, drRadRA),
		container.NewGridWithColumns(4, drPartLabel("Left Leg"), drPhysLL, drEnergyLL, drRadLL),
		container.NewGridWithColumns(4, drPartLabel("Right Leg"), drPhysRL, drEnergyRL, drRadRL),
	)
	bodyRow.Hide()
	torsoRow := container.NewGridWithColumns(
		3,
		newTableHeaderLabel("DR Physical"),
		newTableHeaderLabel("DR Energy"),
		newTableHeaderLabel("DR Radiation"),
		drPhysTorso,
		drEnergyTorso,
		drRadTorso,
	)
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
		for _, e := range []*widget.Entry{
			drPhysHead, drPhysLA, drPhysRA, drPhysLL, drPhysRL,
			drEnergyHead, drEnergyLA, drEnergyRA, drEnergyLL, drEnergyRL,
			drRadHead, drRadLA, drRadRA, drRadLL, drRadRL,
		} {
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
		drPoisonCell,
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
	drEnergyHead := widget.NewEntry()
	drEnergyHead.SetPlaceHolder("DRE H")
	drEnergyHead.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyHead.SetText("0")
	drEnergyTorso := widget.NewEntry()
	drEnergyTorso.SetPlaceHolder("DRE T")
	drEnergyTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyTorso.SetText("0")
	drEnergyLA := widget.NewEntry()
	drEnergyLA.SetPlaceHolder("DRE LA")
	drEnergyLA.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyLA.SetText("0")
	drEnergyRA := widget.NewEntry()
	drEnergyRA.SetPlaceHolder("DRE RA")
	drEnergyRA.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyRA.SetText("0")
	drEnergyLL := widget.NewEntry()
	drEnergyLL.SetPlaceHolder("DRE LL")
	drEnergyLL.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyLL.SetText("0")
	drEnergyRL := widget.NewEntry()
	drEnergyRL.SetPlaceHolder("DRE RL")
	drEnergyRL.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergyRL.SetText("0")
	drRadHead := widget.NewEntry()
	drRadHead.SetPlaceHolder("DRR H")
	drRadHead.TextStyle = fyne.TextStyle{Monospace: true}
	drRadHead.SetText("0")
	drRadTorso := widget.NewEntry()
	drRadTorso.SetPlaceHolder("DRR T")
	drRadTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drRadTorso.SetText("0")
	drRadLA := widget.NewEntry()
	drRadLA.SetPlaceHolder("DRR LA")
	drRadLA.TextStyle = fyne.TextStyle{Monospace: true}
	drRadLA.SetText("0")
	drRadRA := widget.NewEntry()
	drRadRA.SetPlaceHolder("DRR RA")
	drRadRA.TextStyle = fyne.TextStyle{Monospace: true}
	drRadRA.SetText("0")
	drRadLL := widget.NewEntry()
	drRadLL.SetPlaceHolder("DRR LL")
	drRadLL.TextStyle = fyne.TextStyle{Monospace: true}
	drRadLL.SetText("0")
	drRadRL := widget.NewEntry()
	drRadRL.SetPlaceHolder("DRR RL")
	drRadRL.TextStyle = fyne.TextStyle{Monospace: true}
	drRadRL.SetText("0")
	drPhysHead := widget.NewEntry()
	drPhysHead.SetPlaceHolder("DRP H")
	drPhysHead.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysHead.SetText("0")
	drPhysTorso := widget.NewEntry()
	drPhysTorso.SetPlaceHolder("DRP T")
	drPhysTorso.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysTorso.SetText("0")
	drPhysLA := widget.NewEntry()
	drPhysLA.SetPlaceHolder("DRP LA")
	drPhysLA.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysLA.SetText("0")
	drPhysRA := widget.NewEntry()
	drPhysRA.SetPlaceHolder("DRP RA")
	drPhysRA.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysRA.SetText("0")
	drPhysLL := widget.NewEntry()
	drPhysLL.SetPlaceHolder("DRP LL")
	drPhysLL.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysLL.SetText("0")
	drPhysRL := widget.NewEntry()
	drPhysRL.SetPlaceHolder("DRP RL")
	drPhysRL.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysRL.SetText("0")
	drPoison := widget.NewEntry()
	drPoison.SetPlaceHolder("DR Poison")
	drPoison.TextStyle = fyne.TextStyle{Monospace: true}
	drPoison.SetText("0")

	immPhysical := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drPhysHead, drPhysTorso, drPhysLA, drPhysRA, drPhysLL, drPhysRL},
		nil,
	)
	drPoisonCell, immPoison := newResistanceInputCell(drPoison, nil)
	immEnergy := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drEnergyHead, drEnergyTorso, drEnergyLA, drEnergyRA, drEnergyLL, drEnergyRL},
		nil,
	)
	immRadiation := newGlobalImmunityCheck(
		"immune all",
		[]*widget.Entry{drRadHead, drRadTorso, drRadLA, drRadRA, drRadLL, drRadRL},
		nil,
	)

	row := &campaignPlayerInputRow{
		playerName:    playerName,
		characterName: characterName,
		level:         level,
		initiative:    initiative,
		hp:            hp,
		hpMax:         hpMax,
		defense:       defense,
		drEnergyHead:  drEnergyHead,
		drEnergyTorso: drEnergyTorso,
		drEnergyLA:    drEnergyLA,
		drEnergyRA:    drEnergyRA,
		drEnergyLL:    drEnergyLL,
		drEnergyRL:    drEnergyRL,
		drRadHead:     drRadHead,
		drRadTorso:    drRadTorso,
		drRadLA:       drRadLA,
		drRadRA:       drRadRA,
		drRadLL:       drRadLL,
		drRadRL:       drRadRL,
		drPhysHead:    drPhysHead,
		drPhysTorso:   drPhysTorso,
		drPhysLA:      drPhysLA,
		drPhysRA:      drPhysRA,
		drPhysLL:      drPhysLL,
		drPhysRL:      drPhysRL,
		drPoison:      drPoison,
		immPhysical:   immPhysical,
		immEnergy:     immEnergy,
		immRadiation:  immRadiation,
		immPoison:     immPoison,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	activeLabel := widget.NewLabel("yes")
	activeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	drPartLabel := func(text string) *widget.Label {
		l := widget.NewLabel(text)
		l.TextStyle = fyne.TextStyle{Monospace: true}
		return l
	}
	bodyRow := container.NewVBox(
		container.NewGridWithColumns(
			4,
			newTableHeaderLabel("Body Part"),
			newTableHeaderLabel("Physical"),
			newTableHeaderLabel("Energy"),
			newTableHeaderLabel("Radiation"),
		),
		container.NewGridWithColumns(4, drPartLabel("Immune"), immPhysical, immEnergy, immRadiation),
		container.NewGridWithColumns(4, drPartLabel("Head"), drPhysHead, drEnergyHead, drRadHead),
		container.NewGridWithColumns(4, drPartLabel("Torso"), drPhysTorso, drEnergyTorso, drRadTorso),
		container.NewGridWithColumns(4, drPartLabel("Left Arm"), drPhysLA, drEnergyLA, drRadLA),
		container.NewGridWithColumns(4, drPartLabel("Right Arm"), drPhysRA, drEnergyRA, drRadRA),
		container.NewGridWithColumns(4, drPartLabel("Left Leg"), drPhysLL, drEnergyLL, drRadLL),
		container.NewGridWithColumns(4, drPartLabel("Right Leg"), drPhysRL, drEnergyRL, drRadRL),
	)
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
		12,
		playerName,
		characterName,
		level,
		initiative,
		hp,
		hpMax,
		defense,
		drPoisonCell,
		drToggleBtn,
		activeLabel,
		removeBtn,
	)
	row.root = container.NewVBox(baseRow, bodyRow)
	return row
}
