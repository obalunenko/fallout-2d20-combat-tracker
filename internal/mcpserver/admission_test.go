package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

const initializeBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`

func rawPost(t *testing.T, endpoint string, headers map[string]string, host string) int {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, endpoint, strings.NewReader(initializeBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if host != "" {
		req.Host = host
	}
	req.Close = true // no idle keep-alive connections left behind for Stop
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestAdmissionMatrix(t *testing.T) {
	env := newTestEnv(t)
	u, err := url.Parse(env.url)
	require.NoError(t, err)
	ownOrigin := "http://" + u.Host
	valid := "Bearer " + string(testSecret)

	for _, tc := range []struct {
		name    string
		headers map[string]string
		host    string
		want    int
	}{
		{"no token", nil, "", http.StatusUnauthorized},
		{"wrong token", map[string]string{"Authorization": "Bearer nope"}, "", http.StatusUnauthorized},
		{"foreign origin", map[string]string{"Authorization": valid, "Origin": "http://evil.example"}, "", http.StatusForbidden},
		{"other loopback port origin", map[string]string{"Authorization": valid, "Origin": "http://127.0.0.1:8080"}, "", http.StatusForbidden},
		{"own loopback origin", map[string]string{"Authorization": valid, "Origin": ownOrigin}, "", http.StatusOK},
		{"cross-site fetch", map[string]string{"Authorization": valid, "Sec-Fetch-Site": "cross-site"}, "", http.StatusForbidden},
		{"same-origin fetch", map[string]string{"Authorization": valid, "Sec-Fetch-Site": "same-origin"}, "", http.StatusOK},
		{"rebinding host", map[string]string{"Authorization": valid}, "evil.example:" + u.Port(), http.StatusForbidden},
		{"valid", map[string]string{"Authorization": valid}, "", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, rawPost(t, env.url, tc.headers, tc.host))
		})
	}

	// Secret rotation: the old token stops working at once.
	require.NoError(t, env.conn.Apply(context.Background(), domain.AssistantConnectionSettings{Port: mustPort(t, env.url), AccessSecret: "rotated-secret"}))
	require.Equal(t, http.StatusUnauthorized, rawPost(t, env.url, map[string]string{"Authorization": valid}, ""))
	require.Equal(t, http.StatusOK, rawPost(t, env.url, map[string]string{"Authorization": "Bearer rotated-secret"}, ""))

	// A stopped connection refuses the TCP connection itself.
	require.NoError(t, env.conn.Stop(context.Background()))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, env.url, strings.NewReader(initializeBody))
	require.NoError(t, err)
	_, err = http.DefaultClient.Do(req)
	require.Error(t, err)
}

func TestAdmissionRefusalsAreCollapsedAndRedacted(t *testing.T) {
	env := newTestEnv(t)
	for range 5 {
		require.Equal(t, http.StatusUnauthorized, rawPost(t, env.url, map[string]string{"Authorization": "Bearer wrong"}, ""))
	}
	entries := activityOutcomes(t, env.svc)
	require.Len(t, entries, 1, "five bad-secret refusals within a minute collapse into one entry")
	require.Equal(t, domain.OutcomeRefused, entries[0].Outcome)
	require.Equal(t, "connection", entries[0].Operation)
	require.Contains(t, entries[0].Detail, refusedBadSecret)
	require.NotContains(t, fmt.Sprintf("%+v", entries), string(testSecret))
}

func mustPort(t *testing.T, endpoint string) int {
	t.Helper()
	u, err := url.Parse(endpoint)
	require.NoError(t, err)
	var port int
	_, err = fmt.Sscanf(u.Port(), "%d", &port)
	require.NoError(t, err)
	return port
}

func TestAdmissionCountsRefusalsCollapsedInTheLastWindow(t *testing.T) {
	var details []string
	a := newAdmission(1, testSecret, func(d string) { details = append(details, d) })
	now := time.Unix(0, 0)
	a.now = func() time.Time { return now }
	var windowEnd func()
	a.after = func(_ time.Duration, f func()) { windowEnd = f }

	for range 4 {
		a.noteRefusal(refusedBadSecret)
	}
	require.Equal(t, []string{refusedBadSecret}, details)
	windowEnd()
	require.Equal(t, []string{refusedBadSecret, refusedBadSecret + " (+3 more in the previous minute)"}, details, "the window's end counts what it collapsed")

	now = now.Add(2 * time.Minute)
	a.noteRefusal(refusedBadHost)
	a.noteRefusal(refusedBadHost)
	a.flushAll()
	require.Equal(t, refusedBadHost+" (+1 more in the previous minute)", details[3], "stopping counts a window still open")
	windowEnd()
	require.Len(t, details, 4, "nothing is counted twice")
}
