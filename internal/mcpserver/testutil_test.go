package mcpserver

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite"
	"github.com/stretchr/testify/require"
)

const testSecret = domain.AccessSecret("test-secret-value")

// newEmptyService returns a temp-dir SQLite service with no campaign.
func newEmptyService(t *testing.T) *app.Service {
	t.Helper()
	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "tracker.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return app.NewServiceWithLogfAndTimeout(sqlite.NewEncounterStore(db), t.Logf, 30*time.Second)
}

// newTestService returns a temp-dir SQLite service with an active campaign
// "camp-1" whose two player characters are "pc-nora" and "pc-dog".
func newTestService(t *testing.T) *app.Service {
	t.Helper()
	svc := newEmptyService(t)
	start, err := domain.ParseCampaignStartDate("2026-11-01")
	require.NoError(t, err)
	_, err = svc.CreateCampaign(t.Context(), "camp-1", "Wasteland", start, []domain.NewCampaignPlayer{
		{PlayerName: "Ann", Character: domain.Combatant{ID: "pc-nora", Name: "Nora", Side: domain.SideParty, Level: 3, Initiative: 9, HP: 12, MaxHP: 12, Defense: 1}},
		{PlayerName: "Bob", Character: domain.Combatant{ID: "pc-dog", Name: "Dogmeat", Side: domain.SideParty, Level: 2, Initiative: 11, HP: 8, MaxHP: 8}},
	})
	require.NoError(t, err)
	return svc
}

// fakeConfirmer is a DesktopConfirmer whose answer the test chooses.
type fakeConfirmer struct {
	mu       sync.Mutex
	decide   func(ctx context.Context, req app.ConfirmationRequest) app.ConfirmationDecision
	requests []app.ConfirmationRequest
	deadline time.Duration
}

func (f *fakeConfirmer) Confirm(ctx context.Context, req app.ConfirmationRequest) (app.ConfirmationDecision, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	if dl, ok := ctx.Deadline(); ok {
		f.deadline = time.Until(dl)
	}
	decide := f.decide
	f.mu.Unlock()
	if decide == nil {
		return app.ConfirmationAccepted, nil
	}
	return decide(ctx, req), nil
}

func (f *fakeConfirmer) seen() []app.ConfirmationRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]app.ConfirmationRequest(nil), f.requests...)
}

type testEnv struct {
	svc       *app.Service
	conn      *Connection
	confirmer *fakeConfirmer
	url       string
}

// newTestEnv starts a Connection on a free loopback port over a seeded
// service.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	return newTestEnvWith(t, newTestService(t))
}

// newTestEnvWith starts a Connection on a free loopback port over svc.
func newTestEnvWith(t *testing.T, svc *app.Service) *testEnv {
	t.Helper()
	router := app.NewConfirmationRouter()
	confirmer := &fakeConfirmer{}
	router.SetDesktop(confirmer)
	conn := NewConnection(svc, router, Options{Version: "test", Logf: t.Logf})
	require.NoError(t, conn.start(domain.AssistantConnectionSettings{Port: 0, AccessSecret: testSecret}))
	t.Cleanup(func() { _ = conn.Stop(context.WithoutCancel(t.Context())) })
	return &testEnv{svc: svc, conn: conn, confirmer: confirmer, url: conn.Status().Address}
}

type bearerTransport struct {
	token string
	base  http.RoundTripper
}

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// clientOptions configures a test client.
type clientOptions struct {
	// elicit, when set, makes the client declare form elicitation and answer
	// with it.
	elicit func(*mcp.ElicitRequest) *mcp.ElicitResult
	// version pins the protocol version; empty means the latest.
	version string
	// manualRetry turns off the SDK's automatic multi-round-trip retry, so
	// the test sees input_required results and sends the retry itself.
	manualRetry bool
}

func (e *testEnv) client(t *testing.T, o clientOptions) *mcp.ClientSession {
	t.Helper()
	opts := &mcp.ClientOptions{}
	if o.elicit != nil {
		opts.ElicitationHandler = func(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return o.elicit(req), nil
		}
	}
	if o.manualRetry {
		opts.MultiRoundTrip = &mcp.MultiRoundTripOptions{Disabled: true}
	}
	c := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, opts)
	transport := &mcp.StreamableClientTransport{
		Endpoint:   e.url,
		HTTPClient: &http.Client{Transport: bearerTransport{token: string(testSecret), base: http.DefaultTransport}},
	}
	cs, err := c.Connect(t.Context(), transport, &mcp.ClientSessionOptions{ProtocolVersion: o.version})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *mcp.ClientSession, name string, args any) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err, name)
	return res
}

// structured decodes a successful result's structured content into T.
func structured[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	require.False(t, res.IsError, "unexpected tool error: %s", resultText(res))
	b, err := json.Marshal(res.StructuredContent)
	require.NoError(t, err)
	var out T
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// toolErrorOf decodes an error result's JSON error object.
func toolErrorOf(t *testing.T, res *mcp.CallToolResult) toolError {
	t.Helper()
	require.True(t, res.IsError, "expected a tool error, got: %s", resultText(res))
	var body struct {
		Error toolError `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(resultText(res)), &body), resultText(res))
	return body.Error
}

func resultText(res *mcp.CallToolResult) string {
	var parts []string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())
	return port
}

func activityOutcomes(t *testing.T, svc *app.Service) []domain.AssistantActivityEntry {
	t.Helper()
	entries, err := svc.ListAssistantActivity(t.Context(), 200)
	require.NoError(t, err)
	return entries
}
