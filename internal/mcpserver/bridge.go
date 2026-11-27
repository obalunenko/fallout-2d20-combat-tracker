package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	bridgeDefaultURL   = "http://127.0.0.1:47823/mcp"
	bridgeMaxLineBytes = 16 << 20
	// bridgeHTTPTimeout must outlast a desktop confirmation (50 s).
	bridgeHTTPTimeout = 150 * time.Second

	errCodeTrackerUnreachable = -32000
	errCodeSecretRejected     = -32001

	metaProtocolVersion = "io.modelcontextprotocol/protocolVersion"
)

const msgTrackerUnreachable = "Fallout tracker is not running or its assistant connection is off. Open the tracker and enable ASSISTANT."

type bridgeOptions struct {
	// HTTPTimeout bounds one forwarded request. Production uses
	// bridgeHTTPTimeout; tests inject a short value.
	HTTPTimeout time.Duration
}

func defaultBridgeOptions() bridgeOptions { return bridgeOptions{HTTPTimeout: bridgeHTTPTimeout} }

// RunBridge relays newline-delimited JSON-RPC between stdin/stdout and the
// tracker's loopback endpoint, for clients that only launch stdio servers
// (Claude Desktop). It needs FALLOUT_TRACKER_MCP_SECRET and reads an optional
// FALLOUT_TRACKER_MCP_URL. It opens neither the database nor the UI.
// It returns the process exit code: 0 at end of input, 2 for bad setup.
func RunBridge(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env func(string) string) int {
	return runBridge(ctx, stdin, stdout, stderr, env, defaultBridgeOptions())
}

func runBridge(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, env func(string) string, opts bridgeOptions) int {
	secret := strings.TrimSpace(env("FALLOUT_TRACKER_MCP_SECRET"))
	if secret == "" {
		_, _ = fmt.Fprintln(stderr, "mcp-bridge: FALLOUT_TRACKER_MCP_SECRET is required (copy it from the tracker's ASSISTANT dialog)")
		return 2
	}
	endpoint := strings.TrimSpace(env("FALLOUT_TRACKER_MCP_URL"))
	if endpoint == "" {
		endpoint = bridgeDefaultURL
	}
	if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		_, _ = fmt.Fprintf(stderr, "mcp-bridge: invalid FALLOUT_TRACKER_MCP_URL %q\n", endpoint)
		return 2
	}
	b := &bridge{
		endpoint: endpoint,
		secret:   secret,
		client:   &http.Client{Timeout: opts.HTTPTimeout},
		out:      stdout,
		stderr:   stderr,
		calls:    make(map[string]context.CancelFunc),
	}

	scanner := bufio.NewScanner(stdin)
	scanner.Buffer(make([]byte, 64<<10), bridgeMaxLineBytes)
	var wg sync.WaitGroup
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		msg := bytes.Clone(line)
		var m bridgeMessage
		if err := json.Unmarshal(msg, &m); err != nil {
			b.logf("ignoring a line that is not JSON-RPC: %v", err)
			continue
		}
		if m.Method == "" {
			// A response from the client: the stateless endpoint never
			// sends requests, so there is nothing to forward it to.
			continue
		}
		if m.Method == "notifications/cancelled" {
			// The stateless endpoint cannot match a cancellation sent as a
			// separate request, so the bridge aborts the call itself: the
			// tracker sees the request end and declines what it waited on.
			b.cancel(m.Params.RequestID)
			continue
		}
		if m.Method == "initialize" {
			// Later requests need the negotiated version, so the
			// handshake completes before anything else is relayed.
			b.relay(ctx, msg, m)
			continue
		}
		wg.Go(func() { b.relay(ctx, msg, m) })
	}
	wg.Wait()
	if err := scanner.Err(); err != nil {
		b.logf("reading stdin: %v", err)
	}
	return 0
}

type bridgeMessage struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params struct {
		Name      string          `json:"name,omitempty"`
		URI       string          `json:"uri,omitempty"`
		RequestID json.RawMessage `json:"requestId,omitempty"`
		Meta      map[string]any  `json:"_meta,omitempty"`
	} `json:"params"`
}

func (m bridgeMessage) isCall() bool { return len(m.ID) > 0 && string(m.ID) != "null" }

type bridge struct {
	endpoint string
	secret   string
	client   *http.Client
	stderr   io.Writer

	mu      sync.Mutex
	out     io.Writer
	version string                        // negotiated by initialize
	calls   map[string]context.CancelFunc // in-flight requests by JSON-RPC id
}

