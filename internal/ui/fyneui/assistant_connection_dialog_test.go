package fyneui

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

// settingsRepo keeps assistant settings in memory on top of refresherRepo.
type settingsRepo struct {
	*refresherRepo
	settings domain.AssistantConnectionSettings
	found    bool
}

func (r *settingsRepo) GetAssistantConnectionSettings(context.Context) (domain.AssistantConnectionSettings, bool, error) {
	return r.settings, r.found, nil
}

func (r *settingsRepo) SaveAssistantConnectionSettings(_ context.Context, s domain.AssistantConnectionSettings) error {
	r.settings, r.found = s, true
	return nil
}

type fakeConnection struct {
	status   appsvc.ConnectionStatus
	startErr error
	starts   int
	stops    int
	listener func(appsvc.ConnectionStatus)
	applied  []domain.AssistantConnectionSettings
}

func (f *fakeConnection) StartIfEnabled(context.Context) error { return nil }

func (f *fakeConnection) Start(context.Context) error {
	f.starts++
	if f.startErr != nil {
		f.set(appsvc.ConnectionStatus{State: appsvc.ConnectionFailed, Reason: "port 47823 is in use or unavailable"})
		return f.startErr
	}
	f.set(appsvc.ConnectionStatus{State: appsvc.ConnectionRunning, Address: "http://127.0.0.1:47823/mcp"})
	return nil
}

func (f *fakeConnection) Stop(context.Context) error {
	f.stops++
	f.set(appsvc.ConnectionStatus{State: appsvc.ConnectionStopped})
	return nil
}

func (f *fakeConnection) Status() appsvc.ConnectionStatus { return f.status }

func (f *fakeConnection) Apply(_ context.Context, s domain.AssistantConnectionSettings) error {
	f.applied = append(f.applied, s)
	return nil
}

func (f *fakeConnection) OnStatusChange(fn func(appsvc.ConnectionStatus)) { f.listener = fn }

func (f *fakeConnection) set(st appsvc.ConnectionStatus) {
	f.status = st
	if f.listener != nil {
		f.listener(st)
	}
}

func newAssistantTestController(t *testing.T, conn *fakeConnection) (*uiController, *settingsRepo) {
	t.Helper()
	test.NewTempApp(t)
	repo := &settingsRepo{refresherRepo: &refresherRepo{}}
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)
	c := newUIController(t.Context(), nil, svc, &uiState{})
	c.assistant = conn
	c.assistantBtn = widget.NewButton(assistantButtonLabel(conn.Status()), nil)
	conn.OnStatusChange(c.onAssistantStatus)
	return c, repo
}

func TestAssistantPanelToggleSavesAndStarts(t *testing.T) {
	conn := &fakeConnection{status: appsvc.ConnectionStatus{State: appsvc.ConnectionStopped}}
	c, repo := newAssistantTestController(t, conn)

	panel, err := newAssistantPanel(c.ctx, c.svc, conn, testPanelDeps(t, nil))
	require.NoError(t, err)
	require.False(t, panel.enabled.Checked, "off by default")
	require.Equal(t, "Stopped", panel.status.Text)

	panel.enabled.SetChecked(true)
	require.True(t, repo.settings.Enabled, "the choice is saved")
	require.Equal(t, 1, conn.starts)
	require.Equal(t, "Running at http://127.0.0.1:47823/mcp", panel.status.Text)
	require.Equal(t, "ASSISTANT · ON", c.assistantBtn.Text)

	panel.enabled.SetChecked(false)
	panel.stopping.Wait()
	require.False(t, repo.settings.Enabled)
	require.Equal(t, 1, conn.stops)
	require.Equal(t, "Stopped", panel.status.Text)
	require.Equal(t, "ASSISTANT · OFF", c.assistantBtn.Text)
}

