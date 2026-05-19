package fyneui

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/google/uuid"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func Run(svc *appsvc.Service, onShutdown func()) error {
	a := fyneapp.New()
	a.Settings().SetTheme(newPipBoyTheme())
	shutdown := shutdownOnce(onShutdown)
	a.Lifecycle().SetOnStopped(shutdown)
	stopSignals := installSignalShutdown(a, shutdown)
	defer stopSignals()

	w := a.NewWindow("Fallout 2d20 Combat Tracker")
	w.Resize(fyne.NewSize(1100, 700))

	var (
		enc           *domain.Encounter
		selectedIndex int
	)

	roundLabel := widget.NewLabel("")
	activeLabel := widget.NewLabel("")
	selectedLabel := widget.NewLabel("")
	partyAPLabel := widget.NewLabel("")
	threatLabel := widget.NewLabel("")

	roundLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	activeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	partyAPLabel.TextStyle = fyne.TextStyle{Monospace: true}
	threatLabel.TextStyle = fyne.TextStyle{Monospace: true}

	eventLog := []string{"[BOOT] Pip-Boy combat tracker initialized"}
	logList := widget.NewList(
		func() int { return len(eventLog) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("log entry")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(eventLog) {
				return
			}
			o.(*widget.Label).SetText(eventLog[i])
		},
	)
	setEventLog := func(lines []string) {
		eventLog = lines
		logList.Refresh()
	}

	combatantLine := func(c domain.Combatant) string {
		name := c.Name
		if enc != nil {
			name = encounterDisplayNameByID(enc, c.ID)
		}
		prefix := "   "
		if c.Active {
			prefix = ">> "
		}
		return fmt.Sprintf(
			"%s%s [%s] Lvl:%d XP:%d Init:%d HP:%d DEF:%d DR P/E/R/P:%s/%s/%s/%s",
			prefix, name, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.Defense,
			formatDRValue(c.ResistPhysical, c.ImmunePhysical),
			formatDRValue(c.ResistEnergy, c.ImmuneEnergy),
			formatDRValue(c.ResistRadiation, c.ImmuneRadiation),
			formatDRValue(c.ResistPoison, c.ImmunePoison),
		)
	}

	list := widget.NewList(
		func() int {
			if enc == nil {
				return 0
			}
			return len(enc.Combatants)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return label
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if enc == nil || i >= len(enc.Combatants) {
				return
			}
			o.(*widget.Label).SetText(combatantLine(enc.Combatants[i]))
		},
	)
	list.OnSelected = func(id widget.ListItemID) {
		selectedIndex = id
		refreshSelected(selectedLabel, enc, selectedIndex)
	}

	encounterOrderBox := container.NewVBox()
	rebuildEncounterOrder := func() {
		encounterOrderBox.Objects = nil
		if enc == nil || len(enc.Combatants) == 0 {
			empty := widget.NewLabel("No combatants")
			empty.TextStyle = fyne.TextStyle{Monospace: true}
			encounterOrderBox.Add(empty)
			encounterOrderBox.Refresh()
			return
		}

		for _, c := range enc.Combatants {
			line := widget.NewLabel(combatantLine(c))
			line.TextStyle = fyne.TextStyle{Monospace: true, Bold: c.Active}
			encounterOrderBox.Add(line)
		}
		encounterOrderBox.Refresh()
	}

	handleErr := func(err error) {
		if err != nil {
			dialog.ShowError(err, w)
		}
	}

	refreshResources := func() {
		if enc == nil {
			partyAPLabel.SetText("Party AP: 0")
			threatLabel.SetText("GM Threat: 0")
			return
		}
		partyAPLabel.SetText(fmt.Sprintf("Party AP: %d", enc.Resources.PartyAP))
		threatLabel.SetText(fmt.Sprintf("GM Threat: %d", enc.Resources.GMThreat))
	}

	var refresh func()
	var showCreateEncounterDialog func()
	var showEncounterListDialog func()
	var showApplyDamageDialogForIndex func(int)
	var showHealDialogForIndex func(int)
	var refreshDataLog func()

	nextTurnBtn := widget.NewButton("Next Turn", func() {
		_, err := svc.AdvanceTurn()
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	partyAddBtn := widget.NewButton("+ AP", func() {
		_, err := svc.AddPartyAP(1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	partySpendBtn := widget.NewButton("- AP", func() {
		_, err := svc.SpendPartyAP(1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	threatAddBtn := widget.NewButton("+ Threat", func() {
		_, err := svc.AddThreat(1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	threatSpendBtn := widget.NewButton("- Threat", func() {
		_, err := svc.SpendThreat(1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	applyDamageBtn := widget.NewButton("APPLY DAMAGE", func() {
		showApplyDamageDialogForIndex(selectedIndex)
	})
	healBtn := widget.NewButton("HEAL", func() {
		showHealDialogForIndex(selectedIndex)
	})
	applyDamageActiveBtn := widget.NewButton("APPLY DAMAGE", func() {
		targetIndex := selectedIndex
		if enc != nil && len(enc.Combatants) > 0 {
			targetIndex = enc.TurnIndex
		}
		showApplyDamageDialogForIndex(targetIndex)
	})
	healActiveBtn := widget.NewButton("HEAL", func() {
		targetIndex := selectedIndex
		if enc != nil && len(enc.Combatants) > 0 {
			targetIndex = enc.TurnIndex
		}
		showHealDialogForIndex(targetIndex)
	})

	turnPanel := pipPanel(
		"TURN CONTROL",
		container.NewVBox(
			roundLabel,
			activeLabel,
			container.NewGridWithColumns(3, nextTurnBtn, applyDamageActiveBtn, healActiveBtn),
		),
	)
	resourcesPanel := pipPanel(
		"RESOURCES",
		container.NewVBox(
			partyAPLabel,
			container.NewGridWithColumns(2, partyAddBtn, partySpendBtn),
			widget.NewSeparator(),
			threatLabel,
			container.NewGridWithColumns(2, threatAddBtn, threatSpendBtn),
		),
	)
	encounterOrderPanel := pipPanel("ENCOUNTER ORDER", encounterOrderBox)
	combatantsPanel := pipPanel("INITIATIVE ORDER", list)
	selectedPanel := pipPanel(
		"SELECTED COMBATANT",
		container.NewVBox(selectedLabel, widget.NewSeparator(), container.NewGridWithColumns(2, applyDamageBtn, healBtn)),
	)
	logPanel := pipPanel("DATA LOG", logList)

	statTabContent := container.NewVBox(
		turnPanel,
		widget.NewSeparator(),
		resourcesPanel,
		widget.NewSeparator(),
		encounterOrderPanel,
	)
	invTabContent := container.NewGridWithColumns(2, combatantsPanel, selectedPanel)
	dataTabContent := container.NewVBox(logPanel)

	tabs := container.NewAppTabs(
		container.NewTabItem("STAT", container.NewPadded(statTabContent)),
		container.NewTabItem("INV", container.NewPadded(invTabContent)),
		container.NewTabItem("DATA", container.NewPadded(dataTabContent)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabsView := container.New(layout.NewPaddedLayout(), tabs)

	newEncounterBtn := widget.NewButton("NEW ENCOUNTER", func() {
		showCreateEncounterDialog()
	})
	openEncounterBtn := widget.NewButton("OPEN ENCOUNTER", func() {
		showEncounterListDialog()
	})

	setupHint := widget.NewLabel("No active encounter.\nCreate one from scratch to begin tracking combat.")
	setupHint.Alignment = fyne.TextAlignCenter
	setupHint.TextStyle = fyne.TextStyle{Monospace: true}
	setupButton := widget.NewButton("CREATE ENCOUNTER", func() {
		showCreateEncounterDialog()
	})
	noEncounterView := pipPanel(
		"SYSTEM",
		container.NewCenter(container.NewVBox(setupHint, widget.NewSeparator(), setupButton)),
	)

	mainView := container.NewStack(tabsView, noEncounterView)

	refreshDataLog = func() {
		if enc == nil {
			setEventLog([]string{"[BOOT] Pip-Boy combat tracker initialized"})
			return
		}

		logs, err := svc.ListEncounterLogs(enc.ID)
		if err != nil {
			handleErr(err)
			return
		}

		if len(logs) == 0 {
			setEventLog([]string{"No operations yet"})
			return
		}

		lines := make([]string, 0, len(logs))
		for _, logEntry := range logs {
			lines = append(lines, fmt.Sprintf("[%s] [R%d] %s",
				formatLogTimestamp(logEntry.CreatedAt), logEntry.Round, logEntry.Message))
		}
		setEventLog(lines)
	}

	refresh = func() {
		var err error
		enc, err = svc.GetEncounter()
		if err != nil {
			if errors.Is(err, domain.ErrEncounterNotInitialized) {
				enc = nil
				roundLabel.SetText("Round: -")
				activeLabel.SetText("Active: -")
				refreshSelected(selectedLabel, nil, 0)
				refreshResources()
				list.Refresh()
				rebuildEncounterOrder()
				refreshDataLog()
				tabsView.Hide()
				noEncounterView.Show()
				mainView.Refresh()
				return
			}
			handleErr(err)
			return
		}

		roundLabel.SetText(fmt.Sprintf("Round: %d", enc.Round))
		if active := enc.ActiveCombatant(); active != nil {
			activeName := encounterDisplayNameByID(enc, active.ID)
			activeLabel.SetText(fmt.Sprintf("Active: %s (%s, Init:%d)", activeName, active.Side, active.Initiative))
		} else {
			activeLabel.SetText("Active: -")
		}

		if selectedIndex >= len(enc.Combatants) {
			selectedIndex = 0
		}
		refreshSelected(selectedLabel, enc, selectedIndex)
		refreshResources()
		list.Refresh()
		rebuildEncounterOrder()
		refreshDataLog()
		noEncounterView.Hide()
		tabsView.Show()
		mainView.Refresh()
	}

	showCreateEncounterDialog = func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("e.g. Red Rocket Defense")

		var rows []*combatantInputRow
		rowsBox := container.NewVBox()
		headers := container.NewGridWithColumns(
			13,
			newTableHeaderLabel("Name"),
			newTableHeaderLabel("Side"),
			newTableHeaderLabel("Number"),
			newTableHeaderLabel("Level"),
			newTableHeaderLabel("XP"),
			newTableHeaderLabel("Initiative"),
			newTableHeaderLabel("HP"),
			newTableHeaderLabel("Defense"),
			newTableHeaderLabel("DR Phys"),
			newTableHeaderLabel("DR Energy"),
			newTableHeaderLabel("DR Rad"),
			newTableHeaderLabel("DR Poison"),
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
					target.defense.SetText("0")
					target.drPhysical.SetText("0")
					target.drEnergy.SetText("0")
					target.drRadiation.SetText("0")
					target.drPoison.SetText("0")
					target.immPhysical.SetChecked(false)
					target.immEnergy.SetChecked(false)
					target.immRadiation.SetChecked(false)
					target.immPoison.SetChecked(false)
					target.side.SetSelected(defaultSide)
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
			})
			rows = append(rows, row)
			rowsBox.Add(row.root)
			rowsBox.Refresh()
			return row
		}

		addRow("party")
		validationError := widget.NewLabel("")
		validationError.TextStyle = fyne.TextStyle{Monospace: true}
		validationError.Wrapping = fyne.TextWrapWord

		addCombatantBtn := widget.NewButton("+ Add Combatant", func() {
			addRow("npc")
		})
		loadPartyBtn := widget.NewButton("Load Party From DB", func() {
			validationError.SetText("")

			partyMembers, err := svc.ListPartyMembers()
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
		})

		dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
		scroll := container.NewScroll(table)
		scroll.Direction = container.ScrollBoth
		scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))
		combatantsSection := container.NewVBox(container.NewGridWithColumns(2, addCombatantBtn, loadPartyBtn), scroll)

		form := widget.NewForm(
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Combatants", combatantsSection),
		)
		formContent := container.NewVBox(form, widget.NewSeparator(), validationError)

		var createDialog *dialog.CustomDialog
		cancelBtn := widget.NewButton("Cancel", func() {
			createDialog.Hide()
		})
		createBtn := widget.NewButton("Create", func() {
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

			encounterID := uuid.NewString()
			if _, err := svc.CreateEncounter(encounterID, name, combatants); err != nil {
				validationError.SetText(err.Error())
				return
			}

			refresh()
			createDialog.Hide()
		})

		createDialog = dialog.NewCustomWithoutButtons("Create Encounter", formContent, w)
		createDialog.SetButtons([]fyne.CanvasObject{cancelBtn, createBtn})
		createDialog.Resize(dialogSize)
		createDialog.Show()
	}

	showApplyDamageDialogForIndex = func(targetIndex int) {
		if enc == nil || len(enc.Combatants) == 0 {
			dialog.ShowError(fmt.Errorf("no combatants in active encounter"), w)
			return
		}
		if targetIndex < 0 || targetIndex >= len(enc.Combatants) {
			targetIndex = 0
		}

		target := enc.Combatants[targetIndex]
		targetDisplayName := encounterDisplayNameByID(enc, target.ID)
		typeSelect := widget.NewSelect([]string{"physical", "energy", "radiation", "poison"}, nil)
		typeSelect.SetSelected("physical")

		amountEntry := widget.NewEntry()
		amountEntry.SetPlaceHolder("Damage amount")
		amountEntry.SetText("1")
		amountEntry.TextStyle = fyne.TextStyle{Monospace: true}

		damageDialog := dialog.NewForm(
			fmt.Sprintf("Apply Damage: %s", targetDisplayName),
			"Apply",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Type", typeSelect),
				widget.NewFormItem("Amount", amountEntry),
			},
			func(ok bool) {
				if !ok {
					return
				}

				damageType, err := parseDamageType(typeSelect.Selected)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				amountText := strings.TrimSpace(amountEntry.Text)
				amount, err := strconv.Atoi(amountText)
				if err != nil || amount < 0 {
					dialog.ShowError(fmt.Errorf("invalid damage amount %q", amountText), w)
					return
				}

				_, _, err = svc.ApplyDamage(target.ID, damageType, amount)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				refresh()
			},
			w,
		)
		damageDialog.Resize(fyne.NewSize(420, 220))
		damageDialog.Show()
	}

	showHealDialogForIndex = func(targetIndex int) {
		if enc == nil || len(enc.Combatants) == 0 {
			dialog.ShowError(fmt.Errorf("no combatants in active encounter"), w)
			return
		}
		if targetIndex < 0 || targetIndex >= len(enc.Combatants) {
			targetIndex = 0
		}

		target := enc.Combatants[targetIndex]
		targetDisplayName := encounterDisplayNameByID(enc, target.ID)
		amountEntry := widget.NewEntry()
		amountEntry.SetPlaceHolder("Heal amount")
		amountEntry.SetText("1")
		amountEntry.TextStyle = fyne.TextStyle{Monospace: true}

		healDialog := dialog.NewForm(
			fmt.Sprintf("Heal: %s", targetDisplayName),
			"Heal",
			"Cancel",
			[]*widget.FormItem{
				widget.NewFormItem("Amount", amountEntry),
			},
			func(ok bool) {
				if !ok {
					return
				}

				amountText := strings.TrimSpace(amountEntry.Text)
				amount, err := strconv.Atoi(amountText)
				if err != nil || amount < 0 {
					dialog.ShowError(fmt.Errorf("invalid heal amount %q", amountText), w)
					return
				}

				_, _, err = svc.Heal(target.ID, amount)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				refresh()
			},
			w,
		)
		healDialog.Resize(fyne.NewSize(420, 200))
		healDialog.Show()
	}

	showEncounterListDialog = func() {
		summaries, err := svc.ListEncounters()
		if err != nil {
			handleErr(err)
			return
		}
		if len(summaries) == 0 {
			dialog.ShowInformation("Encounters", "No saved encounters yet.", w)
			return
		}

		selectedID := summaries[0].ID
		selectedInfo := widget.NewLabel("")
		selectedInfo.TextStyle = fyne.TextStyle{Monospace: true}
		selectedInfo.Wrapping = fyne.TextWrapWord
		selectedIdx := 0

		var list *widget.List
		var launchBtn *widget.Button
		var restartBtn *widget.Button
		var deleteBtn *widget.Button

		renderSelected := func(idx int) {
			if idx < 0 || idx >= len(summaries) {
				selectedID = ""
				selectedInfo.SetText("No encounter selected")
				return
			}
			s := summaries[idx]
			selectedIdx = idx
			selectedID = s.ID
			selectedInfo.SetText(
				fmt.Sprintf(
					"Name: %s\nID: %s\nRound: %d\nCombatants: %d\nUpdated: %s",
					s.Name, s.ID, s.Round, s.Combatants, formatEncounterUpdatedAt(s.UpdatedAt),
				),
			)
		}

		updateActionButtons := func() {
			if selectedID == "" {
				launchBtn.Disable()
				restartBtn.Disable()
				deleteBtn.Disable()
				return
			}
			launchBtn.Enable()
			restartBtn.Enable()
			deleteBtn.Enable()
		}

		list = widget.NewList(
			func() int { return len(summaries) },
			func() fyne.CanvasObject {
				label := widget.NewLabel("encounter")
				label.TextStyle = fyne.TextStyle{Monospace: true}
				return label
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				s := summaries[i]
				o.(*widget.Label).SetText(
					fmt.Sprintf(
						"%s | Round:%d | Combatants:%d | Updated:%s",
						s.Name, s.Round, s.Combatants, formatEncounterUpdatedAt(s.UpdatedAt),
					),
				)
			},
		)
		list.OnSelected = func(id widget.ListItemID) {
			renderSelected(id)
			updateActionButtons()
		}

		dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
		scroll := container.NewScroll(list)
		scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.45))

		refreshSummaries := func(keepID string) error {
			updated, err := svc.ListEncounters()
			if err != nil {
				return err
			}
			summaries = updated
			list.Refresh()

			if len(summaries) == 0 {
				selectedID = ""
				selectedInfo.SetText("No saved encounters left")
				updateActionButtons()
				return nil
			}

			nextIdx := 0
			if keepID != "" {
				for i := range summaries {
					if summaries[i].ID == keepID {
						nextIdx = i
						break
					}
				}
			}
			renderSelected(nextIdx)
			list.Select(nextIdx)
			updateActionButtons()
			return nil
		}

		var encounterDialog *dialog.CustomDialog

		launchBtn = widget.NewButton("Launch", func() {
			if selectedID == "" {
				return
			}
			_, err := svc.ActivateEncounter(selectedID)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			refresh()
			encounterDialog.Hide()
		})
		restartBtn = widget.NewButton("Restart", func() {
			if selectedID == "" {
				return
			}
			_, err := svc.RestartEncounter(selectedID)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			refresh()
			encounterDialog.Hide()
		})
		deleteBtn = widget.NewButton("Delete", func() {
			if selectedID == "" {
				return
			}
			targetID := selectedID
			targetName := summaries[selectedIdx].Name
			dialog.ShowConfirm(
				"Delete Encounter",
				fmt.Sprintf("Soft delete encounter %q?", targetName),
				func(ok bool) {
					if !ok {
						return
					}

					if err := svc.DeleteEncounter(targetID); err != nil {
						dialog.ShowError(err, w)
						return
					}

					if err := refreshSummaries(""); err != nil {
						dialog.ShowError(err, w)
						return
					}
					refresh()
					if len(summaries) == 0 {
						encounterDialog.Hide()
					}
				},
				w,
			)
		})

		renderSelected(0)
		list.Select(0)
		updateActionButtons()

		actions := container.NewGridWithColumns(3, launchBtn, restartBtn, deleteBtn)
		content := container.NewVBox(
			scroll,
			widget.NewSeparator(),
			selectedInfo,
			widget.NewSeparator(),
			actions,
		)

		encounterDialog = dialog.NewCustom("Encounters", "Close", content, w)
		encounterDialog.Resize(dialogSize)
		encounterDialog.Show()
	}

	header := widget.NewLabel("PIP-BOY // FALLOUT 2D20 COMBAT TRACKER")
	header.Alignment = fyne.TextAlignCenter
	header.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	headerBar := container.NewBorder(nil, nil, openEncounterBtn, newEncounterBtn, header)

	content := container.NewBorder(
		container.NewVBox(headerBar, widget.NewSeparator()),
		nil,
		nil,
		nil,
		mainView,
	)

	refresh()

	background := canvas.NewRectangle(color.NRGBA{R: 1, G: 15, B: 6, A: 255})
	glow := canvas.NewRectangle(color.NRGBA{R: 38, G: 125, B: 66, A: 20})
	w.SetContent(container.NewStack(background, content, glow, newScanlineOverlay()))
	w.ShowAndRun()
	return nil
}

func installSignalShutdown(app fyne.App, onShutdown func()) func() {
	signals := make(chan os.Signal, 1)
	done := make(chan struct{})

	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		select {
		case <-signals:
			onShutdown()
			app.Quit()
		case <-done:
		}
	}()

	return func() {
		close(done)
		signal.Stop(signals)
	}
}

func shutdownOnce(fn func()) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			if fn != nil {
				fn()
			}
		})
	}
}

func refreshSelected(label *widget.Label, enc *domain.Encounter, idx int) {
	if enc == nil || len(enc.Combatants) == 0 {
		label.SetText("No combatants")
		return
	}
	if idx < 0 || idx >= len(enc.Combatants) {
		idx = 0
	}
	c := enc.Combatants[idx]
	displayName := encounterDisplayNameByID(enc, c.ID)
	status := "Ready"
	if c.Defeated {
		status = "Defeated"
	}
	label.SetText(
		fmt.Sprintf(
			"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d\nDefense: %d\nDR Physical: %s\nDR Energy: %s\nDR Radiation: %s\nDR Poison: %s\nStatus: %s",
			displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.Defense,
			formatDRValue(c.ResistPhysical, c.ImmunePhysical),
			formatDRValue(c.ResistEnergy, c.ImmuneEnergy),
			formatDRValue(c.ResistRadiation, c.ImmuneRadiation),
			formatDRValue(c.ResistPoison, c.ImmunePoison),
			status,
		),
	)
}

func formatDRValue(value int, immune bool) string {
	if immune {
		return "IMM"
	}
	return strconv.Itoa(value)
}

func formatEncounterUpdatedAt(raw string) string {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t.Local().Format("2006-01-02 15:04:05")
		}
	}
	return raw
}

