package fyneui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
)

type uiController struct {
	ctx    context.Context
	window fyne.Window
	svc    *appsvc.Service
	state  *uiState

	encounterOrder *encounterOrderView
	refresh        func()
}

func newUIController(ctx context.Context, window fyne.Window, svc *appsvc.Service, state *uiState) *uiController {
	return &uiController{
		ctx:    ctx,
		window: window,
		svc:    svc,
		state:  state,
	}
}

func (c *uiController) handleErr(err error) {
	if err != nil {
		dialog.ShowError(err, c.window)
	}
}

func (c *uiController) newEncounterOrderView(selectedLabel *widget.Label) *encounterOrderView {
	c.encounterOrder = newEncounterOrderView(
		&c.state.enc,
		&c.state.selectedIndex,
		&c.state.expandedCombatantID,
		selectedLabel,
		func(idx int) { c.showApplyDamageDialogForIndex(idx) },
		func(idx int) { c.showHealDialogForIndex(idx) },
	)
	return c.encounterOrder
}

func (c *uiController) mainScreenActions() mainScreenActions {
	return mainScreenActions{
		showCampaignList:    c.showCampaignListDialog,
		showCreateCampaign:  c.showCreateCampaignDialog,
		showEncounterList:   c.showEncounterListDialog,
		showCreateEncounter: c.showCreateEncounterDialog,
	}
}

func (c *uiController) mainScreenControls() mainScreenControls {
	return mainScreenControls{
		nextTurnBtn:    c.newActionButton("Next Turn", uiActionPrimary, c.advanceTurn),
		partyAddBtn:    c.newActionButton("+ AP", uiActionSecondary, c.addPartyAP),
		partySpendBtn:  c.newActionButton("- AP", uiActionSubtle, c.spendPartyAP),
		threatAddBtn:   c.newActionButton("+ Threat", uiActionSecondary, c.addThreat),
		threatSpendBtn: c.newActionButton("- Threat", uiActionSubtle, c.spendThreat),
		applyDamageBtn: newRoleButton("APPLY DAMAGE", uiActionDestructive, func() {
			c.encounterOrder.CollapseDetails()
			c.showApplyDamageDialogForIndex(c.state.selectedIndex)
		}),
		healBtn: newRoleButton("HEAL", uiActionSuccess, func() {
			c.encounterOrder.CollapseDetails()
			c.showHealDialogForIndex(c.state.selectedIndex)
		}),
	}
}

func (c *uiController) newActionButton(label string, role uiActionRole, run func() error) *widget.Button {
	return styleButtonRole(
		newRefreshingActionButton(label, c.encounterOrder.CollapseDetails, c.handleErr, c.refreshAfterAction, run),
		role,
	)
}

func (c *uiController) advanceTurn() error {
	_, err := c.svc.AdvanceTurn(c.ctx)
	return err
}

func (c *uiController) addPartyAP() error {
	_, err := c.svc.AddPartyAP(c.ctx, 1)
	return err
}

func (c *uiController) spendPartyAP() error {
	_, err := c.svc.SpendPartyAP(c.ctx, 1)
	return err
}

func (c *uiController) addThreat() error {
	_, err := c.svc.AddThreat(c.ctx, 1)
	return err
}

func (c *uiController) spendThreat() error {
	_, err := c.svc.SpendThreat(c.ctx, 1)
	return err
}

func (c *uiController) refreshAfterAction() {
	if c.refresh != nil {
		c.refresh()
	}
}

func (c *uiController) showCreateCampaignDialog() {
	showCreateCampaignDialogWindow(c.ctx, c.window, c.svc, c.refresh)
}

func (c *uiController) showCampaignListDialog() {
	showCampaignListDialogWindow(c.ctx, c.window, c.svc, c.state.activeCampaign, c.showCreateCampaignDialog, c.refresh, c.handleErr)
}

func (c *uiController) showCreateEncounterDialog() {
	showCreateEncounterDialogWindow(c.ctx, c.window, c.svc, c.refresh)
}

func (c *uiController) showEncounterListDialog() {
	showEncounterListDialogWindow(c.ctx, c.window, c.svc, c.refresh, c.handleErr)
}

func (c *uiController) showApplyDamageDialogForIndex(targetIndex int) {
	showApplyDamageDialog(c.ctx, c.window, c.svc, c.state.enc, targetIndex, c.refresh)
}

func (c *uiController) showHealDialogForIndex(targetIndex int) {
	showHealDialog(c.ctx, c.window, c.svc, c.state.enc, targetIndex, c.refresh)
}
