package fyneui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

// assistantRefreshDelay coalesces bursts of assistant changes into one redraw.
const assistantRefreshDelay = 100 * time.Millisecond

// assistantRefresher redraws the desktop after assistant changes (FR-021).
// Desktop changes are ignored: the UI already refreshes after its own actions.
type assistantRefresher struct {
	redraw    func()
	afterFunc func(time.Duration, func())
	runOnMain func(func())

	mu      sync.Mutex
	pending bool
}

func newAssistantRefresher(redraw func()) *assistantRefresher {
	return &assistantRefresher{
		redraw:    redraw,
		afterFunc: func(d time.Duration, f func()) { time.AfterFunc(d, f) },
		runOnMain: fyne.Do,
	}
}

// onChange is the app.Service change subscriber. It may run on any goroutine.
func (r *assistantRefresher) onChange(ev appsvc.ChangeEvent) {
	if ev.Source != domain.SourceAssistant {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending {
		return
	}
	r.pending = true
	r.afterFunc(assistantRefreshDelay, func() {
		r.mu.Lock()
		r.pending = false
		r.mu.Unlock()
		r.runOnMain(r.redraw)
	})
}

// redrawAfterAssistant repaints the main screen and the live views that are
// open. Editor dialogs are left alone, so unsaved input survives.
func (c *uiController) redrawAfterAssistant() {
	if c.refresh != nil {
		c.refresh()
	}
	if c.encounterListReload != nil {
		c.encounterListReload()
	}
	if c.assistantPanel != nil {
		c.assistantPanel.reloadActivity()
	}
}

// trackEncounterList registers the open encounter list for live reload.
func (c *uiController) trackEncounterList(reload func()) (untrack func()) {
	c.encounterListReload = reload
	return func() { c.encounterListReload = nil }
}