func formatLogTimestamp(raw string) string {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		t, err := time.Parse(layout, raw)
		if err == nil {
			return t.Local().Format("2006-01-02 15:04:05")
		}
	}
	return raw
}

func encounterDisplayNameByID(enc *domain.Encounter, combatantID string) string {
	if enc == nil {
		return ""
	}
	displayMap := encounterDisplayNameMap(enc)
	if name, ok := displayMap[combatantID]; ok {
		return name
	}
	for i := range enc.Combatants {
		if enc.Combatants[i].ID == combatantID {
			return enc.Combatants[i].Name
		}
	}
	return ""
}

func encounterDisplayNameMap(enc *domain.Encounter) map[string]string {
	names := make(map[string]string, len(enc.Combatants))
	npcCounts := make(map[string]int)
	for i := range enc.Combatants {
		c := enc.Combatants[i]
		if c.Side == domain.SideNPC {
			npcCounts[c.Name]++
		}
	}

	npcSeen := make(map[string]int)
	for i := range enc.Combatants {
		c := enc.Combatants[i]
		if c.Side == domain.SideNPC && npcCounts[c.Name] > 1 {
			npcSeen[c.Name]++
			names[c.ID] = fmt.Sprintf("%s (%s)", c.Name, alphabeticOrdinalLabel(npcSeen[c.Name]-1))
			continue
		}
		names[c.ID] = c.Name
	}
	return names
}

