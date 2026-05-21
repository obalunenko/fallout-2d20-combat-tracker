package fyneui

import (
	"context"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	appsvc "github.com/obalunenko/fallout/internal/app"
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

	state := &uiState{}
	controller := newUIController(ctx, w, svc, state)
	labels := newMainScreenLabels()
	encounterOrder := controller.newEncounterOrderView(labels.selectedLabel)

	screen := newMainScreen(
		encounterOrder,
		labels,
		controller.mainScreenActions(),
		controller.mainScreenControls(),
	)

	refresher := newMainViewRefresher(ctx, svc, state, screen, encounterOrder, controller.handleErr)
	controller.refresh = refresher.Refresh

	controller.refresh()

	w.SetContent(newPipBackground(screen.content))
	w.ShowAndRun()
	return nil
}
