package fyneui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

const (
	assistantSecretMask   = "••••••••••••••••"
	assistantActivityRows = 200
)

// assistantButtonLabel is the header button text for a connection status.
func assistantButtonLabel(st appsvc.ConnectionStatus) string {
	switch st.State {
	case appsvc.ConnectionRunning:
		return "ASSISTANT · ON"
	case appsvc.ConnectionFailed:
		return "ASSISTANT · ERROR"
	default:
		return "ASSISTANT · OFF"
	}
}

func assistantStatusText(st appsvc.ConnectionStatus) string {
	switch st.State {
	case appsvc.ConnectionRunning:
		return "Running at " + st.Address
	case appsvc.ConnectionFailed:
		return "Failed: " + st.Reason + " — choose another port"
	default:
		return "Stopped"
	}
}

// assistantPanel is the content of the assistant connection dialog. It talks
// to the service for settings and to the connection for its lifecycle.
type assistantPanel struct {
	ctx       context.Context
	svc       *appsvc.Service
	conn      appsvc.AssistantConnection
	copyText  func(string)
	handleErr func(error)
	// confirm asks the GM before an irreversible panel action.
	confirm func(title, message string, onYes func())
	// executable is the tracker binary path used in the Claude Desktop snippet.
	executable string

	settings domain.AssistantConnectionSettings
	// stopping tracks Stop calls running off the UI goroutine.
	stopping sync.WaitGroup

	enabled      *widget.Check
	status       *widget.Label
	secret       *widget.Label
	showSecret   *widget.Check
	copySecret   *widget.Button
	port         *widget.Entry
	portError    *widget.Label
	applyPort    *widget.Button
	regenerate   *widget.Button
	codeSnippet  *widget.Label
	copyCode     *widget.Button
	deskSnippet  *widget.Label
	copyDesktop  *widget.Button
	activity     *widget.Entry
	clearHistory *widget.Button
	content      fyne.CanvasObject
}

type assistantPanelDeps struct {
	copyText   func(string)
	handleErr  func(error)
	confirm    func(title, message string, onYes func())
	executable string
}

func newAssistantPanel(ctx context.Context, svc *appsvc.Service, conn appsvc.AssistantConnection, deps assistantPanelDeps) (*assistantPanel, error) {
	settings, err := svc.GetAssistantSettings(ctx)
	if err != nil {
		return nil, err
	}
	p := &assistantPanel{
		ctx: ctx, svc: svc, conn: conn, settings: settings,
		copyText: deps.copyText, handleErr: deps.handleErr, confirm: deps.confirm, executable: deps.executable,
	}
	p.status = newMonospaceLabel("")
	p.status.Wrapping = fyne.TextWrapWord
	p.secret = newMonospaceLabel(assistantSecretMask)
	p.showSecret = widget.NewCheck("Show", func(show bool) { p.renderSecret(show) })
	p.copySecret = newRoleButton("COPY SECRET", uiActionSubtle, func() { p.copyText(string(p.settings.AccessSecret)) })
	p.enabled = widget.NewCheck("Assistant connection enabled", nil)
	p.enabled.SetChecked(settings.Enabled)
	p.enabled.OnChanged = p.setEnabled

	p.port = widget.NewEntry()
	p.port.SetText(strconv.Itoa(settings.Port))
	p.portError = newMonospaceLabel("")
	p.portError.Importance = widget.DangerImportance
	p.applyPort = newRoleButton("APPLY", uiActionSecondary, p.applyPortChange)
	p.regenerate = newRoleButton("REGENERATE SECRET", uiActionWarning, p.regenerateSecret)

	p.codeSnippet = newMonospaceLabel("")
	p.codeSnippet.Wrapping = fyne.TextWrapBreak
	p.copyCode = newRoleButton("COPY", uiActionSubtle, func() { p.copyText(claudeCodeSnippet(p.settings, false)) })
	p.deskSnippet = newMonospaceLabel("")
	p.deskSnippet.Wrapping = fyne.TextWrapBreak
	p.copyDesktop = newRoleButton("COPY", uiActionSubtle, func() { p.copyText(claudeDesktopSnippet(p.settings, p.executable, false)) })

	p.activity = newReadOnlyMonospaceOutput("", 8)
	p.clearHistory = newRoleButton("CLEAR", uiActionDestructive, func() {
		p.confirm("Clear assistant activity", "Delete every assistant activity entry?", func() {
			if err := p.svc.ClearAssistantActivity(p.ctx); err != nil {
				p.handleErr(err)
				return
			}
			p.reloadActivity()
		})
	})

	p.content = container.NewVBox(
		p.enabled,
		p.status,
		widget.NewSeparator(),
		container.NewBorder(nil, nil, newMonospaceLabel("Port (1024-65535):"), p.applyPort, p.port),
		p.portError,
		newMonospaceLabel("Access secret (paste into your assistant client):"),
		container.NewBorder(nil, nil, nil, container.NewHBox(p.showSecret, p.copySecret, p.regenerate), p.secret),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, newMonospaceLabel("Claude Code:"), p.copyCode, p.codeSnippet),
		container.NewBorder(nil, nil, newMonospaceLabel("Claude Desktop:"), p.copyDesktop, p.deskSnippet),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, newMonospaceLabel("Assistant activity (newest first):"), p.clearHistory),
		p.activity,
	)
	p.renderStatus(conn.Status())
	p.renderSecret(false)
	p.reloadActivity()
	return p, nil
}