func alphabeticOrdinalLabel(idx int) string {
	if idx < 0 {
		return "A"
	}
	label := ""
	for idx >= 0 {
		label = string(rune('A'+(idx%26))) + label
		idx = idx/26 - 1
	}
	return label
}

func parseDamageType(v string) (domain.DamageType, error) {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case string(domain.DamagePhysical):
		return domain.DamagePhysical, nil
	case string(domain.DamageEnergy):
		return domain.DamageEnergy, nil
	case string(domain.DamageRadiation):
		return domain.DamageRadiation, nil
	case string(domain.DamagePoison):
		return domain.DamagePoison, nil
	default:
		return "", fmt.Errorf("unknown damage type: %q", v)
	}
}

type combatantInputRow struct {
	name         *widget.Entry
	side         *widget.Select
	number       *widget.Entry
	level        *widget.Entry
	xp           *widget.Entry
	initiative   *widget.Entry
	hp           *widget.Entry
	defense      *widget.Entry
	drPhysical   *widget.Entry
	drEnergy     *widget.Entry
	drRadiation  *widget.Entry
	drPoison     *widget.Entry
	immPhysical  *widget.Check
	immEnergy    *widget.Check
	immRadiation *widget.Check
	immPoison    *widget.Check
	root         *fyne.Container
}

