package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// bridgeClient connects an MCP client to runBridge over in-process pipes.
func bridgeClient(t *testing.T, endpoint, secret, version string) *mcp.ClientSession {
	t.Helper()
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	done := make(chan int, 1)
	go func() {
		env := map[string]string{"FALLOUT_TRACKER_MCP_SECRET": secret, "FALLOUT_TRACKER_MCP_URL": endpoint}
		done <- runBridge(context.Background(), stdinR, stdoutW, io.Discard, func(k string) string { return env[k] }, defaultBridgeOptions())
		_ = stdoutW.Close()
	}()
	c := mcp.NewClient(&mcp.Implementation{Name: "desktop-like", Version: "test"}, nil)
	cs, err := c.Connect(t.Context(), &mcp.IOTransport{Reader: stdoutR, Writer: stdinW}, &mcp.ClientSessionOptions{ProtocolVersion: version})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = cs.Close()
		_ = stdinW.Close()
		require.Equal(t, 0, <-done, "the bridge exits 0 at end of input")
	})
	return cs
}

func TestBridgeRelaysLegacyAndModernClients(t *testing.T) {
	env := newTestEnv(t)
	var mu sync.Mutex
	var headers []http.Header
	target, err := url.Parse(env.url)
	require.NoError(t, err)
	target.Path = "" // the proxy appends the request path itself
	proxy := httputil.NewSingleHostReverseProxy(target)
	recorder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		headers = append(headers, r.Header.Clone())
		mu.Unlock()
		r.Host = target.Host // the endpoint accepts only its own loopback Host
		proxy.ServeHTTP(w, r)
	}))
	t.Cleanup(recorder.Close)

	for _, version := range []string{"2025-11-25", "2026-07-28"} {
		t.Run(version, func(t *testing.T) {
			cs := bridgeClient(t, recorder.URL+"/mcp", string(testSecret), version)
			res := callTool(t, cs, "list_campaigns", nil)
			require.Len(t, structured[campaignList](t, res).Campaigns, 1)
		})
	}

	mu.Lock()
	defer mu.Unlock()
	var sawCall bool
	for _, h := range headers {
		require.Equal(t, "Bearer "+string(testSecret), h.Get("Authorization"), "the secret is forwarded")
		if h.Get("Mcp-Protocol-Version") >= "2026-07-28" && h.Get("Mcp-Method") == "tools/call" {
			require.Equal(t, "list_campaigns", h.Get("Mcp-Name"))
			sawCall = true
		}
	}
	require.True(t, sawCall, "a 2026-07-28 tools/call carried Mcp-Method and Mcp-Name")
}

// rawBridge feeds raw JSON-RPC lines to the bridge and returns its output
// lines.
func rawBridge(t *testing.T, endpoint, secret string, opts bridgeOptions, lines ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	env := map[string]string{"FALLOUT_TRACKER_MCP_SECRET": secret, "FALLOUT_TRACKER_MCP_URL": endpoint}
	code := runBridge(t.Context(), strings.NewReader(strings.Join(lines, "\n")+"\n"), &out, io.Discard, func(k string) string { return env[k] }, opts)
	require.Equal(t, 0, code)
	var msgs []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(l), &m), l)
		msgs = append(msgs, m)
	}
	return msgs
}

func errorCode(t *testing.T, msg map[string]any) float64 {
	t.Helper()
	e, ok := msg["error"].(map[string]any)
	require.True(t, ok, "expected an error, got %v", msg)
	return e["code"].(float64)
}

func TestBridgeReportsUnreachableTrackerAndBadSecret(t *testing.T) {
	ping := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	note := `{"jsonrpc":"2.0","method":"notifications/initialized"}`

	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL + "/mcp"
	closed.Close()
	msgs := rawBridge(t, closedURL, "x", defaultBridgeOptions(), ping, note)
	require.Len(t, msgs, 1, "notifications produce no output")
	require.EqualValues(t, errCodeTrackerUnreachable, errorCode(t, msgs[0]))
	require.Contains(t, msgs[0]["error"].(map[string]any)["message"], "not running")

	env := newTestEnv(t)
	msgs = rawBridge(t, env.url, "wrong-secret", defaultBridgeOptions(), ping)
	require.EqualValues(t, errCodeSecretRejected, errorCode(t, msgs[0]))
}

func TestBridgeTimeoutAndSetup(t *testing.T) {
	require.Equal(t, 150*time.Second, defaultBridgeOptions().HTTPTimeout, "outlasts a 50 s desktop confirmation")

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method == "slow" {
			time.Sleep(400 * time.Millisecond)
		} else {
			time.Sleep(200 * time.Millisecond)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{}}`))
	}))
	t.Cleanup(slow.Close)
	opts := bridgeOptions{HTTPTimeout: 300 * time.Millisecond}

	msgs := rawBridge(t, slow.URL, "x", opts, `{"jsonrpc":"2.0","id":1,"method":"quick","params":{}}`)
	require.Contains(t, msgs[0], "result", "a 200 ms call fits a 300 ms timeout")
	msgs = rawBridge(t, slow.URL, "x", opts, `{"jsonrpc":"2.0","id":2,"method":"slow","params":{}}`)
	require.EqualValues(t, errCodeTrackerUnreachable, errorCode(t, msgs[0]))

	var stderr bytes.Buffer
	code := runBridge(t.Context(), strings.NewReader(""), io.Discard, &stderr, func(string) string { return "" }, defaultBridgeOptions())
	require.Equal(t, 2, code)
	require.Contains(t, stderr.String(), "FALLOUT_TRACKER_MCP_SECRET")
}

func TestBridgeClientCancelDeclinesDesktopConfirmation(t *testing.T) {
	env := newTestEnv(t)
	asked := make(chan struct{})
	gone := make(chan struct{})
	env.confirmer.mu.Lock()
	env.confirmer.decide = func(ctx context.Context, _ app.ConfirmationRequest) app.ConfirmationDecision {
		close(asked)
		select {
		case <-ctx.Done():
			close(gone)
		case <-time.After(10 * time.Second):
		}
		return app.ConfirmationAccepted
	}
	env.confirmer.mu.Unlock()
	cs := bridgeClient(t, env.url, string(testSecret), "2025-11-25")

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		_, _ = cs.CallTool(ctx, &mcp.CallToolParams{Name: "remove_campaign_player", Arguments: dogmeatArgs})
	}()
	<-asked
	cancel()
	select {
	case <-gone:
	case <-time.After(5 * time.Second):
		t.Fatal("the desktop confirmation kept waiting after the client cancelled")
	}

	require.Eventually(t, func() bool {
		for _, e := range activityOutcomes(t, env.svc) {
			if e.Operation == "remove_campaign_player" {
				return e.Outcome == domain.OutcomeDeclined
			}
		}
		return false
	}, 5*time.Second, 20*time.Millisecond, "the cancelled call is logged as declined")
	require.Len(t, mustPlayers(t, env.svc), 2, "nothing was removed")
}