func TestAssistantPanelStartFailureKeepsSwitchOn(t *testing.T) {
	conn := &fakeConnection{status: appsvc.ConnectionStatus{State: appsvc.ConnectionStopped}, startErr: errors.New("address in use")}
	c, repo := newAssistantTestController(t, conn)
	panel, err := newAssistantPanel(c.ctx, c.svc, conn, testPanelDeps(t, nil))
	require.NoError(t, err)

	panel.enabled.SetChecked(true)
	require.True(t, panel.enabled.Checked, "the switch stays on so the GM can change the port")
	require.True(t, repo.settings.Enabled)
	require.Contains(t, panel.status.Text, "Failed: port 47823 is in use")
	require.Equal(t, "ASSISTANT · ERROR", c.assistantBtn.Text)
}

func TestAssistantPanelMasksAndCopiesSecret(t *testing.T) {
	conn := &fakeConnection{status: appsvc.ConnectionStatus{State: appsvc.ConnectionStopped}}
	c, repo := newAssistantTestController(t, conn)
	var copied string
	panel, err := newAssistantPanel(c.ctx, c.svc, conn, testPanelDeps(t, func(s string) { copied = s }))
	require.NoError(t, err)
	secret := string(repo.settings.AccessSecret)
	require.NotEmpty(t, secret, "the secret is generated on first open")

	require.Equal(t, assistantSecretMask, panel.secret.Text, "masked by default")
	panel.showSecret.SetChecked(true)
	require.Equal(t, secret, panel.secret.Text)
	panel.showSecret.SetChecked(false)
	require.Equal(t, assistantSecretMask, panel.secret.Text)

	test.Tap(panel.copySecret)
	require.Equal(t, secret, copied)
}

func testPanelDeps(t *testing.T, copyText func(string)) assistantPanelDeps {
	t.Helper()
	if copyText == nil {
		copyText = func(string) {}
	}
	return assistantPanelDeps{
		copyText:   copyText,
		handleErr:  func(err error) { t.Fatalf("unexpected error: %v", err) },
		confirm:    func(_, _ string, onYes func()) { onYes() },
		executable: "/opt/fallout-tracker",
	}
}

func TestAssistantPanelPortSecretSnippetsAndActivity(t *testing.T) {
	conn := &fakeConnection{status: appsvc.ConnectionStatus{State: appsvc.ConnectionStopped}}
	c, repo := newAssistantTestController(t, conn)
	confirmed := 0
	deps := testPanelDeps(t, nil)
	deps.confirm = func(_, _ string, onYes func()) { confirmed++; onYes() }
	panel, err := newAssistantPanel(c.ctx, c.svc, conn, deps)
	require.NoError(t, err)
	first := repo.settings.AccessSecret

	panel.port.SetText("80")
	test.Tap(panel.applyPort)
	require.Contains(t, panel.portError.Text, "1024 and 65535")
	require.Empty(t, conn.applied, "an invalid port is not applied")

	panel.port.SetText("50123")
	test.Tap(panel.applyPort)
	require.Empty(t, panel.portError.Text)
	require.Equal(t, 50123, repo.settings.Port)
	require.Len(t, conn.applied, 1)
	require.Contains(t, panel.codeSnippet.Text, "http://127.0.0.1:50123/mcp")
	require.NotContains(t, panel.codeSnippet.Text, string(first), "snippets mask the secret by default")

	test.Tap(panel.regenerate)
	require.Equal(t, 1, confirmed, "regenerating asks first")
	require.NotEqual(t, first, repo.settings.AccessSecret)
	require.Equal(t, repo.settings.AccessSecret, conn.applied[len(conn.applied)-1].AccessSecret, "the connection gets the new secret")
	panel.showSecret.SetChecked(true)
	require.Contains(t, panel.codeSnippet.Text, string(repo.settings.AccessSecret))
	require.NotContains(t, panel.deskSnippet.Text, string(first), "no stale secret")
	require.Contains(t, panel.deskSnippet.Text, "/opt/fallout-tracker")
	require.Contains(t, panel.deskSnippet.Text, "mcp-bridge")

	require.Equal(t, "No assistant activity yet", panel.activity.Text)
	test.Tap(panel.clearHistory)
	require.Equal(t, 2, confirmed, "clearing asks first")
}
