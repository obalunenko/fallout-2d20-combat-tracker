package fyneui

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
)

const (
	assistantConfirmTitle = "Assistant wants to…"
	// assistantConfirmTimeout is how long one desktop confirmation waits,
	// measured from when it is queued (contracts/desktop-ui.md).
	assistantConfirmTimeout = 50 * time.Second
)

// desktopConfirmer shows assistant confirmations in the tracker window, one
// dialog at a time, first in first out. It implements app.DesktopConfirmer.
type desktopConfirmer struct {
	window  fyne.Window
	timeout time.Duration
	now     func() time.Time
	tick    time.Duration // countdown refresh interval
	do      func(func())  // runs UI work on the UI goroutine

	mu      sync.Mutex
	queue   []*pendingConfirm // waiting to be shown, oldest first
	current *pendingConfirm   // shown, or about to be shown
}

var _ appsvc.DesktopConfirmer = (*desktopConfirmer)(nil)

// pendingConfirm is one queued or shown confirmation.
type pendingConfirm struct {
	req      appsvc.ConfirmationRequest
	deadline time.Time
	result   chan appsvc.ConfirmationDecision // receives the one decision
	settled  bool
	view     *confirmView // set once shown
}

type confirmView struct {
	dialog    *dialog.ConfirmDialog
	countdown *widget.Label
	waiting   *widget.Label
	stop      chan struct{} // stops the countdown
}

func newDesktopConfirmer(w fyne.Window) *desktopConfirmer {
	return &desktopConfirmer{window: w, timeout: assistantConfirmTimeout, now: time.Now, tick: time.Second, do: fyne.Do}
}

// Confirm queues req and waits for the GM's answer. A request whose deadline
// passes, shown or still queued, is timed out; a cancelled ctx declines it.
func (c *desktopConfirmer) Confirm(ctx context.Context, req appsvc.ConfirmationRequest) (appsvc.ConfirmationDecision, error) {
	p := &pendingConfirm{req: req, result: make(chan appsvc.ConfirmationDecision, 1)}
	now := c.now()
	p.deadline = now.Add(c.timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(p.deadline) {
		p.deadline = dl
	}
	c.mu.Lock()
	c.queue = append(c.queue, p)
	c.mu.Unlock()
	c.advance()

	timer := time.NewTimer(p.deadline.Sub(now))
	defer timer.Stop()
	select {
	case d := <-p.result:
		return d, nil
	case <-timer.C:
		c.finish(p, appsvc.ConfirmationTimedOut, false)
	case <-ctx.Done():
		d := appsvc.ConfirmationDeclined
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			d = appsvc.ConfirmationTimedOut
		}
		c.finish(p, d, false)
	}
	return <-p.result, nil
}

// finish settles p with d unless it is settled already, takes it off the
// screen or out of the queue, and shows the next request. fromDialog is set
// when the dialog itself answered and is already closing.
func (c *desktopConfirmer) finish(p *pendingConfirm, d appsvc.ConfirmationDecision, fromDialog bool) {
	c.mu.Lock()
	if p.settled {
		c.mu.Unlock()
		return
	}
	p.settled = true
	p.result <- d
	var view *confirmView
	if c.current == p {
		c.current, view = nil, p.view
	} else {
		c.queue = slices.DeleteFunc(c.queue, func(q *pendingConfirm) bool { return q == p })
	}
	c.mu.Unlock()

	if view != nil {
		close(view.stop)
		if !fromDialog {
			c.do(view.dialog.Hide)
		}
	}
	c.advance()
}

// advance makes the oldest live request current when nothing is shown,
// timing out requests that expired while queued, and renders the current
// request.
func (c *desktopConfirmer) advance() {
	c.mu.Lock()
	now := c.now()
	for c.current == nil && len(c.queue) > 0 {
		p := c.queue[0]
		c.queue = c.queue[1:]
		if now.Before(p.deadline) {
			c.current = p
			break
		}
		p.settled = true
		p.result <- appsvc.ConfirmationTimedOut
	}
	p := c.current
	c.mu.Unlock()
	if p != nil {
		c.do(func() { c.render(p) })
	}
}

// render shows p's dialog, or refreshes its waiting line when it is shown
// already. It runs on the UI goroutine.
func (c *desktopConfirmer) render(p *pendingConfirm) {
	c.mu.Lock()
	if c.current != p {
		c.mu.Unlock() // settled before it could be shown
		return
	}
	waiting := len(c.queue)
	v, isNew := p.view, p.view == nil
	if isNew {
		v = c.newView(p)
		p.view = v
	}
	left := p.deadline.Sub(c.now())
	c.mu.Unlock()

	v.waiting.SetText(waitingText(waiting))
	if !isNew {
		return
	}
	v.countdown.SetText(countdownText(left))
	v.dialog.Show()
	c.window.RequestFocus()
	go c.runCountdown(p, v)
}

func (c *desktopConfirmer) newView(p *pendingConfirm) *confirmView {
	summary := widget.NewLabel(p.req.Summary)
	summary.Wrapping = fyne.TextWrapWord
	v := &confirmView{countdown: widget.NewLabel(""), waiting: widget.NewLabel(""), stop: make(chan struct{})}
	v.dialog = dialog.NewCustomConfirm(assistantConfirmTitle, "Confirm", "Cancel",
		container.NewVBox(summary, v.countdown, v.waiting),
		func(ok bool) {
			d := appsvc.ConfirmationDeclined
			if ok {
				d = appsvc.ConfirmationAccepted
			}
			c.finish(p, d, true)
		}, c.window)
	v.dialog.SetConfirmImportance(widget.DangerImportance)
	v.dialog.Resize(fyne.NewSize(460, 0))
	return v
}

// runCountdown refreshes the seconds left until v is closed.
func (c *desktopConfirmer) runCountdown(p *pendingConfirm, v *confirmView) {
	ticker := time.NewTicker(c.tick)
	defer ticker.Stop()
	for {
		select {
		case <-v.stop:
			return
		case <-ticker.C:
			text := countdownText(p.deadline.Sub(c.now()))
			c.do(func() { v.countdown.SetText(text) })
		}
	}
}

func countdownText(left time.Duration) string {
	return fmt.Sprintf("Closes in %d s", int(math.Ceil(max(left, 0).Seconds())))
}

func waitingText(n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return "1 more request waiting"
	default:
		return fmt.Sprintf("%d more requests waiting", n)
	}
}
