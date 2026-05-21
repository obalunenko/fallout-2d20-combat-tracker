package fyneui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func Run(ctx context.Context, svc *appsvc.Service, onShutdown func()) error {
	if ctx == nil {
		ctx = context.Background()
	}
	a := fyneapp.New()
	a.Settings().SetTheme(newPipBoyTheme())
	shutdown := shutdownOnce(onShutdown)
	a.Lifecycle().SetOnStopped(shutdown)
	stopSignals := installSignalShutdown(a, shutdown)
	defer stopSignals()

	w := a.NewWindow("Fallout 2d20 Combat Tracker")
	w.Resize(fyne.NewSize(1100, 700))

	var (
		enc                 *domain.Encounter
		activeCampaign      *domain.Campaign
		selectedIndex       int
		expandedCombatantID string
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
			"%s%s [%s] Lvl:%d XP:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
			prefix, name, c.Side, c.Level, c.XP, c.Initiative, c.HP, c.MaxHP, c.Defense, formatDRValue(c.ResistPoison, c.ImmunePoison),
		)
		if isDefeated {
			return line + " [DEFEATED]"
		}
		return line
	}

	expandedCombatantDetails := func(c domain.Combatant) string {
		status := "Ready"
		if c.Defeated || c.HP <= 0 {
			status = "Defeated"
		} else if c.Active {
			status = "Active"
		}
		if isTorsoOnlyCombatant(c) {
			return fmt.Sprintf(
				"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\nDR Physical: %s\nDR Energy: %s\nDR Radiation: %s\nDR Poison: %s",
				encounterDisplayNameByID(enc, c.ID),
				c.Side,
				c.Level,
				c.XP,
				c.Initiative,
				c.HP,
				c.MaxHP,
				c.Defense,
				status,
				formatDRValue(c.ResistPhysicalTorso, c.ImmunePhysical),
				formatDRValue(c.ResistEnergyTorso, c.ImmuneEnergy),
				formatDRValue(c.ResistRadiationTorso, c.ImmuneRadiation),
				formatDRValue(c.ResistPoison, c.ImmunePoison),
			)
		}
		return fmt.Sprintf(
			"Participant Details\nName: %s\nSide: %s\nLevel: %d\nXP: %d\nInitiative: %d\nHP: %d/%d\nDefense: %d\nStatus: %s\nDR Poison: %s\n\nBody Defense / Damage Resistance\nLocation  | Defense | Physical | Energy | Radiation\n-----------------------------------------------------\nHead      | %7d | %8d | %6d | %9d\nTorso     | %7d | %8d | %6d | %9d\nLeft Arm  | %7d | %8d | %6d | %9d\nRight Arm | %7d | %8d | %6d | %9d\nLeft Leg  | %7d | %8d | %6d | %9d\nRight Leg | %7d | %8d | %6d | %9d",
			encounterDisplayNameByID(enc, c.ID),
			c.Side,
			c.Level,
			c.XP,
			c.Initiative,
			c.HP,
			c.MaxHP,
			c.Defense,
			status,
			formatDRValue(c.ResistPoison, c.ImmunePoison),
			c.DefenseHead, c.ResistPhysicalHead, c.ResistEnergyHead, c.ResistRadiationHead,
			c.DefenseTorso, c.ResistPhysicalTorso, c.ResistEnergyTorso, c.ResistRadiationTorso,
			c.DefenseLeftArm, c.ResistPhysicalLeftArm, c.ResistEnergyLeftArm, c.ResistRadiationLeftArm,
			c.DefenseRightArm, c.ResistPhysicalRightArm, c.ResistEnergyRightArm, c.ResistRadiationRightArm,
			c.DefenseLeftLeg, c.ResistPhysicalLeftLeg, c.ResistEnergyLeftLeg, c.ResistRadiationLeftLeg,
			c.DefenseRightLeg, c.ResistPhysicalRightLeg, c.ResistEnergyRightLeg, c.ResistRadiationRightLeg,
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
	var collapseEncounterDetails func()
	list.OnSelected = func(id widget.ListItemID) {
		selectedIndex = id
		if collapseEncounterDetails != nil {
			collapseEncounterDetails()
		}
		refreshSelected(selectedLabel, enc, selectedIndex)
	}

	encounterOrderBox := container.NewVBox()
	var rebuildEncounterOrder func()
	rebuildEncounterOrder = func() {
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
			combatantID := c.ID
			lineBtn := widget.NewButton(combatantLine(c), func() {
				selectedIndex = idx
				refreshSelected(selectedLabel, enc, selectedIndex)
				if expandedCombatantID == combatantID {
					expandedCombatantID = ""
				} else {
					expandedCombatantID = combatantID
				}
				rebuildEncounterOrder()
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
				collapseEncounterDetails()
				showApplyDamageDialogForIndex(idx)
			})
			healBtn := widget.NewButton("HEAL", func() {
				collapseEncounterDetails()
				showHealDialogForIndex(idx)
			})
			damageBtn.Importance = widget.LowImportance
			healBtn.Importance = widget.LowImportance

			details := widget.NewLabel(expandedCombatantDetails(c))
			details.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
			details.Wrapping = fyne.TextWrapWord
			details.Importance = widget.HighImportance
			detailsSection := container.NewVBox(lineBtn)
			if expandedCombatantID == combatantID {
				detailsSection.Add(details)
			}

			row := container.NewBorder(nil, nil, nil, container.NewGridWithColumns(2, damageBtn, healBtn), detailsSection)
			encounterOrderBox.Add(row)
		}
		encounterOrderBox.Refresh()
	}
	collapseEncounterDetails = func() {
		if expandedCombatantID == "" {
			return
		}
		expandedCombatantID = ""
		rebuildEncounterOrder()
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
		collapseEncounterDetails()
		_, err := svc.AdvanceTurn(ctx)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	partyAddBtn := widget.NewButton("+ AP", func() {
		collapseEncounterDetails()
		_, err := svc.AddPartyAP(ctx, 1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	partySpendBtn := widget.NewButton("- AP", func() {
		collapseEncounterDetails()
		_, err := svc.SpendPartyAP(ctx, 1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	threatAddBtn := widget.NewButton("+ Threat", func() {
		collapseEncounterDetails()
		_, err := svc.AddThreat(ctx, 1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	threatSpendBtn := widget.NewButton("- Threat", func() {
		collapseEncounterDetails()
		_, err := svc.SpendThreat(ctx, 1)
		handleErr(err)
		if err != nil {
			return
		}
		refresh()
	})
	applyDamageBtn := widget.NewButton("APPLY DAMAGE", func() {
		collapseEncounterDetails()
		showApplyDamageDialogForIndex(selectedIndex)
	})
	healBtn := widget.NewButton("HEAL", func() {
		collapseEncounterDetails()
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
	selectedPanel := pipPanel(
		"ACTIVE TARGET",
		container.NewVBox(selectedLabel, widget.NewSeparator(), container.NewGridWithColumns(2, applyDamageBtn, healBtn)),
	)
	logPanel := pipPanel("DATA LOG", logOutput)
	campOverviewLabel := widget.NewLabel("No active campaign")
	campOverviewLabel.TextStyle = fyne.TextStyle{Monospace: true}
	campOverviewLabel.Wrapping = fyne.TextWrapWord
	campSnapshotLabel := widget.NewLabel("No active encounter")
	campSnapshotLabel.TextStyle = fyne.TextStyle{Monospace: true}
	campSnapshotLabel.Wrapping = fyne.TextWrapWord

	campRosterOutput := widget.NewMultiLineEntry()
	campRosterOutput.TextStyle = fyne.TextStyle{Monospace: true}
	campRosterOutput.Wrapping = fyne.TextWrapWord
	campRosterOutput.SetMinRowsVisible(10)
	campRosterOutput.Disable()
	campRosterOutput.SetText("No active campaign")

	partyLibraryOutput := widget.NewMultiLineEntry()
	partyLibraryOutput.TextStyle = fyne.TextStyle{Monospace: true}
	partyLibraryOutput.Wrapping = fyne.TextWrapWord
	partyLibraryOutput.SetMinRowsVisible(10)
	partyLibraryOutput.Disable()
	partyLibraryOutput.SetText("No saved party members found")

	statTabContent := container.NewVBox(
		turnPanel,
		widget.NewSeparator(),
		resourcesPanel,
		widget.NewSeparator(),
		encounterOrderPanel,
	)
	statTabScroll := container.NewVScroll(statTabContent)
	campActionsPanel := pipPanel(
		"CAMPAIGN ACTIONS",
		container.NewGridWithColumns(
			4,
			widget.NewButton("OPEN CAMPAIGN", func() { showCampaignListDialog() }),
			widget.NewButton("NEW CAMPAIGN", func() { showCreateCampaignDialog() }),
			widget.NewButton("OPEN ENCOUNTER", func() { showEncounterListDialog() }),
			widget.NewButton("NEW ENCOUNTER", func() { showCreateEncounterDialog() }),
		),
	)
	campTabContent := container.NewVBox(
		pipPanel("CAMPAIGN OVERVIEW", campOverviewLabel),
		widget.NewSeparator(),
		pipPanel("TACTICAL SNAPSHOT", campSnapshotLabel),
		widget.NewSeparator(),
		container.NewGridWithColumns(
			2,
			pipPanel("ACTIVE ROSTER", campRosterOutput),
			pipPanel("PARTY LIBRARY", partyLibraryOutput),
		),
		widget.NewSeparator(),
		selectedPanel,
		widget.NewSeparator(),
		campActionsPanel,
	)
	campTabScroll := container.NewVScroll(campTabContent)
	dataTabContent := container.NewBorder(nil, nil, nil, nil, logPanel)

	tabs := container.NewAppTabs(
		container.NewTabItem("STAT", container.NewPadded(statTabScroll)),
		container.NewTabItem("CAMP", container.NewPadded(campTabScroll)),
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

		logs, err := svc.ListEncounterLogs(ctx, enc.ID)
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
	formatCampaignRoster := func(players []domain.NewCampaignPlayer) string {
		if len(players) == 0 {
			return "No active players in campaign"
		}
		lines := make([]string, 0, len(players))
		for i, p := range players {
			lines = append(lines, fmt.Sprintf(
				"[%02d] %s -> %s | Lvl:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
				i+1,
				p.PlayerName,
				p.Character.Name,
				p.Character.Level,
				p.Character.Initiative,
				p.Character.HP,
				p.Character.MaxHP,
				p.Character.Defense,
				formatDRValue(p.Character.ResistPoison, p.Character.ImmunePoison),
			))
		}
		return strings.Join(lines, "\n")
	}
	formatPartyLibrary := func(members []domain.Combatant) string {
		if len(members) == 0 {
			return "No saved party members found in database"
		}
		lines := make([]string, 0, len(members))
		for i, c := range members {
			lines = append(lines, fmt.Sprintf(
				"[%02d] %s | Lvl:%d Init:%d HP:%d/%d DEF:%d DR Poison:%s",
				i+1,
				c.Name,
				c.Level,
				c.Initiative,
				c.HP,
				c.MaxHP,
				c.Defense,
				formatDRValue(c.ResistPoison, c.ImmunePoison),
			))
		}
		return strings.Join(lines, "\n")
	}
	formatTacticalSnapshot := func(encounter *domain.Encounter) string {
		if encounter == nil {
			return "No active encounter"
		}
		partyTotal, partyAlive, partyDefeated := 0, 0, 0
		npcTotal, npcAlive, npcDefeated := 0, 0, 0
		activeName := "-"
		for i := range encounter.Combatants {
			c := encounter.Combatants[i]
			isDefeated := c.Defeated || c.HP <= 0
			if c.Active && !isDefeated {
				activeName = encounterDisplayNameByID(encounter, c.ID)
			}
			if c.Side == domain.SideParty {
				partyTotal++
				if isDefeated {
					partyDefeated++
				} else {
					partyAlive++
				}
				continue
			}
			npcTotal++
			if isDefeated {
				npcDefeated++
			} else {
				npcAlive++
			}
		}
		return fmt.Sprintf(
			"Encounter: %s\nRound: %d\nActive Turn: %s\nParty: %d total / %d alive / %d defeated\nNPC: %d total / %d alive / %d defeated",
			encounter.Name,
			encounter.Round,
			activeName,
			partyTotal,
			partyAlive,
			partyDefeated,
			npcTotal,
			npcAlive,
			npcDefeated,
		)
	}

	refresh = func() {
		var err error
		activeCampaign, err = svc.GetActiveCampaign(ctx)
		if err != nil {
			if errors.Is(err, domain.ErrCampaignNotInitialized) {
				activeCampaign = nil
				enc = nil
				expandedCombatantID = ""
				campaignStatusLabel.SetText("Campaign: -")
				campOverviewLabel.SetText("No active campaign")
				campSnapshotLabel.SetText("No active encounter")
				campRosterOutput.SetText("No active campaign")
				partyLibraryOutput.SetText("No active campaign")
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
		campOverviewLabel.SetText(fmt.Sprintf(
			"Name: %s\nID: %s\nStart Date: %s\nUpdated: %s",
			activeCampaign.Name,
			activeCampaign.ID,
			activeCampaign.StartDate,
			formatEncounterUpdatedAt(activeCampaign.UpdatedAt),
		))
		setupHint.SetText(fmt.Sprintf("Campaign: %s\nNo active encounter.\nCreate one from scratch to begin tracking combat.", activeCampaign.Name))
		players, rosterErr := svc.ListCampaignPlayers(ctx, activeCampaign.ID)
		if rosterErr != nil {
			handleErr(rosterErr)
			campRosterOutput.SetText("Failed to load campaign players")
		} else {
			campRosterOutput.SetText(formatCampaignRoster(players))
		}
		partyMembers, partyErr := svc.ListPartyMembers(ctx)
		if partyErr != nil {
			handleErr(partyErr)
			partyLibraryOutput.SetText("Failed to load party library")
		} else {
			partyLibraryOutput.SetText(formatPartyLibrary(partyMembers))
		}

		enc, err = svc.GetEncounter(ctx)
		if err != nil {
			if errors.Is(err, domain.ErrEncounterNotInitialized) {
				enc = nil
				expandedCombatantID = ""
				campSnapshotLabel.SetText("No active encounter\nUse NEW ENCOUNTER or OPEN ENCOUNTER to continue.")
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

		campSnapshotLabel.SetText(formatTacticalSnapshot(enc))
		roundLabel.SetText(fmt.Sprintf("Round: %d", enc.Round))
		expandedExists := false
		for i := range enc.Combatants {
			if enc.Combatants[i].ID == expandedCombatantID {
				expandedExists = true
				break
			}
		}
		if !expandedExists {
			expandedCombatantID = ""
		}

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

	showCreateCampaignDialog = func() {
		showCampaignEditorDialog(
			w,
			"Create Campaign",
			"Create",
			"",
			time.Now().Format("2006-01-02"),
			nil,
			func(name, startDate string, players []domain.NewCampaignPlayer) error {
				_, err := svc.ExecuteCreateCampaign(ctx, appsvc.CreateCampaignCommand{
					Name:      name,
					StartDate: startDate,
					Players:   players,
				})
				return err
			},
			refresh,
		)
	}

	showCampaignListDialog = func() {
		showCampaignListDialogWindow(ctx, w, svc, activeCampaign, showCreateCampaignDialog, refresh, handleErr)
	}

	showCreateEncounterDialog = func() {
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

	showApplyDamageDialogForIndex = func(targetIndex int) {
		showApplyDamageDialog(ctx, w, svc, enc, targetIndex, refresh)
	}

	showHealDialogForIndex = func(targetIndex int) {
		showHealDialog(ctx, w, svc, enc, targetIndex, refresh)
	}

	showEncounterListDialog = func() {
		showEncounterListDialogWindow(ctx, w, svc, refresh, handleErr)
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
