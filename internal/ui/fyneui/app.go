package fyneui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"strconv"
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
			11,
			newTableHeaderLabel("Player"),
			newTableHeaderLabel("Character"),
			newTableHeaderLabel("Level"),
			newTableHeaderLabel("Init"),
			newTableHeaderLabel("HP Cur"),
			newTableHeaderLabel("HP Max"),
			newTableHeaderLabel("DEF Base"),
			newTableHeaderLabel("DR Poison"),
			newTableHeaderLabel("DR Details"),
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
				defenseHead := p.Character.DefenseHead
				defenseTorso := p.Character.DefenseTorso
				defenseLA := p.Character.DefenseLeftArm
				defenseRA := p.Character.DefenseRightArm
				defenseLL := p.Character.DefenseLeftLeg
				defenseRL := p.Character.DefenseRightLeg
				row.defenseHead.SetText(strconv.Itoa(defenseHead))
				row.defenseTorso.SetText(strconv.Itoa(defenseTorso))
				row.defenseLA.SetText(strconv.Itoa(defenseLA))
				row.defenseRA.SetText(strconv.Itoa(defenseRA))
				row.defenseLL.SetText(strconv.Itoa(defenseLL))
				row.defenseRL.SetText(strconv.Itoa(defenseRL))
				row.drEnergyHead.SetText(strconv.Itoa(p.Character.ResistEnergyHead))
				row.drEnergyTorso.SetText(strconv.Itoa(p.Character.ResistEnergyTorso))
				row.drEnergyLA.SetText(strconv.Itoa(p.Character.ResistEnergyLeftArm))
				row.drEnergyRA.SetText(strconv.Itoa(p.Character.ResistEnergyRightArm))
				row.drEnergyLL.SetText(strconv.Itoa(p.Character.ResistEnergyLeftLeg))
				row.drEnergyRL.SetText(strconv.Itoa(p.Character.ResistEnergyRightLeg))
				row.drRadHead.SetText(strconv.Itoa(p.Character.ResistRadiationHead))
				row.drRadTorso.SetText(strconv.Itoa(p.Character.ResistRadiationTorso))
				row.drRadLA.SetText(strconv.Itoa(p.Character.ResistRadiationLeftArm))
				row.drRadRA.SetText(strconv.Itoa(p.Character.ResistRadiationRightArm))
				row.drRadLL.SetText(strconv.Itoa(p.Character.ResistRadiationLeftLeg))
				row.drRadRL.SetText(strconv.Itoa(p.Character.ResistRadiationRightLeg))
				row.immPhysical.SetChecked(p.Character.ImmunePhysical)
				if !p.Character.ImmunePhysical {
					row.drPhysHead.SetText(strconv.Itoa(p.Character.ResistPhysicalHead))
					row.drPhysTorso.SetText(strconv.Itoa(p.Character.ResistPhysicalTorso))
					row.drPhysLA.SetText(strconv.Itoa(p.Character.ResistPhysicalLeftArm))
					row.drPhysRA.SetText(strconv.Itoa(p.Character.ResistPhysicalRightArm))
					row.drPhysLL.SetText(strconv.Itoa(p.Character.ResistPhysicalLeftLeg))
					row.drPhysRL.SetText(strconv.Itoa(p.Character.ResistPhysicalRightLeg))
				}
				row.immEnergy.SetChecked(p.Character.ImmuneEnergy)
				row.immRadiation.SetChecked(p.Character.ImmuneRadiation)
				row.immPoison.SetChecked(p.Character.ImmunePoison)
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
				_, err := svc.ExecuteCreateCampaign(ctx, appsvc.CreateCampaignCommand{
					Name:      name,
					StartDate: startDate,
					Players:   players,
				})
				return err
			},
		)
	}

	showCampaignListDialog = func() {
		campaigns, err := svc.ListCampaigns(ctx)
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
			if _, err := svc.ActivateCampaign(ctx, selectedID); err != nil {
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
			players, err := svc.ListCampaignPlayers(ctx, selectedID)
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
					_, updateErr := svc.ExecuteUpdateCampaign(ctx, appsvc.UpdateCampaignCommand{
						CampaignID: current.ID,
						Name:       name,
						StartDate:  startDate,
						Players:    editedPlayers,
					})
					return updateErr
				},
			)
		})
		infoBtn := widget.NewButton("Use Selected", func() {
			if selectedIdx >= 0 && selectedIdx < len(campaigns) {
				if _, err := svc.ActivateCampaign(ctx, campaigns[selectedIdx].ID); err != nil {
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
				_, err := svc.ExecuteCreateEncounter(ctx, appsvc.CreateEncounterCommand{
					Name:       name,
					Combatants: combatants,
				})
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
		locationOptions := []string{"head", "torso", "left_arm", "right_arm", "left_leg", "right_leg"}
		if isTorsoOnlyCombatant(target) {
			locationOptions = []string{"torso"}
		}
		locationSelect := widget.NewSelect(locationOptions, nil)
		locationSelect.SetSelected("torso")

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
				widget.NewFormItem("Location", locationSelect),
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
				location, err := parseBodyLocation(locationSelect.Selected)
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

				_, _, err = svc.ExecuteApplyDamage(ctx, appsvc.ApplyDamageCommand{
					CombatantID: target.ID,
					DamageType:  damageType,
					Location:    location,
					Amount:      amount,
				})
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

				_, _, err = svc.ExecuteHeal(ctx, appsvc.HealCommand{
					CombatantID: target.ID,
					Amount:      amount,
				})
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
		summaries, err := svc.ListEncounters(ctx)
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
			updated, err := svc.ListEncounters(ctx)
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
			_, err := svc.ActivateEncounter(ctx, selectedID)
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
			_, err := svc.RestartEncounter(ctx, selectedID)
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

					if err := svc.DeleteEncounter(ctx, targetID); err != nil {
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
			encForEdit, err := svc.GetEncounterByID(ctx, targetID)
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
					_, updateErr := svc.ExecuteUpdateEncounter(ctx, appsvc.UpdateEncounterCommand{
						EncounterID: targetID,
						Name:        name,
						Combatants:  combatants,
					})
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
