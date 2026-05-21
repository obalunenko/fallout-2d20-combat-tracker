package fyneui

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func showEncounterEditorDialog(
	ctx context.Context,
	w fyne.Window,
	svc *appsvc.Service,
	title string,
	submitLabel string,
	initialName string,
	initialCombatants []domain.Combatant,
	onSubmit func(name string, combatants []domain.Combatant) error,
	refresh func(),
) {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g. Red Rocket Defense")
	nameEntry.SetText(strings.TrimSpace(initialName))

	var rows []*combatantInputRow
	rowsBox := container.NewVBox()
	difficultyPreview := widget.NewLabel("Difficulty: Unknown")
	difficultyPreview.TextStyle = fyne.TextStyle{Monospace: true}
	difficultyPreview.Wrapping = fyne.TextWrapWord

	refreshDifficultyPreview := func() {
		preview := collectCombatantsPreviewFromRows(rows)
		metrics := domain.EvaluateEncounterDifficulty(preview)
		difficultyPreview.SetText(formatDifficultyPreview(metrics))
	}
	headers := container.NewGridWithColumns(
		13,
		newTableHeaderLabel("Name"),
		newTableHeaderLabel("Side"),
		newTableHeaderLabel("Torso"),
		newTableHeaderLabel("Number"),
		newTableHeaderLabel("Level"),
		newTableHeaderLabel("XP"),
		newTableHeaderLabel("Initiative"),
		newTableHeaderLabel("HP Cur"),
		newTableHeaderLabel("HP Max"),
		newTableHeaderLabel("DEF Base"),
		newTableHeaderLabel("DR Poison"),
		newTableHeaderLabel("DR Details"),
		newTableHeaderLabel("Action"),
	)
	table := container.NewVBox(headers, widget.NewSeparator(), rowsBox)

	addRow := func(defaultSide string) *combatantInputRow {
		row := newCombatantInputRow(defaultSide, func(target *combatantInputRow) {
			if len(rows) == 1 {
				target.name.SetText("")
				target.number.SetText("1")
				target.level.SetText("1")
				target.xp.SetText("0")
				target.initiative.SetText("")
				target.hp.SetText("1")
				target.hpMax.SetText("1")
				target.defense.SetText("0")
				target.defenseHead.SetText("0")
				target.defenseTorso.SetText("0")
				target.defenseLA.SetText("0")
				target.defenseRA.SetText("0")
				target.defenseLL.SetText("0")
				target.defenseRL.SetText("0")
				target.drEnergyHead.SetText("0")
				target.drEnergyTorso.SetText("0")
				target.drEnergyLA.SetText("0")
				target.drEnergyRA.SetText("0")
				target.drEnergyLL.SetText("0")
				target.drEnergyRL.SetText("0")
				target.drRadHead.SetText("0")
				target.drRadTorso.SetText("0")
				target.drRadLA.SetText("0")
				target.drRadRA.SetText("0")
				target.drRadLL.SetText("0")
				target.drRadRL.SetText("0")
				target.drPhysHead.SetText("0")
				target.drPhysTorso.SetText("0")
				target.drPhysLA.SetText("0")
				target.drPhysRA.SetText("0")
				target.drPhysLL.SetText("0")
				target.drPhysRL.SetText("0")
				target.immPhysical.SetChecked(false)
				target.immEnergy.SetChecked(false)
				target.immRadiation.SetChecked(false)
				target.drPoison.SetText("0")
				target.immPoison.SetChecked(false)
				target.side.SetSelected(defaultSide)
				target.torsoOnly.SetChecked(defaultSide == "npc")
				refreshDifficultyPreview()
				return
			}
			filtered := make([]*combatantInputRow, 0, len(rows)-1)
			for _, r := range rows {
				if r != target {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
			rowsBox.Remove(target.root)
			rowsBox.Refresh()
			refreshDifficultyPreview()
		}, refreshDifficultyPreview)
		rows = append(rows, row)
		rowsBox.Add(row.root)
		rowsBox.Refresh()
		refreshDifficultyPreview()
		return row
	}

	if len(initialCombatants) == 0 {
		addRow("party")
	} else {
		for _, c := range initialCombatants {
			side := string(c.Side)
			if side != "npc" {
				side = "party"
			}
			row := addRow(side)
			fillCombatantInputRow(row, c, c.Side, 1)
		}
	}
	refreshDifficultyPreview()

	validationError := widget.NewLabel("")
	validationError.TextStyle = fyne.TextStyle{Monospace: true}
	validationError.Wrapping = fyne.TextWrapWord

	addCombatantBtn := widget.NewButton("+ Add Combatant", func() {
		addRow("npc")
	})
	loadPartyBtn := widget.NewButton("Load Party From DB", func() {
		validationError.SetText("")

		partyMembers, err := svc.ListPartyMembers(ctx)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}
		if len(partyMembers) == 0 {
			validationError.SetText("No saved party members found in database")
			return
		}

		next := 0
		if len(rows) == 1 && combatantInputRowIsEmpty(rows[0]) {
			fillCombatantInputRow(rows[0], partyMembers[0], domain.SideParty, 1)
			next = 1
		}
		for i := next; i < len(partyMembers); i++ {
			row := addRow("party")
			fillCombatantInputRow(row, partyMembers[i], domain.SideParty, 1)
		}
		refreshDifficultyPreview()
	})

	dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
	scroll := container.NewScroll(table)
	scroll.Direction = container.ScrollBoth
	scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))
	combatantsSection := container.NewVBox(
		container.NewGridWithColumns(2, addCombatantBtn, loadPartyBtn),
		difficultyPreview,
		scroll,
	)

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("Combatants", combatantsSection),
	)
	formContent := container.NewVBox(form, widget.NewSeparator(), validationError)

	var editorDialog *dialog.CustomDialog
	cancelBtn := widget.NewButton("Cancel", func() {
		editorDialog.Hide()
	})
	submitBtn := widget.NewButton(submitLabel, func() {
		validationError.SetText("")

		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			validationError.SetText("Encounter name is required")
			return
		}

		combatants, err := collectCombatantsFromRows(rows)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}

		if err := onSubmit(name, combatants); err != nil {
			validationError.SetText(err.Error())
			return
		}

		if refresh != nil {
			refresh()
		}
		editorDialog.Hide()
	})

	editorDialog = dialog.NewCustomWithoutButtons(title, formContent, w)
	editorDialog.SetButtons([]fyne.CanvasObject{cancelBtn, submitBtn})
	editorDialog.Resize(dialogSize)
	editorDialog.Show()
}