func (p *assistantPanel) applyPortChange() {
	p.portError.SetText("")
	port, err := strconv.Atoi(strings.TrimSpace(p.port.Text))
	if err == nil {
		err = domain.ValidateAssistantPort(port)
	}
	if err != nil {
		p.portError.SetText("Port must be a number between 1024 and 65535")
		return
	}
	p.settings.Port = port
	if err := p.svc.SaveAssistantSettings(p.ctx, p.settings); err != nil {
		p.handleErr(err)
		return
	}
	// A failed restart shows on the status line; the GM can pick another port.
	_ = p.conn.Apply(p.ctx, p.settings)
	p.renderStatus(p.conn.Status())
	p.renderSecret(p.showSecret.Checked)
}

func (p *assistantPanel) regenerateSecret() {
	p.confirm("Regenerate access secret", "Clients using the current secret stop working until you paste the new one into them. Continue?", func() {
		settings, err := p.svc.RegenerateAccessSecret(p.ctx)
		if err != nil {
			p.handleErr(err)
			return
		}
		p.settings.AccessSecret = settings.AccessSecret
		// A failed restart shows on the status line, as for a port change.
		_ = p.conn.Apply(p.ctx, p.settings)
		p.renderStatus(p.conn.Status())
		p.renderSecret(p.showSecret.Checked)
	})
}

// reloadActivity shows the newest assistant activity entries.
func (p *assistantPanel) reloadActivity() {
	entries, err := p.svc.ListAssistantActivity(p.ctx, assistantActivityRows)
	if err != nil {
		p.activity.SetText("Activity unavailable: " + err.Error())
		return
	}
	if len(entries) == 0 {
		p.activity.SetText("No assistant activity yet")
		return
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		entity := e.EntityLabel
		if entity == "" {
			entity = e.EntityID
		}
		lines = append(lines, strings.Join(nonEmpty(formatTimestamp(e.OccurredAt), e.Operation, entity, string(e.Outcome), e.Detail), " · "))
	}
	p.activity.SetText(strings.Join(lines, "\n"))
}

