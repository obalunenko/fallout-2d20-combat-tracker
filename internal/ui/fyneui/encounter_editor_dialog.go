package fyneui

import (
	"context"
	"fmt"
	"strconv"
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
	loadMonsterBtn := widget.NewButton("Load Monster From DB", func() {
		validationError.SetText("")

		monsters, err := svc.ListMonsterTemplates(ctx)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}
		if len(monsters) == 0 {
			validationError.SetText("No saved monsters found in database")
			return
		}

		selectedIdx := 0
		selectedInfo := widget.NewLabel(formatMonsterTemplateOption(monsters[0]))
		selectedInfo.Wrapping = fyne.TextWrapWord
		countEntry := widget.NewEntry()
		countEntry.SetText("1")
		countEntry.SetPlaceHolder("Count")
		monsterList := widget.NewList(
			func() int { return len(monsters) },
			func() fyne.CanvasObject { return widget.NewLabel("monster") },
			func(i widget.ListItemID, o fyne.CanvasObject) {
				if i < 0 || i >= len(monsters) {
					return
				}
				o.(*widget.Label).SetText(formatMonsterTemplateOption(monsters[i]))
			},
		)
		monsterList.OnSelected = func(id widget.ListItemID) {
			if id < 0 || id >= len(monsters) {
				return
			}
			selectedIdx = id
			selectedInfo.SetText(formatMonsterTemplateOption(monsters[id]))
		}
		monsterList.Select(0)

		content := container.NewBorder(
			nil,
			container.NewVBox(widget.NewForm(widget.NewFormItem("Number", countEntry)), selectedInfo),
			nil,
			nil,
			monsterList,
		)
		dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
		content.Resize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))

		var monsterDialog *dialog.CustomDialog
		cancelBtn := widget.NewButton("Cancel", func() {
			monsterDialog.Hide()
		})
		addBtn := widget.NewButton("Add", func() {
			countText := strings.TrimSpace(countEntry.Text)
			if countText == "" {
				countText = "1"
			}
			count, parseErr := strconv.Atoi(countText)
			if parseErr != nil || count < 1 {
				validationError.SetText(fmt.Sprintf("invalid monster number %q", countText))
				return
			}
			template := monsters[selectedIdx]
			template.ID = ""
			template.PlayerCharacterID = ""
			template.Side = domain.SideNPC

			var row *combatantInputRow
			if len(rows) == 1 && combatantInputRowIsEmpty(rows[0]) {
				row = rows[0]
			} else {
				row = addRow("npc")
			}
			fillCombatantInputRow(row, template, domain.SideNPC, count)
			refreshDifficultyPreview()
			monsterDialog.Hide()
		})
		monsterDialog = dialog.NewCustomWithoutButtons("Monster Library", content, w)
		monsterDialog.SetButtons([]fyne.CanvasObject{cancelBtn, addBtn})
		monsterDialog.Resize(dialogSize)
		monsterDialog.Show()
	})
	saveMonstersBtn := widget.NewButton("Save NPCs To DB", func() {
		validationError.SetText("")
		combatants, err := collectCombatantsFromRows(rows)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}
		monsters := make([]domain.Combatant, 0, len(combatants))
		for _, c := range combatants {
			if c.Side != domain.SideNPC {
				continue
			}
			c.ID = ""
			c.PlayerCharacterID = ""
			c.Active = false
			c.Defeated = false
			monsters = append(monsters, c)
		}
		if len(monsters) == 0 {
			validationError.SetText("No NPC combatants to save")
			return
		}
		saved, err := svc.SaveMonsterTemplates(ctx, monsters)
		if err != nil {
			validationError.SetText(err.Error())
			return
		}
		validationError.SetText(fmt.Sprintf("Saved %d monster template(s)", len(saved)))
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
		container.NewGridWithColumns(4, addCombatantBtn, loadMonsterBtn, saveMonstersBtn, loadPartyBtn),
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

func formatMonsterTemplateOption(c domain.Combatant) string {
	return fmt.Sprintf(
		"%s | Lvl:%d XP:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
		c.Name,
		c.Level,
		c.XP,
		c.Initiative,
		c.HP,
		c.MaxHP,
		c.Defense,
		formatDRValue(c.ResistPoison, c.ImmunePoison),
	)
}
