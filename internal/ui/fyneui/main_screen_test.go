package fyneui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMainScreenMakesActiveTargetCollapsible(t *testing.T) {
	app := test.NewTempApp(t)
	app.Settings().SetTheme(newPipBoyTheme())
	state := &uiState{enc: testEncounter()}
	labels := newMainScreenLabels()
	encounterOrder := newEncounterOrderView(
		&state.enc,
		&state.selectedIndex,
		&state.expandedCombatantID,
		labels.selectedLabel,
		func(int) {},
		func(int) {},
	)

	screen := newMainScreen(
		encounterOrder,
		labels,
		mainScreenActions{
			showCampaignList:    func() {},
			showCreateCampaign:  func() {},
			showEncounterList:   func() {},
			showCreateEncounter: func() {},
		},
		mainScreenControls{
			nextTurnBtn:    widget.NewButton("NEXT", func() {}),
			partyAddBtn:    widget.NewButton("+ AP", func() {}),
			partySpendBtn:  widget.NewButton("- AP", func() {}),
			threatAddBtn:   widget.NewButton("+ THREAT", func() {}),
			threatSpendBtn: widget.NewButton("- THREAT", func() {}),
			applyDamageBtn: widget.NewButton("DMG", func() {}),
			healBtn:        widget.NewButton("HEAL", func() {}),
		},
	)

	require.NotNil(t, screen.activeTargetAccordion)
	require.NotNil(t, screen.activeTarget)
	require.Len(t, screen.activeTargetAccordion.Items, 1)
	assert.Equal(t, "TARGET DETAILS", screen.activeTargetAccordion.Items[0].Title)
	assert.False(t, screen.activeTargetAccordion.Items[0].Open)
	assert.Equal(t, "No active target", screen.activeTarget.nameLabel.Text)
}
