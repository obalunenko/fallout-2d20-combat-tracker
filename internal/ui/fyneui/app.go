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
		enc            *domain.Encounter
		activeCampaign *domain.Campaign
		selectedIndex  int
	)
	var showApplyDamageDialogForIndex func(int)
	var showHealDialogForIndex func(int)

	roundLabel := widget.NewLabel("")
	selectedLabel := widget.NewLabel("")
	partyAPLabel := widget.NewLabel("")
	threatLabel := widget.NewLabel("")

	roundLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	partyAPLabel.TextStyle = fyne.TextStyle{Monospace: true}
	threatLabel.TextStyle = fyne.TextStyle{Monospace: true}

	logOutput := widget.NewMultiLineEntry()
	logOutput.TextStyle = fyne.TextStyle{Monospace: true}
	logOutput.Wrapping = fyne.TextWrapWord
	logOutput.SetMinRowsVisible(18)
	logOutput.Disable()
	logOutput.SetText("[BOOT] Pip-Boy combat tracker initialized")
	setEventLog := func(lines []string) {
		logOutput.SetText(strings.Join(lines, "\n"))
	}

	combatantLine := func(c domain.Combatant) string {
		name := c.Name
		if enc != nil {
			name = encounterDisplayNameByID(enc, c.ID)
		}
		prefix := "   "
		isDefeated := c.Defeated || c.HP <= 0
		if c.Active && !isDefeated {
			prefix = ">> "
		} else if isDefeated {
			prefix = "xx "
		}
		line := fmt.Sprintf(
			"%s%s [%s] Lvl:%d XP:%d Init:%d HP:%d/%d DEF:%d DR P/E/R/P:%s/%s/%s/%s",
			prefix, name, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP, c.Defense,
			formatDRValue(c.ResistPhysical, c.ImmunePhysical),
			formatDRValue(c.ResistEnergy, c.ImmuneEnergy),
			formatDRValue(c.ResistRadiation, c.ImmuneRadiation),
			formatDRValue(c.ResistPoison, c.ImmunePoison),
		)
		if isDefeated {
			return line + " [DEFEATED]"
		}
		return line
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
			label := o.(*widget.Label)
			c := enc.Combatants[i]
			label.SetText(combatantLine(c))
			if c.Defeated || c.HP <= 0 {
				label.Importance = widget.LowImportance
				label.TextStyle = fyne.TextStyle{Monospace: true, Italic: true}
				return
			}
			label.Importance = widget.MediumImportance
			label.TextStyle = fyne.TextStyle{Monospace: true}
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

		for i, c := range enc.Combatants {
			idx := i
			line := widget.NewLabel(combatantLine(c))
			isDefeated := c.Defeated || c.HP <= 0
			line.TextStyle = fyne.TextStyle{Monospace: true, Bold: c.Active && !isDefeated, Italic: isDefeated}
			if isDefeated {
				line.Importance = widget.LowImportance
			}

			damageBtn := widget.NewButton("DMG", func() {
				showApplyDamageDialogForIndex(idx)
			})
			healBtn := widget.NewButton("HEAL", func() {
				showHealDialogForIndex(idx)
			})
			damageBtn.Importance = widget.LowImportance
			healBtn.Importance = widget.LowImportance

			row := container.NewBorder(nil, nil, nil, container.NewGridWithColumns(2, damageBtn, healBtn), line)
			encounterOrderBox.Add(row)
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
	var showCreateCampaignDialog func()
	var showCampaignListDialog func()
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

	turnPanel := pipPanel(
		"TURN CONTROL",
		container.NewVBox(
			roundLabel,
			nextTurnBtn,
		),
	)
	resourcesPanel := pipPanel(
		"RESOURCES",
		container.NewGridWithColumns(
			6,
			partyAPLabel,
			partyAddBtn,
			partySpendBtn,
			threatLabel,
			threatAddBtn,
			threatSpendBtn,
		),
	)
	encounterOrderPanel := pipPanel("ENCOUNTER ORDER", encounterOrderBox)
	combatantsPanel := pipPanel("INITIATIVE ORDER", list)
	selectedPanel := pipPanel(
		"SELECTED COMBATANT",
		container.NewVBox(selectedLabel, widget.NewSeparator(), container.NewGridWithColumns(2, applyDamageBtn, healBtn)),
	)
	logPanel := pipPanel("DATA LOG", logOutput)

	statTabContent := container.NewVBox(
		turnPanel,
		widget.NewSeparator(),
		resourcesPanel,
		widget.NewSeparator(),
		encounterOrderPanel,
	)
	invTabContent := container.NewGridWithColumns(2, combatantsPanel, selectedPanel)
	dataTabContent := container.NewBorder(nil, nil, nil, nil, logPanel)

	tabs := container.NewAppTabs(
		container.NewTabItem("STAT", container.NewPadded(statTabContent)),
		container.NewTabItem("INV", container.NewPadded(invTabContent)),
		container.NewTabItem("DATA", container.NewPadded(dataTabContent)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabsView := container.New(layout.NewPaddedLayout(), tabs)

	campaignStatusLabel := widget.NewLabel("Campaign: -")
	campaignStatusLabel.TextStyle = fyne.TextStyle{Monospace: true}
	newEncounterBtn := widget.NewButton("NEW ENCOUNTER", func() {
		showCreateEncounterDialog()
	})
	openEncounterBtn := widget.NewButton("OPEN ENCOUNTER", func() {
		showEncounterListDialog()
	})
	newCampaignBtn := widget.NewButton("NEW CAMPAIGN", func() {
		showCreateCampaignDialog()
	})
	openCampaignBtn := widget.NewButton("OPEN CAMPAIGN", func() {
		showCampaignListDialog()
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

	campaignHint := widget.NewLabel("No active campaign.\nCreate or choose a campaign before running encounters.")
	campaignHint.Alignment = fyne.TextAlignCenter
	campaignHint.TextStyle = fyne.TextStyle{Monospace: true}
	campaignActions := container.NewGridWithColumns(2,
		widget.NewButton("CREATE CAMPAIGN", func() { showCreateCampaignDialog() }),
		widget.NewButton("OPEN CAMPAIGNS", func() { showCampaignListDialog() }),
	)
	noCampaignView := pipPanel(
		"CAMPAIGN CONTROL",
		container.NewCenter(container.NewVBox(campaignHint, widget.NewSeparator(), campaignActions)),
	)

	mainView := container.NewStack(tabsView, noEncounterView, noCampaignView)

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
		activeCampaign, err = svc.GetActiveCampaign()
		if err != nil {
			if errors.Is(err, domain.ErrCampaignNotInitialized) {
				activeCampaign = nil
				enc = nil
				campaignStatusLabel.SetText("Campaign: -")
				roundLabel.SetText("Round: -")
				refreshSelected(selectedLabel, nil, 0)
				refreshResources()
				list.Refresh()
				rebuildEncounterOrder()
				refreshDataLog()
				tabsView.Hide()
				noEncounterView.Hide()
				noCampaignView.Show()
				mainView.Refresh()
				return
			}
			handleErr(err)
			return
		}
		campaignStatusLabel.SetText(fmt.Sprintf("Campaign: %s (%s)", activeCampaign.Name, activeCampaign.StartDate))
		setupHint.SetText(fmt.Sprintf("Campaign: %s\nNo active encounter.\nCreate one from scratch to begin tracking combat.", activeCampaign.Name))

		enc, err = svc.GetEncounter()
		if err != nil {
			if errors.Is(err, domain.ErrEncounterNotInitialized) {
				enc = nil
				roundLabel.SetText("Round: -")
				refreshSelected(selectedLabel, nil, 0)
				refreshResources()
				list.Refresh()
				rebuildEncounterOrder()
				refreshDataLog()
				tabsView.Hide()
				noCampaignView.Hide()
				noEncounterView.Show()
				mainView.Refresh()
				return
			}
			handleErr(err)
			return
		}

		roundLabel.SetText(fmt.Sprintf("Round: %d", enc.Round))

		if selectedIndex >= len(enc.Combatants) {
			selectedIndex = 0
		}
		refreshSelected(selectedLabel, enc, selectedIndex)
		refreshResources()
		list.Refresh()
		rebuildEncounterOrder()
		refreshDataLog()
		noCampaignView.Hide()
		noEncounterView.Hide()
		tabsView.Show()
		mainView.Refresh()
	}

	showCampaignEditorDialog := func(
		title string,
		submitLabel string,
		initialName string,
		initialStartDate string,
		initialPlayers []domain.NewCampaignPlayer,
		onSubmit func(name, startDate string, players []domain.NewCampaignPlayer) error,
	) {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("e.g. Commonwealth Survival")
		nameEntry.SetText(strings.TrimSpace(initialName))
		startDateEntry := widget.NewEntry()
		startDateEntry.SetPlaceHolder("YYYY-MM-DD")
		if strings.TrimSpace(initialStartDate) == "" {
			startDateEntry.SetText(time.Now().Format("2006-01-02"))
		} else {
			startDateEntry.SetText(strings.TrimSpace(initialStartDate))
		}

		var rows []*campaignPlayerInputRow
		rowsBox := container.NewVBox()
		headers := container.NewGridWithColumns(
			13,
			newTableHeaderLabel("Player"),
			newTableHeaderLabel("Character"),
			newTableHeaderLabel("Level"),
			newTableHeaderLabel("Init"),
			newTableHeaderLabel("HP Cur"),
			newTableHeaderLabel("HP Max"),
			newTableHeaderLabel("Defense"),
			newTableHeaderLabel("DR Phys"),
			newTableHeaderLabel("DR Energy"),
			newTableHeaderLabel("DR Rad"),
			newTableHeaderLabel("DR Poison"),
			newTableHeaderLabel("Active"),
			newTableHeaderLabel("Action"),
		)
		table := container.NewVBox(headers, widget.NewSeparator(), rowsBox)
		addRow := func() *campaignPlayerInputRow {
			row := newCampaignPlayerInputRow(func(target *campaignPlayerInputRow) {
				if len(rows) == 1 {
					target.playerName.SetText("")
					target.characterName.SetText("")
					target.level.SetText("1")
					target.initiative.SetText("1")
					target.hp.SetText("1")
					target.hpMax.SetText("1")
					target.defense.SetText("0")
					target.drPhysical.SetText("0")
					target.drEnergy.SetText("0")
					target.drRadiation.SetText("0")
					target.drPoison.SetText("0")
					target.immPhysical.SetChecked(false)
					target.immEnergy.SetChecked(false)
					target.immRadiation.SetChecked(false)
					target.immPoison.SetChecked(false)
					return
				}
				filtered := make([]*campaignPlayerInputRow, 0, len(rows)-1)
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
		if len(initialPlayers) == 0 {
			addRow()
		} else {
			for _, p := range initialPlayers {
				row := addRow()
				row.playerName.SetText(p.PlayerName)
				row.characterName.SetText(p.Character.Name)
				row.level.SetText(strconv.Itoa(p.Character.Level))
				row.initiative.SetText(strconv.Itoa(p.Character.Initiative))
				row.hp.SetText(strconv.Itoa(p.Character.HP))
				maxHP := p.Character.MaxHP
				if maxHP <= 0 {
					maxHP = p.Character.HP
				}
				if maxHP <= 0 {
					maxHP = 1
				}
				row.hpMax.SetText(strconv.Itoa(maxHP))
				row.defense.SetText(strconv.Itoa(p.Character.Defense))
				row.immPhysical.SetChecked(p.Character.ImmunePhysical)
				row.immEnergy.SetChecked(p.Character.ImmuneEnergy)
				row.immRadiation.SetChecked(p.Character.ImmuneRadiation)
				row.immPoison.SetChecked(p.Character.ImmunePoison)
				if !p.Character.ImmunePhysical {
					row.drPhysical.SetText(strconv.Itoa(p.Character.ResistPhysical))
				}
				if !p.Character.ImmuneEnergy {
					row.drEnergy.SetText(strconv.Itoa(p.Character.ResistEnergy))
				}
				if !p.Character.ImmuneRadiation {
					row.drRadiation.SetText(strconv.Itoa(p.Character.ResistRadiation))
				}
				if !p.Character.ImmunePoison {
					row.drPoison.SetText(strconv.Itoa(p.Character.ResistPoison))
				}
			}
		}

		validationError := widget.NewLabel("")
		validationError.TextStyle = fyne.TextStyle{Monospace: true}
		validationError.Wrapping = fyne.TextWrapWord
		addPlayerBtn := widget.NewButton("+ Add Player", func() { addRow() })
		scroll := container.NewScroll(table)
		dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
		scroll.Direction = container.ScrollBoth
		scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.5))
		playerSection := container.NewVBox(addPlayerBtn, scroll)

		form := widget.NewForm(
			widget.NewFormItem("Campaign Name", nameEntry),
			widget.NewFormItem("Start Date", startDateEntry),
			widget.NewFormItem("Players", playerSection),
		)
		formContent := container.NewVBox(form, widget.NewSeparator(), validationError)

		var editorDialog *dialog.CustomDialog
		cancelBtn := widget.NewButton("Cancel", func() { editorDialog.Hide() })
		submitBtn := widget.NewButton(submitLabel, func() {
			validationError.SetText("")
			campaignName := strings.TrimSpace(nameEntry.Text)
			startDate := strings.TrimSpace(startDateEntry.Text)
			if campaignName == "" {
				validationError.SetText("Campaign name is required")
				return
			}
			if _, err := time.Parse("2006-01-02", startDate); err != nil {
				validationError.SetText("Start date must be in YYYY-MM-DD format")
				return
			}

			players, err := collectCampaignPlayersFromRows(rows)
			if err != nil {
				validationError.SetText(err.Error())
				return
			}
			if err := onSubmit(campaignName, startDate, players); err != nil {
				validationError.SetText(err.Error())
				return
			}
			refresh()
			editorDialog.Hide()
		})

		editorDialog = dialog.NewCustomWithoutButtons(title, formContent, w)
		editorDialog.SetButtons([]fyne.CanvasObject{cancelBtn, submitBtn})
		editorDialog.Resize(dialogSize)
		editorDialog.Show()
	}

	showCreateCampaignDialog = func() {
		showCampaignEditorDialog(
			"Create Campaign",
			"Create",
			"",
			time.Now().Format("2006-01-02"),
			nil,
			func(name, startDate string, players []domain.NewCampaignPlayer) error {
				_, err := svc.CreateCampaign(uuid.NewString(), name, startDate, players)
				return err
			},
		)
	}

	showCampaignListDialog = func() {
		campaigns, err := svc.ListCampaigns()
		if err != nil {
			handleErr(err)
			return
		}
		if len(campaigns) == 0 {
			dialog.ShowInformation("Campaigns", "No campaigns yet. Create one first.", w)
			return
		}

		selectedID := campaigns[0].ID
		selectedInfo := widget.NewLabel("")
		selectedInfo.TextStyle = fyne.TextStyle{Monospace: true}
		selectedInfo.Wrapping = fyne.TextWrapWord
		selectedIdx := 0

		renderSelected := func(idx int) {
			if idx < 0 || idx >= len(campaigns) {
				selectedID = ""
				selectedInfo.SetText("No campaign selected")
				return
			}
			selectedIdx = idx
			selectedID = campaigns[idx].ID
			current := ""
			if activeCampaign != nil && activeCampaign.ID == campaigns[idx].ID {
				current = " (active)"
			}
			selectedInfo.SetText(fmt.Sprintf(
				"Name: %s%s\nID: %s\nStart Date: %s\nUpdated: %s",
				campaigns[idx].Name, current, campaigns[idx].ID, campaigns[idx].StartDate, formatEncounterUpdatedAt(campaigns[idx].UpdatedAt),
			))
		}

		list := widget.NewList(
			func() int { return len(campaigns) },
			func() fyne.CanvasObject {
				label := widget.NewLabel("campaign")
				label.TextStyle = fyne.TextStyle{Monospace: true}
				return label
			},
			func(i widget.ListItemID, o fyne.CanvasObject) {
				c := campaigns[i]
				activeMark := ""
				if activeCampaign != nil && activeCampaign.ID == c.ID {
					activeMark = " [active]"
				}
				o.(*widget.Label).SetText(fmt.Sprintf("%s%s | Start:%s | Updated:%s", c.Name, activeMark, c.StartDate, formatEncounterUpdatedAt(c.UpdatedAt)))
			},
		)
		list.OnSelected = func(id widget.ListItemID) { renderSelected(id) }

		dialogSize := dynamicEncounterDialogSize(w.Canvas().Size())
		scroll := container.NewScroll(list)
		scroll.SetMinSize(fyne.NewSize(dialogSize.Width-80, dialogSize.Height*0.45))

		var campaignDialog *dialog.CustomDialog
		activateBtn := widget.NewButton("Activate", func() {
			if selectedID == "" {
				return
			}
			if _, err := svc.ActivateCampaign(selectedID); err != nil {
				dialog.ShowError(err, w)
				return
			}
			refresh()
			campaignDialog.Hide()
		})
		createBtn := widget.NewButton("Create New", func() {
			campaignDialog.Hide()
			showCreateCampaignDialog()
		})
		editBtn := widget.NewButton("Edit", func() {
			if selectedID == "" || selectedIdx < 0 || selectedIdx >= len(campaigns) {
				return
			}
			players, err := svc.ListCampaignPlayers(selectedID)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			current := campaigns[selectedIdx]
			campaignDialog.Hide()
			showCampaignEditorDialog(
				"Edit Campaign",
				"Save",
				current.Name,
				current.StartDate,
				players,
				func(name, startDate string, editedPlayers []domain.NewCampaignPlayer) error {
					_, updateErr := svc.UpdateCampaign(current.ID, name, startDate, editedPlayers)
					return updateErr
				},
			)
		})
		infoBtn := widget.NewButton("Use Selected", func() {
			if selectedIdx >= 0 && selectedIdx < len(campaigns) {
				if _, err := svc.ActivateCampaign(campaigns[selectedIdx].ID); err != nil {
					dialog.ShowError(err, w)
					return
				}
				refresh()
				campaignDialog.Hide()
			}
		})

		renderSelected(0)
		list.Select(0)

		content := container.NewVBox(
			scroll,
			widget.NewSeparator(),
			selectedInfo,
			widget.NewSeparator(),
			container.NewGridWithColumns(4, activateBtn, infoBtn, editBtn, createBtn),
		)

		campaignDialog = dialog.NewCustom("Campaigns", "Close", content, w)
		campaignDialog.Resize(dialogSize)
		campaignDialog.Show()
	}

	showEncounterEditorDialog := func(
		title string,
		submitLabel string,
		initialName string,
		initialCombatants []domain.Combatant,
		onSubmit func(name string, combatants []domain.Combatant) error,
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
			14,
			newTableHeaderLabel("Name"),
			newTableHeaderLabel("Side"),
			newTableHeaderLabel("Number"),
			newTableHeaderLabel("Level"),
			newTableHeaderLabel("XP"),
			newTableHeaderLabel("Initiative"),
			newTableHeaderLabel("HP Cur"),
			newTableHeaderLabel("HP Max"),
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
					target.hpMax.SetText("1")
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

			refresh()
			editorDialog.Hide()
		})

		editorDialog = dialog.NewCustomWithoutButtons(title, formContent, w)
		editorDialog.SetButtons([]fyne.CanvasObject{cancelBtn, submitBtn})
		editorDialog.Resize(dialogSize)
		editorDialog.Show()
	}

	showCreateEncounterDialog = func() {
		showEncounterEditorDialog(
			"Create Encounter",
			"Create",
			"",
			nil,
			func(name string, combatants []domain.Combatant) error {
				encounterID := uuid.NewString()
				_, err := svc.CreateEncounter(encounterID, name, combatants)
				return err
			},
		)
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
		var editBtn *widget.Button

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
					"Name: %s\nID: %s\nRound: %d\nCombatants: %d\nDifficulty: %s\nUpdated: %s",
					s.Name, s.ID, s.Round, s.Combatants, formatEncounterDifficultySummary(s), formatEncounterUpdatedAt(s.UpdatedAt),
				),
			)
		}

		updateActionButtons := func() {
			if selectedID == "" {
				launchBtn.Disable()
				restartBtn.Disable()
				deleteBtn.Disable()
				editBtn.Disable()
				return
			}
			launchBtn.Enable()
			restartBtn.Enable()
			deleteBtn.Enable()
			editBtn.Enable()
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
						"%s | %s | Round:%d | Combatants:%d | Updated:%s",
						s.Name, formatEncounterDifficultySummary(s), s.Round, s.Combatants, formatEncounterUpdatedAt(s.UpdatedAt),
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
		editBtn = widget.NewButton("Edit", func() {
			if selectedID == "" {
				return
			}
			targetID := selectedID
			targetName := summaries[selectedIdx].Name
			encForEdit, err := svc.GetEncounterByID(targetID)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			encounterDialog.Hide()
			showEncounterEditorDialog(
				fmt.Sprintf("Edit Encounter: %s", targetName),
				"Save",
				encForEdit.Name,
				encForEdit.Combatants,
				func(name string, combatants []domain.Combatant) error {
					_, updateErr := svc.UpdateEncounter(targetID, name, combatants)
					return updateErr
				},
			)
		})

		renderSelected(0)
		list.Select(0)
		updateActionButtons()

		actions := container.NewGridWithColumns(4, launchBtn, restartBtn, editBtn, deleteBtn)
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
	leftControls := container.NewHBox(openCampaignBtn, openEncounterBtn)
	rightControls := container.NewHBox(newCampaignBtn, newEncounterBtn)
	headerBar := container.NewBorder(nil, nil, leftControls, rightControls, header)
	topBar := container.NewVBox(headerBar, campaignStatusLabel)

	content := container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
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
			fyne.Do(app.Quit)
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
			"Name: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nDR Physical: %s\nDR Energy: %s\nDR Radiation: %s\nDR Poison: %s\nStatus: %s",
			displayName, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP, c.Defense,
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

func formatEncounterDifficultySummary(s domain.EncounterSummary) string {
	if strings.TrimSpace(s.Difficulty) == "" {
		return "Unknown"
	}
	if s.PartyCount == 0 || s.EnemyCount == 0 {
		return s.Difficulty
	}
	return fmt.Sprintf(
		"%s (P:%d avgLvl:%.1f budget:%d | NPC:%d avgLvl:%.1f XP:%d)",
		s.Difficulty,
		s.PartyCount,
		s.PartyAvgLevel,
		s.PartyXPBudget,
		s.EnemyCount,
		s.EnemyAvgLevel,
		s.EnemyTotalXP,
	)
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
	hpMax        *widget.Entry
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

type campaignPlayerInputRow struct {
	playerName    *widget.Entry
	characterName *widget.Entry
	level         *widget.Entry
	initiative    *widget.Entry
	hp            *widget.Entry
	hpMax         *widget.Entry
	defense       *widget.Entry
	drPhysical    *widget.Entry
	drEnergy      *widget.Entry
	drRadiation   *widget.Entry
	drPoison      *widget.Entry
	immPhysical   *widget.Check
	immEnergy     *widget.Check
	immRadiation  *widget.Check
	immPoison     *widget.Check
	root          *fyne.Container
}

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
	drPhysical := widget.NewEntry()
	drPhysical.SetPlaceHolder("DR Phys")
	drPhysical.TextStyle = fyne.TextStyle{Monospace: true}
	drPhysical.SetText("0")
	drPhysical.OnChanged = func(string) { notifyChange() }
	drEnergy := widget.NewEntry()
	drEnergy.SetPlaceHolder("DR Energy")
	drEnergy.TextStyle = fyne.TextStyle{Monospace: true}
	drEnergy.SetText("0")
	drEnergy.OnChanged = func(string) { notifyChange() }
	drRadiation := widget.NewEntry()
	drRadiation.SetPlaceHolder("DR Rad")
	drRadiation.TextStyle = fyne.TextStyle{Monospace: true}
	drRadiation.SetText("0")
	drRadiation.OnChanged = func(string) { notifyChange() }
	drPoison := widget.NewEntry()
	drPoison.SetPlaceHolder("DR Poison")
	drPoison.TextStyle = fyne.TextStyle{Monospace: true}
	drPoison.SetText("0")
	drPoison.OnChanged = func(string) { notifyChange() }

	drPhysicalCell, immPhysical := newResistanceInputCell(drPhysical, notifyChange)
	drEnergyCell, immEnergy := newResistanceInputCell(drEnergy, notifyChange)
	drRadiationCell, immRadiation := newResistanceInputCell(drRadiation, notifyChange)
	drPoisonCell, immPoison := newResistanceInputCell(drPoison, notifyChange)

	row := &combatantInputRow{
		name:         name,
		side:         side,
		number:       number,
		level:        level,
		xp:           xp,
		initiative:   initiative,
		hp:           hp,
		hpMax:        hpMax,
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
			notifyChange()
			return
		}
		row.number.Enable()
		row.xp.Enable()
		notifyChange()
	}
	side.SetSelected(defaultSide)

	row.root = container.NewGridWithColumns(
		14,
		name,
		side,
		number,
		level,
		xp,
		initiative,
		hp,
		hpMax,
		defense,
		drPhysicalCell,
		drEnergyCell,
		drRadiationCell,
		drPoisonCell,
		removeBtn,
	)
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

	drPhysicalCell, immPhysical := newResistanceInputCell(drPhysical, nil)
	drEnergyCell, immEnergy := newResistanceInputCell(drEnergy, nil)
	drRadiationCell, immRadiation := newResistanceInputCell(drRadiation, nil)
	drPoisonCell, immPoison := newResistanceInputCell(drPoison, nil)

	row := &campaignPlayerInputRow{
		playerName:    playerName,
		characterName: characterName,
		level:         level,
		initiative:    initiative,
		hp:            hp,
		hpMax:         hpMax,
		defense:       defense,
		drPhysical:    drPhysical,
		drEnergy:      drEnergy,
		drRadiation:   drRadiation,
		drPoison:      drPoison,
		immPhysical:   immPhysical,
		immEnergy:     immEnergy,
		immRadiation:  immRadiation,
		immPoison:     immPoison,
	}
	removeBtn := widget.NewButton("Remove", func() { onRemove(row) })
	activeLabel := widget.NewLabel("yes")
	activeLabel.TextStyle = fyne.TextStyle{Monospace: true}
	row.root = container.NewGridWithColumns(
		13,
		playerName,
		characterName,
		level,
		initiative,
		hp,
		hpMax,
		defense,
		drPhysicalCell,
		drEnergyCell,
		drRadiationCell,
		drPoisonCell,
		activeLabel,
		removeBtn,
	)
	return row
}

func newResistanceInputCell(entry *widget.Entry, onChanged func()) (fyne.CanvasObject, *widget.Check) {
	immune := widget.NewCheck("immune", func(checked bool) {
		if checked {
			entry.SetText("0")
			entry.Disable()
			if onChanged != nil {
				onChanged()
			}
			return
		}
		entry.Enable()
		if onChanged != nil {
			onChanged()
		}
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
		strings.TrimSpace(row.hpMax.Text) == "1" &&
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
	maxHP := template.MaxHP
	if maxHP <= 0 {
		maxHP = template.HP
	}
	if maxHP <= 0 {
		maxHP = 1
	}
	row.hpMax.SetText(strconv.Itoa(maxHP))
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
		if err != nil || hp < 0 {
			return nil, fmt.Errorf("combatant %q: invalid HP %q", name, hpText)
		}
		hpMaxText := strings.TrimSpace(row.hpMax.Text)
		if hpMaxText == "" {
			return nil, fmt.Errorf("combatant %q: max HP is required", name)
		}
		hpMax, err := strconv.Atoi(hpMaxText)
		if err != nil || hpMax < 1 {
			return nil, fmt.Errorf("combatant %q: invalid max HP %q", name, hpMaxText)
		}
		if hp > hpMax {
			return nil, fmt.Errorf("combatant %q: current HP cannot exceed max HP", name)
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
				MaxHP:           hpMax,
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

func collectCombatantsPreviewFromRows(rows []*combatantInputRow) []domain.Combatant {
	preview := make([]domain.Combatant, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.name.Text)
		if name == "" {
			continue
		}

		side := domain.SideNPC
		if row.side.Selected == "party" {
			side = domain.SideParty
		}

		level := 1
		if parsed, err := strconv.Atoi(strings.TrimSpace(row.level.Text)); err == nil && parsed > 0 {
			level = parsed
		}

		count := 1
		if side == domain.SideNPC {
			if parsed, err := strconv.Atoi(strings.TrimSpace(row.number.Text)); err == nil && parsed > 0 {
				count = parsed
			}
		}

		xp := 0
		if side == domain.SideNPC {
			if parsed, err := strconv.Atoi(strings.TrimSpace(row.xp.Text)); err == nil && parsed >= 0 {
				xp = parsed
			}
		}

		for i := 0; i < count; i++ {
			preview = append(preview, domain.Combatant{
				Name:  name,
				Side:  side,
				Level: level,
				XP:    xp,
			})
		}
	}
	return preview
}

func formatDifficultyPreview(metrics domain.EncounterDifficultyMetrics) string {
	if metrics.PartyCount == 0 || metrics.EnemyCount == 0 {
		return "Difficulty: Unknown (add at least one party member and one NPC)"
	}
	return fmt.Sprintf(
		"Difficulty: %s (xp ratio: %.2f | party: %d avg lvl %.1f budget %d | npc: %d avg lvl %.1f total xp: %d)",
		metrics.Label,
		metrics.Score,
		metrics.PartyCount,
		metrics.PartyAvgLevel,
		metrics.PartyXPBudget,
		metrics.EnemyCount,
		metrics.EnemyAvgLevel,
		metrics.EnemyTotalXP,
	)
}

func collectCampaignPlayersFromRows(rows []*campaignPlayerInputRow) ([]domain.NewCampaignPlayer, error) {
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, row := range rows {
		playerName := strings.TrimSpace(row.playerName.Text)
		characterName := strings.TrimSpace(row.characterName.Text)
		if playerName == "" && characterName == "" {
			continue
		}
		if playerName == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if characterName == "" {
			return nil, fmt.Errorf("character name is required for player %q", playerName)
		}

		level, err := parsePositiveIntOrError(strings.TrimSpace(row.level.Text), "level", playerName)
		if err != nil {
			return nil, err
		}
		initiative, err := parseNonNegativeIntOrError(strings.TrimSpace(row.initiative.Text), "initiative", playerName)
		if err != nil {
			return nil, err
		}
		hp, err := parseNonNegativeIntOrError(strings.TrimSpace(row.hp.Text), "HP", playerName)
		if err != nil {
			return nil, err
		}
		hpMax, err := parsePositiveIntOrError(strings.TrimSpace(row.hpMax.Text), "max HP", playerName)
		if err != nil {
			return nil, err
		}
		if hp > hpMax {
			return nil, fmt.Errorf("current HP cannot exceed max HP for %q", playerName)
		}
		defense, err := parseNonNegativeIntOrError(strings.TrimSpace(row.defense.Text), "defense", playerName)
		if err != nil {
			return nil, err
		}
		drPhysical, immPhysical, err := parseResistanceCell(characterName, "physical", row.drPhysical.Text, row.immPhysical.Checked)
		if err != nil {
			return nil, err
		}
		drEnergy, immEnergy, err := parseResistanceCell(characterName, "energy", row.drEnergy.Text, row.immEnergy.Checked)
		if err != nil {
			return nil, err
		}
		drRadiation, immRadiation, err := parseResistanceCell(characterName, "radiation", row.drRadiation.Text, row.immRadiation.Checked)
		if err != nil {
			return nil, err
		}
		drPoison, immPoison, err := parseResistanceCell(characterName, "poison", row.drPoison.Text, row.immPoison.Checked)
		if err != nil {
			return nil, err
		}

		players = append(players, domain.NewCampaignPlayer{
			PlayerName: playerName,
			Character: domain.Combatant{
				ID:              uuid.NewString(),
				Name:            characterName,
				Side:            domain.SideParty,
				Level:           level,
				Initiative:      initiative,
				HP:              hp,
				MaxHP:           hpMax,
				Defense:         defense,
				ResistPhysical:  drPhysical,
				ResistEnergy:    drEnergy,
				ResistRadiation: drRadiation,
				ResistPoison:    drPoison,
				ImmunePhysical:  immPhysical,
				ImmuneEnergy:    immEnergy,
				ImmuneRadiation: immRadiation,
				ImmunePoison:    immPoison,
			},
		})
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	return players, nil
}

func parsePositiveIntOrError(raw, fieldName, label string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s for %q is required", fieldName, label)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, fmt.Errorf("invalid %s %q for %q", fieldName, raw, label)
	}
	return value, nil
}

func parseNonNegativeIntOrError(raw, fieldName, label string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("%s for %q is required", fieldName, label)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("invalid %s %q for %q", fieldName, raw, label)
	}
	return value, nil
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
