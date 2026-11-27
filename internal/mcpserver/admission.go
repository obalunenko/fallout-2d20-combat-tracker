package mcpserver

import (
	"context"
	"crypto/subtle"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
)

// Refusal reasons recorded in the activity log.
const (
	refusedBrowserOrigin = "browser_origin"
	refusedBadHost       = "bad_host"
	refusedBadSecret     = "bad_secret"
)

// refusalWindow is how long refusals for one reason collapse into one entry.
const refusalWindow = time.Minute

// admission checks every request before it reaches the MCP handler
// (contracts/mcp-connection.md, "Request admission"): no browser origin, a
// loopback Host, and the current access secret.
type admission struct {
	port   int
	secret atomic.Pointer[string]
	// refuse records a refusal; it is called outside any request lock.
	refuse func(detail string)

	mu       sync.Mutex
	lastSeen map[string]time.Time
	skipped  map[string]int
	now      func() time.Time
	// after runs f once d has passed; it ends a refusal window.
	after func(d time.Duration, f func())
}

func newAdmission(port int, secret domain.AccessSecret, refuse func(detail string)) *admission {
	a := &admission{port: port, refuse: refuse, lastSeen: map[string]time.Time{}, skipped: map[string]int{}, now: time.Now,
		after: func(d time.Duration, f func()) { time.AfterFunc(d, f) }}
	a.setSecret(secret)
	return a
}

func (a *admission) setSecret(secret domain.AccessSecret) {
	s := string(secret)
	a.secret.Store(&s)
}

func (a *admission) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.originAllowed(r) {
			a.deny(w, http.StatusForbidden, refusedBrowserOrigin, "browser requests are not allowed")
			return
		}
		if !a.hostAllowed(r.Host) {
			a.deny(w, http.StatusForbidden, refusedBadHost, "host not allowed")
			return
		}
		if !a.tokenValid(r.Header.Get("Authorization")) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			a.deny(w, http.StatusUnauthorized, refusedBadSecret, "missing or invalid access secret")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// originAllowed refuses browser requests. Non-browser MCP clients send no
// Origin (verified for Claude Code, research R3); this endpoint's own
// loopback origin is allowed because no other page is served on its port.
func (a *admission) originAllowed(r *http.Request) bool {
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "none" && site != "same-origin" {
		return false
	}
	origin := r.Header.Get("Origin")
	return origin == "" || origin == a.ownOrigin("127.0.0.1") || origin == a.ownOrigin("localhost")
}

func (a *admission) ownOrigin(host string) string { return fmt.Sprintf("http://%s:%d", host, a.port) }

// hostAllowed blocks DNS rebinding: the Host header must name this loopback
// endpoint.
func (a *admission) hostAllowed(host string) bool {
	return host == fmt.Sprintf("127.0.0.1:%d", a.port) || host == fmt.Sprintf("localhost:%d", a.port)
}

func (a *admission) tokenValid(header string) bool {
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || token == "" {
		return false
	}
	want := a.secret.Load()
	return want != nil && subtle.ConstantTimeCompare([]byte(token), []byte(*want)) == 1
}

func (a *admission) deny(w http.ResponseWriter, status int, reason, message string) {
	a.noteRefusal(reason)
	http.Error(w, fmt.Sprintf(`{"error":%q}`, message), status)
}

// noteRefusal records at most one activity entry per reason per
// refusalWindow, and one more when the window ends that counts the refusals
// it collapsed. This keeps a hostile local process from flooding the log.
func (a *admission) noteRefusal(reason string) {
	a.mu.Lock()
	now := a.now()
	last, seen := a.lastSeen[reason]
	if seen && now.Sub(last) < refusalWindow {
		a.skipped[reason]++
		if a.skipped[reason] == 1 {
			a.after(refusalWindow-now.Sub(last), func() { a.flush(reason) })
		}
		a.mu.Unlock()
		return
	}
	skipped := a.skipped[reason]
	a.lastSeen[reason], a.skipped[reason] = now, 0
	a.mu.Unlock()
	a.record(reason, skipped)
}

// flush records the refusals collapsed so far for reason.
func (a *admission) flush(reason string) {
	a.mu.Lock()
	skipped := a.skipped[reason]
	a.skipped[reason] = 0
	a.mu.Unlock()
	if skipped > 0 {
		a.record(reason, skipped)
	}
}

// flushAll records every count still collapsed; the connection calls it when
// it stops.
func (a *admission) flushAll() {
	a.mu.Lock()
	reasons := slices.Collect(maps.Keys(a.skipped))
	a.mu.Unlock()
	for _, reason := range reasons {
		a.flush(reason)
	}
}

func (a *admission) record(reason string, skipped int) {
	detail := reason
	if skipped > 0 {
		detail = fmt.Sprintf("%s (+%d more in the previous minute)", reason, skipped)
	}
	if a.refuse != nil {
		a.refuse(detail)
	}
}

// refusalRecorder returns the admission callback that writes refusals to the
// activity log.
func (d *deps) refusalRecorder() func(detail string) {
	return func(detail string) {
		d.svc.AppendAssistantActivity(context.Background(), domain.AssistantActivityEntry{
			Operation: "connection",
			Outcome:   domain.OutcomeRefused,
			Detail:    detail,
		})
	}
}
