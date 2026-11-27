package fyneui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

type editorProbe struct {
	saved, closed, reloaded, overwrites int
	shownErr                            string
	answer                              bool
	asked, notified                     int
}

func (p *editorProbe) callbacks(entity string) editorCallbacks {
	return editorCallbacks{
		entity:  entity,
		saved:   func() { p.saved++ },
		close:   func() { p.closed++ },
		showErr: func(s string) { p.shownErr = s },
		ask: func(_, _, confirmText, dismissText string, answer func(bool)) {
			p.asked++
			if confirmText != "Overwrite" || dismissText != "Reload" {
				panic("unexpected buttons")
			}
			answer(p.answer)
		},
		notify: func(_, _ string, onClosed func()) {
			p.notified++
			onClosed()
		},
	}
}

func (p *editorProbe) stale(overwriteErr error) *staleEdit {
	return &staleEdit{
		entity:    "encounter",
		overwrite: func() error { p.overwrites++; return overwriteErr },
		reload:    func() { p.reloaded++ },
	}
}

func TestStaleEditOverwriteSavesOnlyAfterChoice(t *testing.T) {
	p := &editorProbe{answer: true}
	require.True(t, handleEditorSaveError(p.stale(nil), p.callbacks("encounter")))
	require.Equal(t, 1, p.asked)
	require.Equal(t, 1, p.overwrites)
	require.Equal(t, 1, p.saved, "the overwrite resubmits and closes the editor")
	require.Zero(t, p.reloaded)
}

func TestStaleEditReloadDiscardsAndReopens(t *testing.T) {
	p := &editorProbe{answer: false}
	require.True(t, handleEditorSaveError(p.stale(nil), p.callbacks("campaign")))
	require.Equal(t, 1, p.closed)
	require.Equal(t, 1, p.reloaded)
	require.Zero(t, p.overwrites, "nothing is saved")
	require.Zero(t, p.saved)
}

func TestStaleEditOverwriteHitByAnotherChangeAsksAgain(t *testing.T) {
	p := &editorProbe{answer: true}
	inner := p.stale(nil)
	outer := p.stale(inner)
	require.True(t, handleEditorSaveError(outer, p.callbacks("encounter")))
	require.Equal(t, 2, p.asked)
	require.Equal(t, 1, p.saved)
}

func TestDeletedEntityEditorOnlyCloses(t *testing.T) {
	p := &editorProbe{}
	require.True(t, handleEditorSaveError(domain.ErrEntityDeleted, p.callbacks("encounter")))
	require.Equal(t, 1, p.notified)
	require.Equal(t, 1, p.closed)
	require.Zero(t, p.saved, "the encounter is not recreated")
}

func TestBusyKeepsEditorOpen(t *testing.T) {
	p := &editorProbe{}
	require.True(t, handleEditorSaveError(domain.ErrTrackerBusy, p.callbacks("encounter")))
	require.Equal(t, busyMessage, p.shownErr)
	require.Zero(t, p.closed)

	require.False(t, handleEditorSaveError(errors.New("name is required"), p.callbacks("encounter")), "plain validation errors use the normal path")
}
