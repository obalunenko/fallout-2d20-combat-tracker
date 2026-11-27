package fyneui

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

func TestAssistantRefresherCoalescesAssistantEvents(t *testing.T) {
	redraws := 0
	var scheduled []func()
	r := newAssistantRefresher(func() { redraws++ })
	r.afterFunc = func(d time.Duration, f func()) {
		require.Equal(t, assistantRefreshDelay, d)
		scheduled = append(scheduled, f)
	}
	r.runOnMain = func(f func()) { f() }

	for range 3 {
		r.onChange(appsvc.ChangeEvent{Source: domain.SourceAssistant})
	}
	require.Len(t, scheduled, 1, "a burst within the delay schedules one redraw")
	scheduled[0]()
	require.Equal(t, 1, redraws)

	r.onChange(appsvc.ChangeEvent{Source: domain.SourceDesktop})
	require.Len(t, scheduled, 1, "desktop changes already refresh themselves")

	r.onChange(appsvc.ChangeEvent{Source: domain.SourceAssistant})
	require.Len(t, scheduled, 2, "a later assistant change schedules again")
}

func TestRedrawAfterAssistantLeavesEditorsAlone(t *testing.T) {
	c := newUIController(t.Context(), nil, nil, &uiState{})
	refreshed, reloaded := 0, 0
	c.refresh = func() { refreshed++ }
	untrack := c.trackEncounterList(func() { reloaded++ })
	editorInput := widget.NewEntry()
	editorInput.SetText("unsaved edit")

	c.redrawAfterAssistant()
	require.Equal(t, 1, refreshed)
	require.Equal(t, 1, reloaded)
	require.Equal(t, "unsaved edit", editorInput.Text, "an open editor keeps its input")

	untrack()
	c.redrawAfterAssistant()
	require.Equal(t, 1, reloaded, "a closed list is no longer reloaded")
}
