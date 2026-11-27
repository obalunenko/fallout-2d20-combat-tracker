package fyneui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	appsvc "github.com/obalunenko/fallout/internal/app"
)

const assistantStopTimeout = 5 * time.Second

// Options wires optional collaborators into the UI.
type Options struct {
	// Assistant is the assistant (MCP) connection. Nil hides the ASSISTANT
	// controls.
	Assistant appsvc.AssistantConnection
	// Confirmations receives the desktop confirmer for assistant operations
	// the client cannot confirm in conversation.
	Confirmations *appsvc.ConfirmationRouter
}

// Run opens the tracker window and blocks until the app quits. On shutdown it
// stops the assistant connection, when one is wired, and then calls
// onShutdown.
func Run(ctx context.Context, svc *appsvc.Service, onShutdown func(), opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	a := fyneapp.New()
	a.Settings().SetTheme(newPipBoyTheme())
	shutdown := shutdownOnce(func() {
		if opts.Assistant != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), assistantStopTimeout)
			_ = opts.Assistant.Stop(stopCtx)
			cancel()
		}
		if onShutdown != nil {
			onShutdown()
		}
	})
	a.Lifecycle().SetOnStopped(shutdown)
	stopSignals := installSignalShutdown(a, shutdown)
	defer stopSignals()

	w := a.NewWindow("Fallout 2d20 Combat Tracker")
	w.Resize(fyne.NewSize(1100, 700))
	if opts.Confirmations != nil {
		opts.Confirmations.SetDesktop(newDesktopConfirmer(w))
	}

	state := &uiState{}
	controller := newUIController(ctx, w, svc, state)
	controller.assistant = opts.Assistant
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

	// Assistant changes redraw the open views live (FR-021).
	unsubscribe := svc.SubscribeChanges(newAssistantRefresher(controller.redrawAfterAssistant).onChange)
	defer unsubscribe()

	if opts.Assistant != nil {
		controller.assistantBtn = screen.assistantBtn
		opts.Assistant.OnStatusChange(controller.onAssistantStatus)
		// A failed start (for example a busy port) is shown on the ASSISTANT
		// button and in its dialog; it never stops the app (FR-006).
		_ = opts.Assistant.StartIfEnabled(ctx)
		controller.onAssistantStatus(opts.Assistant.Status())
	}

	w.SetContent(newPipBackground(screen.content))
	w.ShowAndRun()
	return nil
}
