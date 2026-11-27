package fyneui

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
)

// uiLoop runs queued UI work on one goroutine, as the Fyne main loop does.
type uiLoop chan func()

func newUILoop(t *testing.T) uiLoop {
	t.Helper()
	loop := make(uiLoop, 64)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for f := range loop {
			f()
		}
	}()
	t.Cleanup(func() {
		close(loop)
		<-stopped
	})
	return loop
}

// do queues f, like fyne.Do.
func (l uiLoop) do(f func()) { l <- f }

// run runs f on the UI goroutine and waits for it.
func (l uiLoop) run(f func()) {
	done := make(chan struct{})
	l <- func() {
		defer close(done)
		f()
	}
	<-done
}

type confirmerHarness struct {
	*desktopConfirmer
	ui uiLoop
	w  fyne.Window
}

func newTestConfirmer(t *testing.T) *confirmerHarness {
	t.Helper()
	test.NewTempApp(t)
	ui := newUILoop(t)
	var w fyne.Window
	ui.run(func() {
		w = test.NewWindow(widget.NewLabel("tracker"))
		w.Resize(fyne.NewSize(800, 600))
	})
	t.Cleanup(func() { ui.run(w.Close) })
	c := newDesktopConfirmer(w)
	c.do = ui.do
	return &confirmerHarness{desktopConfirmer: c, ui: ui, w: w}
}

type confirmOutcome struct {
	decision appsvc.ConfirmationDecision
	err      error
}

// confirmAsync runs Confirm in the background and returns its outcome.
func (h *confirmerHarness) confirmAsync(ctx context.Context, summary string) <-chan confirmOutcome {
	out := make(chan confirmOutcome, 1)
	go func() {
		d, err := h.Confirm(ctx, appsvc.ConfirmationRequest{Operation: "op", Summary: summary})
		out <- confirmOutcome{d, err}
	}()
	return out
}

// screen is what the window shows: the open dialog's summary, countdown
// and waiting line. summary is empty when no dialog is open.
type screen struct {
	summary, countdown, waiting string
}

func (h *confirmerHarness) screen() screen {
	var s screen
	h.ui.run(func() {
		if h.w.Canvas().Overlays().Top() == nil {
			return
		}
		h.mu.Lock()
		p := h.current
		h.mu.Unlock()
		if p == nil || p.view == nil {
			return
		}
		s = screen{summary: p.req.Summary, countdown: p.view.countdown.Text, waiting: p.view.waiting.Text}
	})
	return s
}

func (h *confirmerHarness) queued() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.queue)
}

func (h *confirmerHarness) waitScreen(t *testing.T, want func(screen) bool, msg string) screen {
	t.Helper()
	var s screen
	require.Eventually(t, func() bool {
		s = h.screen()
		return want(s)
	}, 5*time.Second, 5*time.Millisecond, msg)
	return s
}

func (h *confirmerHarness) waitShown(t *testing.T, summary string) screen {
	t.Helper()
	return h.waitScreen(t, func(s screen) bool { return s.summary == summary }, "dialog not shown: "+summary)
}

func (h *confirmerHarness) waitQueued(t *testing.T, n int) {
	t.Helper()
	require.Eventually(t, func() bool { return h.queued() == n }, 5*time.Second, 5*time.Millisecond)
}

// dialogButton returns the button labelled text in the open dialog.
func (h *confirmerHarness) dialogButton(t *testing.T, text string) *widget.Button {
	t.Helper()
	var found *widget.Button
	h.ui.run(func() {
		for _, o := range test.LaidOutObjects(h.w.Canvas().Overlays().Top()) {
			if b, ok := o.(*widget.Button); ok && b.Text == text {
				found = b
			}
		}
	})
	require.NotNil(t, found, "no %q button in the dialog", text)
	return found
}

func (h *confirmerHarness) tap(t *testing.T, text string) {
	t.Helper()
	b := h.dialogButton(t, text)
	h.ui.run(func() { test.Tap(b) })
}

func outcomeOf(t *testing.T, ch <-chan confirmOutcome) appsvc.ConfirmationDecision {
	t.Helper()
	select {
	case o := <-ch:
		require.NoError(t, o.err)
		return o.decision
	case <-time.After(5 * time.Second):
		t.Fatal("Confirm did not return")
		return ""
	}
}

