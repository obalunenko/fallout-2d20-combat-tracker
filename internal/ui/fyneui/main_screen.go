package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type mainScreenActions struct {
	showCampaignList    func()
	showCreateCampaign  func()
	showEncounterList   func()
	showCreateEncounter func()
}

type mainScreenLabels struct {
	roundLabel      *widget.Label
	activeTurnLabel *widget.Label
	selectedLabel   *widget.Label
	partyAPLabel    *widget.Label
	threatLabel     *widget.Label
	logOutput       *widget.Entry
}

func newMainScreenLabels() mainScreenLabels {
	roundLabel := widget.NewLabel("")
	activeTurnLabel := widget.NewLabel("")
	selectedLabel := widget.NewLabel("")
	partyAPLabel := widget.NewLabel("")
	threatLabel := widget.NewLabel("")

	roundLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	activeTurnLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	activeTurnLabel.Importance = widget.HighImportance
	activeTurnLabel.Wrapping = fyne.TextWrapWord
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	partyAPLabel.TextStyle = fyne.TextStyle{Monospace: true}
	threatLabel.TextStyle = fyne.TextStyle{Monospace: true}

	return mainScreenLabels{
		roundLabel:      roundLabel,
		activeTurnLabel: activeTurnLabel,
		selectedLabel:   selectedLabel,
		partyAPLabel:    partyAPLabel,
		threatLabel:     threatLabel,
		logOutput:       newReadOnlyMonospaceOutput("[BOOT] Pip-Boy combat tracker initialized", 18),
	}
}

type mainScreenControls struct {
	nextTurnBtn,
	partyAddBtn,
	partySpendBtn,
	threatAddBtn,
	threatSpendBtn,
	applyDamageBtn,
	healBtn *widget.Button
}

type mainScreen struct {
	roundLabel          *widget.Label
	activeTurnLabel     *widget.Label
	selectedLabel       *widget.Label
	partyAPLabel        *widget.Label
	threatLabel         *widget.Label
	campOverviewLabel   *widget.Label
	campSnapshotLabel   *widget.Label
	campaignStatusLabel *widget.Label
	setupHint           *widget.Label

	logOutput          *widget.Entry
	campRosterOutput   *widget.Entry
	partyLibraryOutput *widget.Entry

	activeTarget          *activeTargetView
	activeTargetAccordion *widget.Accordion
	activeTargetPanel     fyne.CanvasObject

	tabsView        fyne.CanvasObject
	noEncounterView fyne.CanvasObject
	noCampaignView  fyne.CanvasObject
	mainView        fyne.CanvasObject
	content         fyne.CanvasObject
}

