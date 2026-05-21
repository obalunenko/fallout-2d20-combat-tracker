package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type encounterOrderView struct {
	enc                 **domain.Encounter
	selectedIndex       *int
	expandedCombatantID *string
	selectedLabel       *widget.Label
	list                *widget.List
	orderBox            *fyne.Container
	showApplyDamage     func(int)
	showHeal            func(int)
}

func newEncounterOrderView(
	enc **domain.Encounter,
	selectedIndex *int,
	expandedCombatantID *string,
	selectedLabel *widget.Label,
	showApplyDamage func(int),
	showHeal func(int),
) *encounterOrderView {
	v := &encounterOrderView{
		enc:                 enc,
		selectedIndex:       selectedIndex,
		expandedCombatantID: expandedCombatantID,
		selectedLabel:       selectedLabel,
		orderBox:            container.NewVBox(),
		showApplyDamage:     showApplyDamage,
		showHeal:            showHeal,
	}
	v.list = widget.NewList(
		func() int {
			current := v.currentEncounter()
			if current == nil {
				return 0
			}
			return len(current.Combatants)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			current := v.currentEncounter()
			if current == nil || i >= len(current.Combatants) {
				return
			}
			label := o.(*widget.Label)
			c := current.Combatants[i]
			label.SetText(formatCombatantLine(current, c))
			if c.Defeated || c.HP <= 0 {
				label.Importance = widget.LowImportance
				label.TextStyle = fyne.TextStyle{Monospace: true, Italic: true}
				return
			}
			label.Importance = widget.MediumImportance
			label.TextStyle = fyne.TextStyle{Monospace: true}
		},
	)
	v.list.OnSelected = func(id widget.ListItemID) {
		*v.selectedIndex = id
		v.CollapseDetails()
		refreshSelected(v.selectedLabel, v.currentEncounter(), *v.selectedIndex)
	}
	return v
}

func (v *encounterOrderView) List() *widget.List {
	return v.list
}

func (v *encounterOrderView) OrderBox() fyne.CanvasObject {
	return v.orderBox
}

func (v *encounterOrderView) Rebuild() {
	v.orderBox.Objects = nil
	current := v.currentEncounter()
	if current == nil || len(current.Combatants) == 0 {
		empty := widget.NewLabel("No combatants")
		empty.TextStyle = fyne.TextStyle{Monospace: true}
		v.orderBox.Add(empty)
		v.orderBox.Refresh()
		return
	}

	for i, c := range current.Combatants {
		idx := i
		combatantID := c.ID
		lineBtn := widget.NewButton(formatCombatantLine(current, c), func() {
			*v.selectedIndex = idx
			refreshSelected(v.selectedLabel, current, *v.selectedIndex)
			if *v.expandedCombatantID == combatantID {
				*v.expandedCombatantID = ""
			} else {
				*v.expandedCombatantID = combatantID
			}
			v.Rebuild()
		})
		isDefeated := c.Defeated || c.HP <= 0
		switch {
		case isDefeated:
			lineBtn.Importance = widget.LowImportance
		case c.Active:
			lineBtn.Importance = widget.MediumImportance
		default:
			lineBtn.Importance = widget.LowImportance
		}

		damageBtn := widget.NewButton("DMG", func() {
			v.CollapseDetails()
			v.showApplyDamage(idx)
		})
		healBtn := widget.NewButton("HEAL", func() {
			v.CollapseDetails()
			v.showHeal(idx)
		})
		damageBtn.Importance = widget.LowImportance
		healBtn.Importance = widget.LowImportance

		details := widget.NewLabel(formatExpandedCombatantDetails(current, c))
		details.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		details.Wrapping = fyne.TextWrapWord
		details.Importance = widget.HighImportance
		detailsSection := container.NewVBox(lineBtn)
		if *v.expandedCombatantID == combatantID {
			detailsSection.Add(details)
		}

		row := container.NewBorder(nil, nil, nil, container.NewGridWithColumns(2, damageBtn, healBtn), detailsSection)
		v.orderBox.Add(row)
	}
	v.orderBox.Refresh()
}

func (v *encounterOrderView) CollapseDetails() {
	if *v.expandedCombatantID == "" {
		return
	}
	*v.expandedCombatantID = ""
	v.Rebuild()
}

func (v *encounterOrderView) currentEncounter() *domain.Encounter {
	if v.enc == nil {
		return nil
	}
	return *v.enc
}
