package fyneui

import (
	"fmt"

	"fyne.io/fyne/v2/widget"

	"github.com/obalunenko/fallout/internal/domain"
)

type mainViewPresenter struct {
	state          *uiState
	screen         *mainScreen
	encounterOrder *encounterOrderView
}

func newMainViewPresenter(state *uiState, screen *mainScreen, encounterOrder *encounterOrderView) *mainViewPresenter {
	return &mainViewPresenter{
		state:          state,
		screen:         screen,
		encounterOrder: encounterOrder,
	}
}

func (p *mainViewPresenter) showNoCampaign() {
	p.state.activeCampaign = nil
	p.state.enc = nil
	p.state.expandedCombatantID = ""
	p.screen.campaignStatusLabel.SetText("Campaign: -")
	p.screen.campOverviewLabel.SetText("No active campaign")
	p.screen.campSnapshotLabel.SetText("No active encounter")
	p.screen.campRosterOutput.SetText("No active campaign")
	p.screen.partyLibraryOutput.SetText("No active campaign")
	p.screen.roundLabel.SetText("Round: -")
	p.refreshEncounterWidgets()
	p.hideActiveTargetPanel()
	p.screen.tabsView.Hide()
	p.screen.noEncounterView.Hide()
	p.screen.noCampaignView.Show()
	p.screen.mainView.Refresh()
}

func (p *mainViewPresenter) showActiveCampaign(campaign *domain.Campaign) {
	p.screen.campaignStatusLabel.SetText(fmt.Sprintf("Campaign: %s (%s)", campaign.Name, formatCampaignStartDate(campaign.StartDate)))
	p.screen.campOverviewLabel.SetText(formatCampaignOverview(campaign))
	p.screen.setupHint.SetText(fmt.Sprintf("Campaign: %s\nNo active encounter.\nCreate one from scratch to begin tracking combat.", campaign.Name))
}

func (p *mainViewPresenter) showNoEncounter() {
	p.state.enc = nil
	p.state.expandedCombatantID = ""
	p.screen.campSnapshotLabel.SetText("No active encounter\nUse NEW ENCOUNTER or OPEN ENCOUNTER to continue.")
	p.screen.roundLabel.SetText("Round: -")
	p.refreshEncounterWidgets()
	p.hideActiveTargetPanel()
	p.screen.tabsView.Hide()
	p.screen.noCampaignView.Hide()
	p.screen.noEncounterView.Show()
	p.screen.mainView.Refresh()
}

func (p *mainViewPresenter) hideActiveTargetPanel() {
	if p.screen.activeTargetPanel == nil {
		return
	}
	p.screen.activeTargetPanel.Hide()
	p.screen.activeTargetPanel.Refresh()
}

func (p *mainViewPresenter) showActiveEncounter() {
	enc := p.state.enc
	p.screen.campSnapshotLabel.SetText(formatTacticalSnapshot(enc))
	p.screen.roundLabel.SetText(fmt.Sprintf("Round: %d", enc.Round))
	p.clearMissingExpandedCombatant()
	p.clampSelectedIndex()
	p.refreshEncounterWidgets()
	p.screen.noCampaignView.Hide()
	p.screen.noEncounterView.Hide()
	p.screen.tabsView.Show()
	p.screen.mainView.Refresh()
}

func (p *mainViewPresenter) showCampaignRoster(players []domain.NewCampaignPlayer) {
	p.screen.campRosterOutput.SetText(formatCampaignRoster(players))
}

func (p *mainViewPresenter) showCampaignRosterError() {
	p.screen.campRosterOutput.SetText("Failed to load campaign players")
}

func (p *mainViewPresenter) showPartyLibrary(members []domain.Combatant) {
	p.screen.partyLibraryOutput.SetText(formatPartyLibrary(members))
}

func (p *mainViewPresenter) showPartyLibraryError() {
	p.screen.partyLibraryOutput.SetText("Failed to load party library")
}

func (p *mainViewPresenter) logOutput() *widget.Entry {
	return p.screen.logOutput
}

func (p *mainViewPresenter) refreshEncounterWidgets() {
	if p.screen.activeTarget != nil {
		p.screen.activeTarget.SetTarget(p.state.enc, p.state.selectedIndex)
	} else {
		refreshSelected(p.screen.selectedLabel, p.state.enc, p.state.selectedIndex)
	}
	refreshActiveTurnLabel(p.state.enc, p.screen.activeTurnLabel)
	refreshResourceLabels(p.state.enc, p.screen.partyAPLabel, p.screen.threatLabel)
	p.encounterOrder.List().Refresh()
	p.encounterOrder.Rebuild()
}

func (p *mainViewPresenter) clearMissingExpandedCombatant() {
	expandedExists := false
	for i := range p.state.enc.Combatants {
		if p.state.enc.Combatants[i].ID == p.state.expandedCombatantID {
			expandedExists = true
			break
		}
	}
	if !expandedExists {
		p.state.expandedCombatantID = ""
	}
}

func (p *mainViewPresenter) clampSelectedIndex() {
	if p.state.selectedIndex >= len(p.state.enc.Combatants) {
		p.state.selectedIndex = 0
	}
}