func nonEmpty(parts ...string) []string {
	out := parts[:0]
	for _, s := range parts {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func endpointURL(port int) string { return fmt.Sprintf("http://127.0.0.1:%d/mcp", port) }

func shownSecret(s domain.AccessSecret, mask bool) string {
	if mask {
		return assistantSecretMask
	}
	return string(s)
}

// claudeCodeSnippet is the command that registers the tracker in Claude Code.
func claudeCodeSnippet(s domain.AssistantConnectionSettings, mask bool) string {
	return fmt.Sprintf(`claude mcp add --scope user --transport http fallout-tracker %s --header "Authorization: Bearer %s"`,
		endpointURL(s.Port), shownSecret(s.AccessSecret, mask))
}

// claudeDesktopSnippet is the claude_desktop_config.json entry that runs the
// stdio bridge.
func claudeDesktopSnippet(s domain.AssistantConnectionSettings, executable string, mask bool) string {
	return fmt.Sprintf(`{"mcpServers": {"fallout-tracker": {"command": %q, "args": ["mcp-bridge"], "env": {"FALLOUT_TRACKER_MCP_SECRET": %q, "FALLOUT_TRACKER_MCP_URL": %q}}}}`,
		executable, shownSecret(s.AccessSecret, mask), endpointURL(s.Port))
}

func trackerExecutable() string {
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return "fallout-tracker"
}

func (p *assistantPanel) setEnabled(on bool) {
	p.settings.Enabled = on
	if err := p.svc.SaveAssistantSettings(p.ctx, p.settings); err != nil {
		p.handleErr(err)
		return
	}
	if !on {
		// Stopping waits for requests in flight, so it runs off the UI
		// goroutine.
		p.stopping.Go(func() {
			err := p.conn.Stop(p.ctx)
			fyne.Do(func() {
				p.renderStatus(p.conn.Status())
				if err != nil {
					p.handleErr(err)
				}
			})
		})
		return
	}
	// On a failed start the switch stays on and the status shows why
	// (FR-006); the GM can change the port and retry.
	_ = p.conn.Start(p.ctx)
	p.renderStatus(p.conn.Status())
}

func (p *assistantPanel) renderStatus(st appsvc.ConnectionStatus) {
	p.status.SetText(assistantStatusText(st))
}

func (p *assistantPanel) renderSecret(show bool) {
	p.secret.SetText(shownSecret(p.settings.AccessSecret, !show))
	p.codeSnippet.SetText(claudeCodeSnippet(p.settings, !show))
	p.deskSnippet.SetText(claudeDesktopSnippet(p.settings, p.executable, !show))
}

// showAssistantDialog opens the assistant connection dialog.
func (c *uiController) showAssistantDialog() {
	if c.assistant == nil {
		return
	}
	panel, err := newAssistantPanel(c.ctx, c.svc, c.assistant, assistantPanelDeps{
		copyText:  c.copyToClipboard,
		handleErr: c.handleErr,
		confirm: func(title, message string, onYes func()) {
			dialog.ShowConfirm(title, message, func(ok bool) {
				if ok {
					onYes()
				}
			}, c.window)
		},
		executable: trackerExecutable(),
	})
	if err != nil {
		c.handleErr(err)
		return
	}
	c.assistantPanel = panel
	d := dialog.NewCustom("Assistant connection", "Close", panel.content, c.window)
	d.SetOnClosed(func() { c.assistantPanel = nil })
	d.Resize(fyne.NewSize(640, 0))
	d.Show()
}

// onAssistantStatus updates the header button and an open dialog. It may be
// called from any goroutine.
func (c *uiController) onAssistantStatus(st appsvc.ConnectionStatus) {
	fyne.Do(func() {
		if c.assistantBtn != nil {
			c.assistantBtn.SetText(assistantButtonLabel(st))
		}
		if c.assistantPanel != nil {
			c.assistantPanel.renderStatus(st)
		}
	})
}

func (c *uiController) copyToClipboard(text string) {
	if app := fyne.CurrentApp(); app != nil {
		app.Clipboard().SetContent(strings.TrimSpace(text))
	}
}