func newCombatantInputRow(defaultSide string, onRemove func(*combatantInputRow)) *combatantInputRow {
	name := widget.NewEntry()
	name.SetPlaceHolder("Name")
	name.TextStyle = fyne.TextStyle{Monospace: true}

	side := widget.NewSelect([]string{"party", "npc"}, nil)
	side.SetSelected(defaultSide)
	number := widget.NewEntry()
	number.SetPlaceHolder("Count")
	number.TextStyle = fyne.TextStyle{Monospace: true}
	number.SetText("1")
	level := widget.NewEntry()
	level.SetPlaceHolder("Level")
	level.TextStyle = fyne.TextStyle{Monospace: true}
	level.SetText("1")
	xp := widget.NewEntry()
	xp.SetPlaceHolder("XP")
	xp.TextStyle = fyne.TextStyle{Monospace: true}
	xp.SetText("0")

	initiative := widget.NewEntry()
	initiative.SetPlaceHolder("Init")
	initiative.TextStyle = fyne.TextStyle{Monospace: true}
	hp := widget.NewEntry()
	hp.SetPlaceHolder("HP")
	hp.TextStyle = fyne.TextStyle{Monospace: true}
	hp.SetText("1")
	defense := widget.NewEntry()
	defense.SetPlaceHolder("Defense")
	defense.TextStyle = fyne.TextStyle{Monospace: true}
	defense.SetText("0")
	drPhysical := widget.NewEntry()
	drPhysical.SetPlaceHolder("DR Phys")
	drPhysical.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysical.SetText("0")
	drEnergy := widget.NewEntry()
	drEnergy.SetPlaceHolder("DR Energy")
	drEnergy.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergy.SetText("0")
	drRadiation := widget.NewEntry()
	drRadiation.SetPlaceHolder("DR Rad")
	drRadiation.TextStyle = fyne.TextStyle{Monospace: true}
	drRadiation.SetText("0")
	drPoison := widget.NewEntry()
	drPoison.SetPlaceHolder("DR Poison")
	drPoison.TextStyle = fyne.TextStyle{Monospace: true}
	drPoison.SetText("0")

	drPhysicalCell, immPhysical := newResistanceInputCell(drPhysical)
	drEnergyCell, immEnergy := newResistanceInputCell(drEnergy)
	drRadiationCell, immRadiation := newResistanceInputCell(drRadiation)
	drPoisonCell, immPoison := newResistanceInputCell(drPoison)

	row := &combatantInputRow{
		name:         name,
		side:         side,
		number:       number,
		level:        level,
		xp:           xp,
		initiative:   initiative,
		hp:           hp,
		defense:      defense,
		drPhysical:   drPhysical,
		drEnergy:     drEnergy,
		drRadiation:  drRadiation,
		drPoison:     drPoison,
		immPhysical:  immPhysical,
		immEnergy:    immEnergy,
		immRadiation: immRadiation,
		immPoison:    immPoison,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	side.OnChanged = func(value string) {
		if value == "party" {
			row.number.SetText("1")
			row.number.Disable()
			row.xp.SetText("0")
			row.xp.Disable()
			return
		}
		row.number.Enable()
		row.xp.Enable()
	}
	side.SetSelected(defaultSide)

	row.root = container.NewGridWithColumns(
		13,
		name,
		side,
		number,
		level,
		xp,
		initiative,
		hp,
		defense,
		drPhysicalCell,
		drEnergyCell,
		drRadiationCell,
		drPoisonCell,
		removeBtn,
	)
	return row
}

func newResistanceInputCell(entry *widget.Entry) (fyne.CanvasObject, *widget.Check) {
	immune := widget.NewCheck("immune", func(checked bool) {
		if checked {
			entry.SetText("0")
			entry.Disable()
			return
		}
		entry.Enable()
	})
	return container.NewBorder(nil, nil, nil, immune, entry), immune
}

func newTableHeaderLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	return l
}