func TestDesktopConfirmerShowsRequestsInOrder(t *testing.T) {
	h := newTestConfirmer(t)
	first := h.confirmAsync(t.Context(), "Remove Bob (Dogmeat) from campaign «Wasteland»?")
	s := h.waitShown(t, "Remove Bob (Dogmeat) from campaign «Wasteland»?")
	require.Equal(t, "Closes in 50 s", s.countdown)
	require.Empty(t, s.waiting)

	second := h.confirmAsync(t.Context(), "Mark Nora unavailable?")
	h.waitQueued(t, 1)
	s = h.waitScreen(t, func(s screen) bool { return s.waiting == "1 more request waiting" }, "waiting line")
	require.Equal(t, "Remove Bob (Dogmeat) from campaign «Wasteland»?", s.summary, "one dialog at a time")

	h.tap(t, "Confirm")
	require.Equal(t, appsvc.ConfirmationAccepted, outcomeOf(t, first))
	s = h.waitShown(t, "Mark Nora unavailable?")
	require.Empty(t, s.waiting)

	h.tap(t, "Cancel")
	require.Equal(t, appsvc.ConfirmationDeclined, outcomeOf(t, second))
	require.Equal(t, screen{}, h.screen(), "no dialog is left open")
}

func TestDesktopConfirmerConfirmButtonIsDangerous(t *testing.T) {
	h := newTestConfirmer(t)
	done := h.confirmAsync(t.Context(), "Delete encounter «Ambush»?")
	h.waitShown(t, "Delete encounter «Ambush»?")
	require.Equal(t, widget.DangerImportance, h.dialogButton(t, "Confirm").Importance)
	h.tap(t, "Confirm")
	require.Equal(t, appsvc.ConfirmationAccepted, outcomeOf(t, done))
}

func TestDesktopConfirmerCountsDown(t *testing.T) {
	h := newTestConfirmer(t)
	start := time.Now()
	var elapsed atomic.Int64
	h.now = func() time.Time { return start.Add(time.Duration(elapsed.Load())) }
	h.tick = 5 * time.Millisecond
	done := h.confirmAsync(t.Context(), "Restart encounter «Ambush»?")
	require.Equal(t, "Closes in 50 s", h.waitShown(t, "Restart encounter «Ambush»?").countdown)

	elapsed.Store(int64(8 * time.Second))
	h.waitScreen(t, func(s screen) bool { return s.countdown == "Closes in 42 s" }, "countdown")
	h.tap(t, "Cancel")
	require.Equal(t, appsvc.ConfirmationDeclined, outcomeOf(t, done))
}

func TestDesktopConfirmerTimesOut(t *testing.T) {
	h := newTestConfirmer(t)
	h.timeout = 50 * time.Millisecond
	require.Equal(t, appsvc.ConfirmationTimedOut, outcomeOf(t, h.confirmAsync(t.Context(), "Restart encounter «Ambush»?")))
	h.waitScreen(t, func(s screen) bool { return s == screen{} }, "the dialog is closed")
}

func TestDesktopConfirmerDropsRequestThatExpiresWhileQueued(t *testing.T) {
	h := newTestConfirmer(t)
	first := h.confirmAsync(t.Context(), "first")
	h.waitShown(t, "first")

	short, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	second := h.confirmAsync(short, "second")
	require.Equal(t, appsvc.ConfirmationTimedOut, outcomeOf(t, second))
	require.Zero(t, h.queued())
	s := h.waitScreen(t, func(s screen) bool { return s.waiting == "" }, "waiting line cleared")
	require.Equal(t, "first", s.summary, "the open dialog stays")

	h.tap(t, "Confirm")
	require.Equal(t, appsvc.ConfirmationAccepted, outcomeOf(t, first))
	require.Equal(t, screen{}, h.screen(), "the expired request is never shown")
}

func TestDesktopConfirmerCancelDeclines(t *testing.T) {
	h := newTestConfirmer(t)
	shownCtx, cancelShown := context.WithCancel(t.Context())
	defer cancelShown()
	first := h.confirmAsync(shownCtx, "first")
	h.waitShown(t, "first")
	queuedCtx, cancelQueued := context.WithCancel(t.Context())
	defer cancelQueued()
	second := h.confirmAsync(queuedCtx, "second")
	h.waitQueued(t, 1)
	third := h.confirmAsync(t.Context(), "third")
	h.waitQueued(t, 2)

	cancelQueued()
	require.Equal(t, appsvc.ConfirmationDeclined, outcomeOf(t, second), "a cancelled queued request is removed")
	s := h.waitScreen(t, func(s screen) bool { return s.waiting == "1 more request waiting" }, "waiting line")
	require.Equal(t, "first", s.summary)

	cancelShown()
	require.Equal(t, appsvc.ConfirmationDeclined, outcomeOf(t, first), "a cancelled open request closes its dialog")
	h.waitShown(t, "third")

	h.tap(t, "Confirm")
	require.Equal(t, appsvc.ConfirmationAccepted, outcomeOf(t, third))
}

func TestConfirmDialogTexts(t *testing.T) {
	require.Equal(t, "Closes in 50 s", countdownText(50*time.Second))
	require.Equal(t, "Closes in 1 s", countdownText(200*time.Millisecond))
	require.Equal(t, "Closes in 0 s", countdownText(-time.Second))
	require.Empty(t, waitingText(0))
	require.Equal(t, "1 more request waiting", waitingText(1))
	require.Equal(t, "2 more requests waiting", waitingText(2))
}