func newMainScreen(encounterOrder *encounterOrderView, labels mainScreenLabels, actions mainScreenActions, controls mainScreenControls) *mainScreen {
	screen := &mainScreen{
		roundLabel:         labels.roundLabel,
		activeTurnLabel:    labels.activeTurnLabel,
		selectedLabel:      labels.selectedLabel,
		partyAPLabel:       labels.partyAPLabel,
		threatLabel:        labels.threatLabel,
		logOutput:          labels.logOutput,
		campOverviewLabel:  newWrappedMonospaceLabel("No active campaign"),
		campSnapshotLabel:  newWrappedMonospaceLabel("No active encounter"),
		campRosterOutput:   newReadOnlyMonospaceOutput("No active campaign", 10),
		partyLibraryOutput: newReadOnlyMonospaceOutput("No saved party members found", 10),
	}

	turnPanel := pipPanel(
		"TURN CONTROL",
		container.NewVBox(
			screen.roundLabel,
			screen.activeTurnLabel,
			controls.nextTurnBtn,
		),
	)
	resourcesPanel := pipPanel(
		"RESOURCES",
		container.NewGridWithColumns(
			6,
			screen.partyAPLabel,
			controls.partyAddBtn,
			controls.partySpendBtn,
			screen.threatLabel,
			controls.threatAddBtn,
			controls.threatSpendBtn,
		),
	)
	encounterOrderPanel := pipPanel("ENCOUNTER ORDER", encounterOrder.OrderBox())
	screen.activeTarget = newActiveTargetView(screen.selectedLabel, controls.applyDamageBtn, controls.healBtn)
	screen.activeTargetAccordion = screen.activeTarget.accordion
	selectedPanel := pipPanel(
		"ACTIVE TARGET",
		screen.activeTarget.Root(),
	)
	selectedPanel.Hide()
	screen.activeTargetPanel = selectedPanel
	encounterOrder.SetOnSelect(func(idx int, repeatedSelection bool) {
		if repeatedSelection && selectedPanel.Visible() {
			screen.activeTarget.SetTarget(nil, 0)
			selectedPanel.Hide()
			selectedPanel.Refresh()
			return
		}
		screen.activeTarget.SetTarget(encounterOrder.currentEncounter(), idx)
		selectedPanel.Show()
		selectedPanel.Refresh()
	})
	logPanel := pipPanel("DATA LOG", screen.logOutput)

	statLeftControls := container.NewVBox(
		turnPanel,
		widget.NewSeparator(),
		resourcesPanel,
		widget.NewSeparator(),
	)
	statLeft := container.NewBorder(
		statLeftControls,
		nil,
		nil,
		nil,
		container.NewVScroll(encounterOrderPanel),
	)
	statTabContent := container.NewHSplit(
		statLeft,
		container.NewVScroll(selectedPanel),
	)
	statTabContent.Offset = 0.70
	campActionsPanel := pipPanel(
		"CAMPAIGN ACTIONS",
		container.NewGridWithColumns(
			4,
			newRoleButton("OPEN CAMPAIGN", uiActionSecondary, func() { actions.showCampaignList() }),
			newRoleButton("NEW CAMPAIGN", uiActionSecondary, func() { actions.showCreateCampaign() }),
			newRoleButton("OPEN ENCOUNTER", uiActionSecondary, func() { actions.showEncounterList() }),
			newRoleButton("NEW ENCOUNTER", uiActionPrimary, func() { actions.showCreateEncounter() }),
		),
	)
	campTabContent := container.NewVBox(
		pipPanel("CAMPAIGN OVERVIEW", screen.campOverviewLabel),
		widget.NewSeparator(),
		pipPanel("TACTICAL SNAPSHOT", screen.campSnapshotLabel),
		widget.NewSeparator(),
		container.NewGridWithColumns(
			2,
			pipPanel("ACTIVE ROSTER", screen.campRosterOutput),
			pipPanel("PARTY LIBRARY", screen.partyLibraryOutput),
		),
		widget.NewSeparator(),
		campActionsPanel,
	)
	campTabScroll := container.NewVScroll(campTabContent)
	dataTabContent := container.NewBorder(nil, nil, nil, nil, logPanel)

	tabs := container.NewAppTabs(
		container.NewTabItem("STAT", container.NewPadded(statTabContent)),
		container.NewTabItem("CAMP", container.NewPadded(campTabScroll)),
		container.NewTabItem("DATA", container.NewPadded(dataTabContent)),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	screen.tabsView = container.New(layout.NewPaddedLayout(), tabs)

	screen.campaignStatusLabel = newMonospaceLabel("Campaign: -")

	newEncounterBtn := newRoleButton("NEW ENCOUNTER", uiActionPrimary, func() {
		actions.showCreateEncounter()
	})
	openEncounterBtn := newRoleButton("OPEN ENCOUNTER", uiActionSubtle, func() {
		actions.showEncounterList()
	})
	newCampaignBtn := newRoleButton("NEW CAMPAIGN", uiActionSecondary, func() {
		actions.showCreateCampaign()
	})
	openCampaignBtn := newRoleButton("OPEN CAMPAIGN", uiActionSubtle, func() {
		actions.showCampaignList()
	})

	screen.setupHint = newMonospaceLabel("No active encounter.\nCreate one from scratch to begin tracking combat.")
	screen.setupHint.Alignment = fyne.TextAlignCenter
	setupButton := newRoleButton("CREATE ENCOUNTER", uiActionPrimary, func() {
		actions.showCreateEncounter()
	})
	screen.noEncounterView = pipPanel(
		"SYSTEM",
		container.NewCenter(container.NewVBox(screen.setupHint, widget.NewSeparator(), setupButton)),
	)

	campaignHint := newMonospaceLabel("No active campaign.\nCreate or choose a campaign before running encounters.")
	campaignHint.Alignment = fyne.TextAlignCenter
	campaignActions := container.NewGridWithColumns(2,
		newRoleButton("CREATE CAMPAIGN", uiActionPrimary, func() { actions.showCreateCampaign() }),
		newRoleButton("OPEN CAMPAIGNS", uiActionSecondary, func() { actions.showCampaignList() }),
	)
	screen.noCampaignView = pipPanel(
		"CAMPAIGN CONTROL",
		container.NewCenter(container.NewVBox(campaignHint, widget.NewSeparator(), campaignActions)),
	)

	screen.mainView = container.NewStack(screen.tabsView, screen.noEncounterView, screen.noCampaignView)
	screen.content = newMainContentWithHeader(
		screen.campaignStatusLabel,
		screen.mainView,
		openCampaignBtn,
		openEncounterBtn,
		newCampaignBtn,
		newEncounterBtn,
	)
	return screen
}
