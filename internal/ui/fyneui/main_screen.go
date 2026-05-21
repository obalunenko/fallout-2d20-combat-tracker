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
	roundLabel    *widget.Label
	selectedLabel *widget.Label
	partyAPLabel  *widget.Label
	threatLabel   *widget.Label
	logOutput     *widget.Entry
}

func newMainScreenLabels() mainScreenLabels {
	roundLabel := widget.NewLabel("")
	selectedLabel := widget.NewLabel("")
	partyAPLabel := widget.NewLabel("")
	threatLabel := widget.NewLabel("")

	roundLabel.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	selectedLabel.TextStyle = fyne.TextStyle{Monospace: true}
	partyAPLabel.TextStyle = fyne.TextStyle{Monospace: true}
	threatLabel.TextStyle = fyne.TextStyle{Monospace: true}

	return mainScreenLabels{
		roundLabel:    roundLabel,
		selectedLabel: selectedLabel,
		partyAPLabel:  partyAPLabel,
		threatLabel:   threatLabel,
		logOutput:     newReadOnlyMonospaceOutput("[BOOT] Pip-Boy combat tracker initialized", 18),
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

	tabsView        fyne.CanvasObject
	noEncounterView fyne.CanvasObject
	noCampaignView  fyne.CanvasObject
	mainView        fyne.CanvasObject
	content         fyne.CanvasObject
}

func newMainScreen(encounterOrder *encounterOrderView, labels mainScreenLabels, actions mainScreenActions, controls mainScreenControls) *mainScreen {
	screen := &mainScreen{
		roundLabel:         labels.roundLabel,
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
	selectedPanel := pipPanel(
		"ACTIVE TARGET",
		container.NewVBox(screen.selectedLabel, widget.NewSeparator(), container.NewGridWithColumns(2, controls.applyDamageBtn, controls.healBtn)),
	)
	logPanel := pipPanel("DATA LOG", screen.logOutput)

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
			widget.NewButton("OPEN CAMPAIGN", func() { actions.showCampaignList() }),
			widget.NewButton("NEW CAMPAIGN", func() { actions.showCreateCampaign() }),
			widget.NewButton("OPEN ENCOUNTER", func() { actions.showEncounterList() }),
			widget.NewButton("NEW ENCOUNTER", func() { actions.showCreateEncounter() }),
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
	screen.tabsView = container.New(layout.NewPaddedLayout(), tabs)

	screen.campaignStatusLabel = newMonospaceLabel("Campaign: -")

	newEncounterBtn := widget.NewButton("NEW ENCOUNTER", func() {
		actions.showCreateEncounter()
	})
	openEncounterBtn := widget.NewButton("OPEN ENCOUNTER", func() {
		actions.showEncounterList()
	})
	newCampaignBtn := widget.NewButton("NEW CAMPAIGN", func() {
		actions.showCreateCampaign()
	})
	openCampaignBtn := widget.NewButton("OPEN CAMPAIGN", func() {
		actions.showCampaignList()
	})

	screen.setupHint = newMonospaceLabel("No active encounter.\nCreate one from scratch to begin tracking combat.")
	screen.setupHint.Alignment = fyne.TextAlignCenter
	setupButton := widget.NewButton("CREATE ENCOUNTER", func() {
		actions.showCreateEncounter()
	})
	screen.noEncounterView = pipPanel(
		"SYSTEM",
		container.NewCenter(container.NewVBox(screen.setupHint, widget.NewSeparator(), setupButton)),
	)

	campaignHint := newMonospaceLabel("No active campaign.\nCreate or choose a campaign before running encounters.")
	campaignHint.Alignment = fyne.TextAlignCenter
	campaignActions := container.NewGridWithColumns(2,
		widget.NewButton("CREATE CAMPAIGN", func() { actions.showCreateCampaign() }),
		widget.NewButton("OPEN CAMPAIGNS", func() { actions.showCampaignList() }),
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