// relay forwards one message. A request can be aborted by a later
// notifications/cancelled with its id.
func (b *bridge) relay(ctx context.Context, raw []byte, m bridgeMessage) {
	if m.isCall() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		id := string(m.ID)
		b.mu.Lock()
		b.calls[id] = cancel
		b.mu.Unlock()
		defer func() {
			b.mu.Lock()
			delete(b.calls, id)
			b.mu.Unlock()
			cancel()
		}()
	}
	b.forward(ctx, raw, m)
}

func (b *bridge) cancel(id json.RawMessage) {
	b.mu.Lock()
	cancel := b.calls[string(id)]
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (b *bridge) forward(ctx context.Context, raw []byte, m bridgeMessage) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.endpoint, bytes.NewReader(raw))
	if err != nil {
		b.fail(m, errCodeTrackerUnreachable, msgTrackerUnreachable)
		return
	}
	req.Header.Set("Authorization", "Bearer "+b.secret)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if v := b.protocolVersion(m); v != "" {
		req.Header.Set("Mcp-Protocol-Version", v)
		if v >= mrtrProtocolVersion {
			req.Header.Set("Mcp-Method", m.Method)
			if name := mcpName(m); name != "" {
				req.Header.Set("Mcp-Name", name)
			}
		}
	}

	resp, err := b.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return // cancelled by the client, which expects no response
		}
		b.logf("forwarding %s: %v", m.Method, err)
		b.fail(m, errCodeTrackerUnreachable, msgTrackerUnreachable)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		b.fail(m, errCodeTrackerUnreachable, msgTrackerUnreachable)
		return
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		b.fail(m, errCodeSecretRejected, "Access secret rejected — copy the current secret from the tracker")
		return
	case resp.StatusCode == http.StatusAccepted:
		return // notification accepted
	case resp.StatusCode != http.StatusOK:
		b.logf("%s: HTTP %d: %s", m.Method, resp.StatusCode, bytes.TrimSpace(body))
		b.fail(m, errCodeTrackerUnreachable, fmt.Sprintf("tracker returned HTTP %d", resp.StatusCode))
		return
	}

	for _, msg := range responseMessages(resp.Header.Get("Content-Type"), body) {
		if m.Method == "initialize" {
			b.rememberVersion(msg)
		}
		b.write(msg)
	}
}

// protocolVersion is the per-request version for 2026-07-28 requests, which
// carry it in _meta, or the version negotiated by initialize.
func (b *bridge) protocolVersion(m bridgeMessage) string {
	if v, ok := m.Params.Meta[metaProtocolVersion].(string); ok && v != "" {
		return v
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.version
}

func mcpName(m bridgeMessage) string {
	switch m.Method {
	case "tools/call", "prompts/get":
		return m.Params.Name
	case "resources/read":
		return m.Params.URI
	}
	return ""
}

func (b *bridge) rememberVersion(msg []byte) {
	var res struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if json.Unmarshal(msg, &res) == nil && res.Result.ProtocolVersion != "" {
		b.mu.Lock()
		b.version = res.Result.ProtocolVersion
		b.mu.Unlock()
	}
}

// responseMessages extracts JSON-RPC messages from a JSON or event-stream
// body.
func responseMessages(contentType string, body []byte) [][]byte {
	mt, _, _ := mime.ParseMediaType(contentType)
	if mt != "text/event-stream" {
		var compact bytes.Buffer
		if err := json.Compact(&compact, body); err != nil || compact.Len() == 0 {
			return nil
		}
		return [][]byte{compact.Bytes()}
	}
	var out [][]byte
	for _, line := range bytes.Split(body, []byte("\n")) {
		if data, ok := bytes.CutPrefix(bytes.TrimSpace(line), []byte("data:")); ok {
			if data = bytes.TrimSpace(data); len(data) > 0 {
				out = append(out, data)
			}
		}
	}
	return out
}

func (b *bridge) fail(m bridgeMessage, code int, message string) {
	if !m.isCall() {
		return
	}
	msg, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      m.ID,
		"error":   map[string]any{"code": code, "message": message},
	})
	if err != nil {
		return
	}
	b.write(msg)
}

func (b *bridge) write(msg []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, err := b.out.Write(append(msg, '\n')); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		b.logf("writing stdout: %v", err)
	}
}

func (b *bridge) logf(format string, args ...any) {
	_, _ = fmt.Fprintf(b.stderr, "mcp-bridge: "+format+"\n", args...)
}
