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
				resetCombatantInputRow(target, domain.SideNPC)
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
		addRow("npc")
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

		rowsByID := make(map[string]*combatantInputRow, len(rows))
		for _, row := range rows {
			if row.linkedParty && strings.TrimSpace(row.playerCharacterID) != "" {
				rowsByID[row.playerCharacterID] = row
			}
		}

		nextEmpty := -1
		if len(rows) == 1 && combatantInputRowIsEmpty(rows[0]) {
			nextEmpty = 0
		}
		loaded := 0
		for _, member := range partyMembers {
			memberID := strings.TrimSpace(member.ID)
			if memberID == "" {
				continue
			}
			if existing := rowsByID[memberID]; existing != nil {
				fillCombatantInputRow(existing, member, domain.SideParty, 1)
				loaded++
				continue
			}
			var row *combatantInputRow
			if nextEmpty >= 0 {
				row = rows[nextEmpty]
				nextEmpty = -1
			} else {
				row = addRow("party")
			}
			fillCombatantInputRow(row, member, domain.SideParty, 1)
			loaded++
		}
		if loaded == 0 {
			validationError.SetText("No party members with IDs found in database")
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

func showCreateEncounterDialogWindow(ctx context.Context, w fyne.Window, svc *appsvc.Service, refresh func()) {
	showEncounterEditorDialog(
		ctx,
		w,
		svc,
		"Create Encounter",
		"Create",
		"",
		nil,
		func(name string, combatants []domain.Combatant) error {
			_, err := svc.ExecuteCreateEncounter(ctx, appsvc.CreateEncounterCommand{
				Name:       name,
				Combatants: combatants,
			})
			return err
		},
		refresh,
	)
}
