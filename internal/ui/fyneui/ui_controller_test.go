package fyneui

import (
	"context"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func TestUIControllerResourceButtonsRunServiceActionsAndRefresh(t *testing.T) {
	test.NewTempApp(t)
	encounter := testEncounter()
	encounter.Resources = domain.Resources{PartyAP: 1, GMThreat: 1}
	state := &uiState{enc: encounter}
	repo := &refresherRepo{encounter: encounter}
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)
	controller := newUIController(context.Background(), nil, svc, state)
	labels := newMainScreenLabels()
	controller.newEncounterOrderView(labels.selectedLabel)

	refreshes := 0
	controller.refresh = func() { refreshes++ }
	controls := controller.mainScreenControls()

	controls.partyAddBtn.OnTapped()
	assert.Equal(t, 2, encounter.Resources.PartyAP)
	assert.Equal(t, 1, refreshes)

	controls.partySpendBtn.OnTapped()
	assert.Equal(t, 1, encounter.Resources.PartyAP)
	assert.Equal(t, 2, refreshes)

	controls.threatAddBtn.OnTapped()
	assert.Equal(t, 2, encounter.Resources.GMThreat)
	assert.Equal(t, 3, refreshes)

	controls.threatSpendBtn.OnTapped()
	assert.Equal(t, 1, encounter.Resources.GMThreat)
	assert.Equal(t, 4, refreshes)
}

func TestUIControllerNextTurnButtonAdvancesEncounterAndRefreshes(t *testing.T) {
	test.NewTempApp(t)
	encounter := domain.NewEncounter("enc-1", "Turns", []domain.Combatant{
		{ID: "pc-1", Name: "Alpha", Initiative: 12, HP: 8, MaxHP: 8},
		{ID: "npc-1", Name: "Raider", Initiative: 8, HP: 6, MaxHP: 6},
	})
	state := &uiState{enc: encounter}
	repo := &refresherRepo{encounter: encounter}
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)
	controller := newUIController(context.Background(), nil, svc, state)
	labels := newMainScreenLabels()
	controller.newEncounterOrderView(labels.selectedLabel)

	refreshes := 0
	controller.refresh = func() { refreshes++ }
	controls := controller.mainScreenControls()

	controls.nextTurnBtn.OnTapped()

	assert.Equal(t, 1, encounter.TurnIndex)
	assert.False(t, encounter.Combatants[0].Active)
	assert.True(t, encounter.Combatants[1].Active)
	assert.Equal(t, 1, refreshes)
}
