package fyneui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestMainViewPresenterShowNoCampaignResetsStateAndShowsCampaignGate(t *testing.T) {
	state := &uiState{
		enc:                 testEncounter(),
		activeCampaign:      &domain.Campaign{ID: "camp-1", Name: "Old campaign", StartDate: testCampaignDate()},
		selectedIndex:       1,
		expandedCombatantID: "pc-1",
	}
	presenter, screen := newTestMainViewPresenter(t, state)

	presenter.showNoCampaign()

	assert.Nil(t, state.activeCampaign)
	assert.Nil(t, state.enc)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "Campaign: -", screen.campaignStatusLabel.Text)
	assert.Equal(t, "No active campaign", screen.campOverviewLabel.Text)
	assert.Equal(t, "No active encounter", screen.campSnapshotLabel.Text)
	assert.Equal(t, "No active campaign", screen.campRosterOutput.Text)
	assert.Equal(t, "No active campaign", screen.partyLibraryOutput.Text)
	assert.Equal(t, "Round: -", screen.roundLabel.Text)
	assert.Equal(t, "No combatants", screen.selectedLabel.Text)
	assert.Equal(t, "Party AP: 0", screen.partyAPLabel.Text)
	assert.Equal(t, "GM Threat: 0", screen.threatLabel.Text)
	assert.False(t, screen.tabsView.Visible())
	assert.False(t, screen.noEncounterView.Visible())
	assert.True(t, screen.noCampaignView.Visible())
}

func TestMainViewPresenterShowNoEncounterKeepsCampaignAndShowsEncounterGate(t *testing.T) {
	activeCampaign := &domain.Campaign{ID: "camp-1", Name: "Vault 13", StartDate: testCampaignDate()}
	state := &uiState{
		enc:                 testEncounter(),
		activeCampaign:      activeCampaign,
		selectedIndex:       1,
		expandedCombatantID: "npc-1",
	}
	presenter, screen := newTestMainViewPresenter(t, state)

	presenter.showNoEncounter()

	require.Same(t, activeCampaign, state.activeCampaign)
	assert.Nil(t, state.enc)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "No active encounter\nUse NEW ENCOUNTER or OPEN ENCOUNTER to continue.", screen.campSnapshotLabel.Text)
	assert.Equal(t, "Round: -", screen.roundLabel.Text)
	assert.Equal(t, "No combatants", screen.selectedLabel.Text)
	assert.Equal(t, "Party AP: 0", screen.partyAPLabel.Text)
	assert.Equal(t, "GM Threat: 0", screen.threatLabel.Text)
	assert.False(t, screen.tabsView.Visible())
	assert.True(t, screen.noEncounterView.Visible())
	assert.False(t, screen.noCampaignView.Visible())
}

func TestMainViewPresenterShowActiveEncounterRefreshesEncounterWidgets(t *testing.T) {
	enc := testEncounter()
	enc.Resources = domain.Resources{PartyAP: 3, GMThreat: 2}
	enc.Round = 4
	state := &uiState{
		enc:                 enc,
		activeCampaign:      &domain.Campaign{ID: "camp-1", Name: "Vault 13", StartDate: testCampaignDate()},
		selectedIndex:       99,
		expandedCombatantID: "missing",
	}
	presenter, screen := newTestMainViewPresenter(t, state)

	presenter.showActiveEncounter()

	assert.Equal(t, 0, state.selectedIndex)
	assert.Empty(t, state.expandedCombatantID)
	assert.Equal(t, "Round: 4", screen.roundLabel.Text)
	assert.Equal(t, "Party AP: 3", screen.partyAPLabel.Text)
	assert.Equal(t, "GM Threat: 2", screen.threatLabel.Text)
	assert.Contains(t, screen.selectedLabel.Text, "Participant Details")
	assert.Contains(t, screen.selectedLabel.Text, "Name")
	assert.Contains(t, screen.selectedLabel.Text, "Alpha")
	assert.Contains(t, screen.campSnapshotLabel.Text, "Encounter: Test Encounter")
	assert.True(t, screen.tabsView.Visible())
	assert.False(t, screen.noEncounterView.Visible())
	assert.False(t, screen.noCampaignView.Visible())
	assert.NotEmpty(t, presenter.encounterOrder.OrderBox().(*fyne.Container).Objects)
}

func TestMainViewPresenterShowActiveCampaignUpdatesCampaignCopy(t *testing.T) {
	state := &uiState{}
	presenter, screen := newTestMainViewPresenter(t, state)

	presenter.showActiveCampaign(&domain.Campaign{ID: "camp-1", Name: "Vault 13", StartDate: testCampaignDate()})

	assert.Equal(t, "Campaign: Vault 13 (2026-01-01)", screen.campaignStatusLabel.Text)
	assert.Contains(t, screen.campOverviewLabel.Text, "Name: Vault 13")
	assert.Contains(t, screen.setupHint.Text, "Campaign: Vault 13")
}

func newTestMainViewPresenter(t *testing.T, state *uiState) (*mainViewPresenter, *mainScreen) {
	t.Helper()
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())

	labels := newMainScreenLabels()
	encounterOrder := newEncounterOrderView(
		&state.enc,
		&state.selectedIndex,
		&state.expandedCombatantID,
		labels.selectedLabel,
		func(int) {},
		func(int) {},
	)
	screen := newTestMainScreen(labels)
	return newMainViewPresenter(state, screen, encounterOrder), screen
}

func newTestMainScreen(labels mainScreenLabels) *mainScreen {
	tabsView := canvas.NewRectangle(nil)
	noEncounterView := canvas.NewRectangle(nil)
	noCampaignView := canvas.NewRectangle(nil)
	activeTarget := newActiveTargetView(
		labels.selectedLabel,
		widget.NewButton("DMG", func() {}),
		widget.NewButton("HEAL", func() {}),
	)

	return &mainScreen{
		roundLabel:            labels.roundLabel,
		selectedLabel:         labels.selectedLabel,
		partyAPLabel:          labels.partyAPLabel,
		threatLabel:           labels.threatLabel,
		logOutput:             labels.logOutput,
		activeTarget:          activeTarget,
		activeTargetAccordion: activeTarget.accordion,
		campOverviewLabel:     widget.NewLabel("No active campaign"),
		campSnapshotLabel:     widget.NewLabel("No active encounter"),
		campaignStatusLabel:   widget.NewLabel("Campaign: -"),
		setupHint:             widget.NewLabel("No active encounter.\nCreate one from scratch to begin tracking combat."),
		campRosterOutput:      widget.NewMultiLineEntry(),
		partyLibraryOutput:    widget.NewMultiLineEntry(),
		tabsView:              tabsView,
		noEncounterView:       noEncounterView,
		noCampaignView:        noCampaignView,
		mainView:              container.NewStack(tabsView, noEncounterView, noCampaignView),
	}
}

func testEncounter() *domain.Encounter {
	return domain.NewEncounter("enc-1", "Test Encounter", []domain.Combatant{
		{ID: "pc-1", Name: "Alpha", Side: domain.SideParty, Initiative: 12, HP: 8, MaxHP: 8},
		{ID: "npc-1", Name: "Raider", Side: domain.SideNPC, Initiative: 8, HP: 6, MaxHP: 6},
	})
}

func testCampaignDate() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