func combatantInputRowIsEmpty(row *combatantInputRow) bool {
	if row == nil {
		return true
	}
	return strings.TrimSpace(row.name.Text) == "" &&
		strings.TrimSpace(row.initiative.Text) == "" &&
		strings.TrimSpace(row.level.Text) == "1" &&
		strings.TrimSpace(row.xp.Text) == "0" &&
		strings.TrimSpace(row.hp.Text) == "1" &&
		strings.TrimSpace(row.defense.Text) == "0" &&
		strings.TrimSpace(row.drPhysical.Text) == "0" &&
		strings.TrimSpace(row.drEnergy.Text) == "0" &&
		strings.TrimSpace(row.drRadiation.Text) == "0" &&
		strings.TrimSpace(row.drPoison.Text) == "0" &&
		!row.immPhysical.Checked &&
		!row.immEnergy.Checked &&
		!row.immRadiation.Checked &&
		!row.immPoison.Checked
}

func fillCombatantInputRow(row *combatantInputRow, template domain.Combatant, side domain.Side, count int) {
	if row == nil {
		return
	}
	if count < 1 {
		count = 1
	}

	selectedSide := string(side)
	if selectedSide != "npc" {
		selectedSide = "party"
	}
	row.side.SetSelected(selectedSide)

	row.name.SetText(strings.TrimSpace(template.Name))
	row.number.SetText(strconv.Itoa(count))
	row.level.SetText(strconv.Itoa(template.Level))
	row.xp.SetText(strconv.Itoa(template.XP))
	row.initiative.SetText(strconv.Itoa(template.Initiative))
	row.hp.SetText(strconv.Itoa(template.HP))
	row.defense.SetText(strconv.Itoa(template.Defense))

	row.immPhysical.SetChecked(template.ImmunePhysical)
	row.immEnergy.SetChecked(template.ImmuneEnergy)
	row.immRadiation.SetChecked(template.ImmuneRadiation)
	row.immPoison.SetChecked(template.ImmunePoison)

	if !template.ImmunePhysical {
		row.drPhysical.SetText(strconv.Itoa(template.ResistPhysical))
	}
	if !template.ImmuneEnergy {
		row.drEnergy.SetText(strconv.Itoa(template.ResistEnergy))
	}
	if !template.ImmuneRadiation {
		row.drRadiation.SetText(strconv.Itoa(template.ResistRadiation))
	}
	if !template.ImmunePoison {
		row.drPoison.SetText(strconv.Itoa(template.ResistPoison))
	}
}

