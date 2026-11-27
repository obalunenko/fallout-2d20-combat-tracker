package mcpserver

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestConnectionStartIfEnabledRespectsSettings(t *testing.T) {
	svc := newTestService(t)
	conn := NewConnection(svc, app.NewConfirmationRouter(), Options{})
	t.Cleanup(func() { _ = conn.Stop(context.WithoutCancel(t.Context())) })

	require.NoError(t, conn.StartIfEnabled(t.Context()))
	require.Equal(t, app.ConnectionStopped, conn.Status().State, "off by default")

	settings, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)
	settings.Enabled, settings.Port = true, freePort(t)
	require.NoError(t, svc.SaveAssistantSettings(t.Context(), settings))
	require.NoError(t, conn.StartIfEnabled(t.Context()))
	st := conn.Status()
	require.Equal(t, app.ConnectionRunning, st.State)
	require.Equal(t, "http://127.0.0.1:"+strconv.Itoa(settings.Port)+"/mcp", st.Address)
}

func TestConnectionPortInUseFailsWithoutBreakingTheService(t *testing.T) {
	svc := newTestService(t)
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })
	port := busy.Addr().(*net.TCPAddr).Port

	conn := NewConnection(svc, app.NewConfirmationRouter(), Options{})
	var states []app.ConnectionState
	conn.OnStatusChange(func(s app.ConnectionStatus) { states = append(states, s.State) })
	err = conn.start(domain.AssistantConnectionSettings{Port: port, AccessSecret: testSecret})
	require.Error(t, err)
	st := conn.Status()
	require.Equal(t, app.ConnectionFailed, st.State)
	require.Contains(t, st.Reason, strconv.Itoa(port))
	require.Equal(t, []app.ConnectionState{app.ConnectionFailed}, states)

	_, err = svc.ListCampaigns(t.Context())
	require.NoError(t, err, "the tracker keeps working")
}

func TestConnectionBindsLoopbackAndMovesOnPortChange(t *testing.T) {
	env := newTestEnv(t)
	var mu sync.Mutex
	var states []app.ConnectionState
	env.conn.OnStatusChange(func(s app.ConnectionStatus) {
		mu.Lock()
		defer mu.Unlock()
		states = append(states, s.State)
	})
	require.True(t, strings.HasPrefix(env.url, "http://127.0.0.1:"), "listener is loopback only")

	newPort := freePort(t)
	require.NoError(t, env.conn.Apply(t.Context(), domain.AssistantConnectionSettings{Port: newPort, AccessSecret: testSecret}))
	st := env.conn.Status()
	require.Equal(t, "http://127.0.0.1:"+strconv.Itoa(newPort)+"/mcp", st.Address, "still loopback after Apply")
	_, err := net.DialTimeout("tcp", strings.TrimSuffix(strings.TrimPrefix(env.url, "http://"), "/mcp"), time.Second)
	require.Error(t, err, "the old port no longer listens")

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []app.ConnectionState{app.ConnectionStopped, app.ConnectionRunning}, states)
}

func TestConnectionStopLetsInFlightCallsFinish(t *testing.T) {
	env := newTestEnv(t)
	release := make(chan struct{})
	started := make(chan struct{})
	mcp.AddTool(env.conn.server, &mcp.Tool{Name: "slow"}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct{}, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
		}
		return nil, struct{}{}, nil
	})
	cs := env.client(t, clientOptions{})

	done := make(chan error, 1)
	go func() {
		_, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "slow"})
		done <- err
	}()
	<-started
	stopped := make(chan error, 1)
	go func() { stopped <- env.conn.Stop(context.Background()) }()
	time.Sleep(100 * time.Millisecond)
	close(release)

	require.NoError(t, <-done, "the in-flight call finishes")
	require.NoError(t, <-stopped)
	require.Equal(t, app.ConnectionStopped, env.conn.Status().State)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	_, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "list_campaigns"})
	require.Error(t, err, "later calls are refused")
}

// failedConnection returns a connection whose start failed on a busy port.
func failedConnection(t *testing.T) (*Connection, domain.AssistantConnectionSettings) {
	t.Helper()
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = busy.Close() })
	conn := NewConnection(newTestService(t), app.NewConfirmationRouter(), Options{})
	t.Cleanup(func() { _ = conn.Stop(context.WithoutCancel(t.Context())) })
	settings := domain.AssistantConnectionSettings{Enabled: true, Port: busy.Addr().(*net.TCPAddr).Port, AccessSecret: testSecret}
	require.Error(t, conn.start(settings))
	require.Equal(t, app.ConnectionFailed, conn.Status().State)
	return conn, settings
}

func TestConnectionApplyRecoversFromFailedStart(t *testing.T) {
	conn, settings := failedConnection(t)
	settings.Port = freePort(t)

	off := settings
	off.Enabled = false
	require.NoError(t, conn.Apply(t.Context(), off))
	require.Equal(t, app.ConnectionFailed, conn.Status().State, "a disabled connection does not start")

	require.NoError(t, conn.Apply(t.Context(), settings), "choosing a free port starts the endpoint")
	st := conn.Status()
	require.Equal(t, app.ConnectionRunning, st.State)
	require.Equal(t, "http://127.0.0.1:"+strconv.Itoa(settings.Port)+"/mcp", st.Address)
}

func TestConnectionStopAfterFailedStartClearsFailure(t *testing.T) {
	conn, _ := failedConnection(t)
	var states []app.ConnectionState
	conn.OnStatusChange(func(s app.ConnectionStatus) { states = append(states, s.State) })

	require.NoError(t, conn.Stop(t.Context()))
	require.Equal(t, app.ConnectionStatus{State: app.ConnectionStopped}, conn.Status())
	require.Equal(t, []app.ConnectionState{app.ConnectionStopped}, states)

	require.NoError(t, conn.Stop(t.Context()))
	require.Len(t, states, 1, "stopping a stopped connection reports nothing")
}

func TestConnectionStopDeclinesPendingDesktopConfirmation(t *testing.T) {
	env := newTestEnv(t)
	asked := make(chan struct{})
	env.confirmer.mu.Lock()
	env.confirmer.decide = func(ctx context.Context, _ app.ConfirmationRequest) app.ConfirmationDecision {
		close(asked)
		<-ctx.Done()
		return app.ConfirmationAccepted // the GM answers after the stop
	}
	env.confirmer.mu.Unlock()
	cs := env.client(t, clientOptions{})

	go func() {
		_, _ = cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "remove_campaign_player", Arguments: dogmeatArgs})
	}()
	<-asked
	began := time.Now()
	require.NoError(t, env.conn.Stop(context.Background()))
	require.Less(t, time.Since(began), shutdownTimeout, "stopping does not wait for the GM")

	require.Len(t, mustPlayers(t, env.svc), 2, "the confirmed removal never ran")
	var outcome domain.AssistantOutcome
	for _, e := range activityOutcomes(t, env.svc) {
		if e.Operation == "remove_campaign_player" {
			outcome = e.Outcome
		}
	}
	require.Equal(t, domain.OutcomeDeclined, outcome)

	// After a restart, confirmations wait for the GM again.
	env.confirmer.mu.Lock()
	env.confirmer.decide = nil
	env.confirmer.mu.Unlock()
	require.NoError(t, env.conn.start(domain.AssistantConnectionSettings{Port: 0, AccessSecret: testSecret}))
	env.url = env.conn.Status().Address
	structured[playerRemoved](t, callTool(t, env.client(t, clientOptions{}), "remove_campaign_player", dogmeatArgs))
}
