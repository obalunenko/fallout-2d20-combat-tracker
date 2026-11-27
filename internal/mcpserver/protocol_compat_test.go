package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// TestProtocolCompatibility checks the transport decision from research R2/R8:
// a stateless, JSON-response Streamable HTTP handler serves both the
// 2026-07-28 revision and a legacy 2025-11-25 client without sessions.
func TestProtocolCompatibility(t *testing.T) {
	var seen []string
	srv := mcp.NewServer(&mcp.Implementation{Name: "compat", Version: "test"}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "ping"}, func(_ context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, struct {
		OK bool `json:"ok"`
	}, error) {
		seen = append(seen, req.ProtocolVersion())
		return nil, struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	})
	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		t.Run(version, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			client := mcp.NewClient(&mcp.Implementation{Name: "compat-client", Version: "test"}, nil)
			cs, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL}, &mcp.ClientSessionOptions{ProtocolVersion: version})
			require.NoError(t, err)
			t.Cleanup(func() { _ = cs.Close() })

			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "ping"})
			require.NoError(t, err)
			require.False(t, res.IsError)
		})
	}
	require.Equal(t, []string{"2026-07-28", "2025-11-25"}, seen)

	resp, err := http.Get(ts.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestCaptureClientHeaders is a manual helper for task T004: it records the
// headers a real client (Claude Code) sends, so research R3 can confirm the
// Origin policy. It is skipped unless FALLOUT_CAPTURE_HEADERS=1.
func TestCaptureClientHeaders(t *testing.T) {
	if os.Getenv("FALLOUT_CAPTURE_HEADERS") != "1" {
		t.Skip("set FALLOUT_CAPTURE_HEADERS=1 to capture real client headers")
	}
	port := os.Getenv("FALLOUT_CAPTURE_PORT")
	if port == "" {
		port = "47899"
	}
	srv := mcp.NewServer(&mcp.Implementation{Name: "capture", Version: "test"}, nil)
	inner := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	logging := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys := make([]string, 0, len(r.Header))
		for k := range r.Header {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteString(r.Method + " " + r.URL.Path + " Host=" + r.Host)
		for _, k := range keys {
			v := strings.Join(r.Header.Values(k), ",")
			if strings.EqualFold(k, "Authorization") {
				v = "[redacted]"
			}
			b.WriteString(" | " + k + "=" + v)
		}
		t.Log(b.String())
		inner.ServeHTTP(w, r)
	})
	hs := &http.Server{Addr: "127.0.0.1:" + port, Handler: logging, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = hs.ListenAndServe() }()
	t.Cleanup(func() { _ = hs.Close() })
	time.Sleep(60 * time.Second)
}