func collectCombatantsFromRows(rows []*combatantInputRow) ([]domain.Combatant, error) {
	combatants := make([]domain.Combatant, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.name.Text)
		if name == "" {
			continue
		}

		initiativeText := strings.TrimSpace(row.initiative.Text)
		if initiativeText == "" {
			return nil, fmt.Errorf("combatant %q: initiative is required", name)
		}
		initiative, err := strconv.Atoi(initiativeText)
		if err != nil {
			return nil, fmt.Errorf("combatant %q: invalid initiative %q", name, initiativeText)
		}
		levelText := strings.TrimSpace(row.level.Text)
		if levelText == "" {
			return nil, fmt.Errorf("combatant %q: level is required", name)
		}
		level, err := strconv.Atoi(levelText)
		if err != nil || level < 1 {
			return nil, fmt.Errorf("combatant %q: invalid level %q", name, levelText)
		}
		countText := strings.TrimSpace(row.number.Text)
		if countText == "" {
			countText = "1"
		}
		count, err := strconv.Atoi(countText)
		if err != nil || count < 1 {
			return nil, fmt.Errorf("combatant %q: invalid number %q", name, countText)
		}
		xpText := strings.TrimSpace(row.xp.Text)
		if xpText == "" {
			return nil, fmt.Errorf("combatant %q: XP is required", name)
		}
		xp, err := strconv.Atoi(xpText)
		if err != nil || xp < 0 {
			return nil, fmt.Errorf("combatant %q: invalid XP %q", name, xpText)
		}
		hpText := strings.TrimSpace(row.hp.Text)
		if hpText == "" {
			return nil, fmt.Errorf("combatant %q: HP is required", name)
		}
		hp, err := strconv.Atoi(hpText)
		if err != nil || hp <= 0 {
			return nil, fmt.Errorf("combatant %q: invalid HP %q", name, hpText)
		}
		defenseText := strings.TrimSpace(row.defense.Text)
		if defenseText == "" {
			return nil, fmt.Errorf("combatant %q: defense is required", name)
		}
		defense, err := strconv.Atoi(defenseText)
		if err != nil || defense < 0 {
			return nil, fmt.Errorf("combatant %q: invalid defense %q", name, defenseText)
		}
		drPhysical, immPhysical, err := parseResistanceCell(name, "physical", row.drPhysical.Text, row.immPhysical.Checked)
		if err != nil {
			return nil, err
		}
		drEnergy, immEnergy, err := parseResistanceCell(name, "energy", row.drEnergy.Text, row.immEnergy.Checked)
		if err != nil {
			return nil, err
		}
		drRadiation, immRadiation, err := parseResistanceCell(name, "radiation", row.drRadiation.Text, row.immRadiation.Checked)
		if err != nil {
			return nil, err
		}
		drPoison, immPoison, err := parseResistanceCell(name, "poison", row.drPoison.Text, row.immPoison.Checked)
		if err != nil {
			return nil, err
		}

		side := domain.SideNPC
		if row.side.Selected == "party" {
			side = domain.SideParty
			xp = 0
			count = 1
		}

		for i := 0; i < count; i++ {
			combatants = append(combatants, domain.Combatant{
				ID:              uuid.NewString(),
				Name:            name,
				Side:            side,
				Level:           level,
				XP:              xp,
				Initiative:      initiative,
				HP:              hp,
				Defense:         defense,
				ResistPhysical:  drPhysical,
				ResistEnergy:    drEnergy,
				ResistRadiation: drRadiation,
				ResistPoison:    drPoison,
				ImmunePhysical:  immPhysical,
				ImmuneEnergy:    immEnergy,
				ImmuneRadiation: immRadiation,
				ImmunePoison:    immPoison,
			})
		}
	}

	if len(combatants) == 0 {
		return nil, fmt.Errorf("add at least one combatant")
	}

	return combatants, nil
}

func parseResistanceCell(combatantName, resistType, raw string, immuneChecked bool) (int, bool, error) {
	if immuneChecked {
		return 0, true, nil
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false, fmt.Errorf("combatant %q: %s resistance is required", combatantName, resistType)
	}

	lower := strings.ToLower(value)
	if lower == "imm" || lower == "immune" || lower == "immunity" {
		return 0, true, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, false, fmt.Errorf("combatant %q: invalid %s resistance %q", combatantName, resistType, value)
	}
	return n, false, nil
}

func dynamicEncounterDialogSize(canvasSize fyne.Size) fyne.Size {
	width := canvasSize.Width * 0.94
	height := canvasSize.Height * 0.84

	if width < 860 {
		width = 860
	}
	if height < 480 {
		height = 480
	}
	return fyne.NewSize(width, height)
}

func pipPanel(title string, body fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := widget.NewLabel("> " + title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	header := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
	)
	content := container.NewBorder(header, nil, nil, nil, body)

	panelBG := canvas.NewRectangle(color.NRGBA{R: 5, G: 26, B: 11, A: 236})
	return container.NewStack(panelBG, container.NewPadded(content))
}

func newScanlineOverlay() fyne.CanvasObject {
	scan := canvas.NewRasterWithPixels(func(x, y, w, h int) color.Color {
		if y%3 == 0 {
			return color.NRGBA{R: 150, G: 255, B: 180, A: 16}
		}
		if y%7 == 0 && x%2 == 0 {
			return color.NRGBA{R: 0, G: 0, B: 0, A: 12}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	})
	return scan
}
