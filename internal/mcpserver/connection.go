package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

const (
	endpointPath      = "/mcp"
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 10 * time.Second
)

// Options configures a Connection.
type Options struct {
	// Version is the tracker build version reported to clients.
	Version string
	// Logf receives internal errors. Nil discards them.
	Logf func(string, ...any)
}

// Connection is the embedded assistant endpoint. It implements
// app.AssistantConnection.
type Connection struct {
	deps   *deps
	server *mcp.Server

	mu        sync.Mutex
	status    app.ConnectionStatus
	httpSrv   *http.Server
	admission *admission
	listeners []func(app.ConnectionStatus)
}

var _ app.AssistantConnection = (*Connection)(nil)

// NewConnection builds a stopped connection. Confirmations that the client
// cannot show in conversation go to router.
func NewConnection(svc *app.Service, router *app.ConfirmationRouter, opts Options) *Connection {
	logf := opts.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	version := opts.Version
	if version == "" {
		version = "dev"
	}
	d := &deps{svc: svc, confirm: newConfirmer(router), logf: logf}
	server, err := newServer(d, version)
	if err != nil {
		// The skills are embedded at build time; tests catch a bad file.
		panic(fmt.Sprintf("mcpserver: invalid embedded skills: %v", err))
	}
	return &Connection{
		deps:   d,
		server: server,
		status: app.ConnectionStatus{State: app.ConnectionStopped},
	}
}

// StartIfEnabled starts the connection when the stored settings enable it.
func (c *Connection) StartIfEnabled(ctx context.Context) error {
	settings, err := c.deps.svc.GetAssistantSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return nil
	}
	return c.start(settings)
}

// Start starts the connection with the stored settings.
func (c *Connection) Start(ctx context.Context) error {
	settings, err := c.deps.svc.GetAssistantSettings(ctx)
	if err != nil {
		return err
	}
	return c.start(settings)
}

func (c *Connection) start(settings domain.AssistantConnectionSettings) error {
	c.mu.Lock()
	if c.httpSrv != nil {
		c.mu.Unlock()
		return nil
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(settings.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		st := app.ConnectionStatus{State: app.ConnectionFailed, Reason: fmt.Sprintf("port %d is in use or unavailable", settings.Port)}
		c.status = st
		c.mu.Unlock()
		c.notify(st)
		return fmt.Errorf("start assistant connection: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	c.deps.confirm.resume()
	c.admission = newAdmission(port, settings.AccessSecret, c.deps.refusalRecorder())
	mux := http.NewServeMux()
	mux.Handle(endpointPath, c.admission.wrap(newHandler(c.server)))
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: readHeaderTimeout}
	c.httpSrv = srv
	st := app.ConnectionStatus{State: app.ConnectionRunning, Address: fmt.Sprintf("http://127.0.0.1:%d%s", port, endpointPath)}
	c.status = st
	c.mu.Unlock()
	c.notify(st)

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.deps.logf("assistant connection stopped: %v", err)
		}
	}()
	return nil
}

// Stop shuts the endpoint down. Desktop confirmations still open close as
// declined, so their operations never run. Other requests in flight get up to
// 5 seconds to finish; after that the remaining connections are closed.
// Later requests are refused because nothing listens.
func (c *Connection) Stop(ctx context.Context) error {
	c.mu.Lock()
	srv, adm := c.httpSrv, c.admission
	c.httpSrv, c.admission = nil, nil
	failed := c.status.State == app.ConnectionFailed
	c.mu.Unlock()
	if srv == nil {
		// Nothing listens. After a failed start the status still says why;
		// stopping clears it.
		if failed {
			c.setStatus(app.ConnectionStatus{State: app.ConnectionStopped})
		}
		return nil
	}
	c.deps.confirm.cancelPending()
	ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		// Requests (or connections that never sent one) outlived the grace
		// period: close them. Stopping still succeeded.
		c.deps.logf("assistant connection: forcing shutdown: %v", err)
		_ = srv.Close()
	}
	adm.flushAll()
	c.setStatus(app.ConnectionStatus{State: app.ConnectionStopped})
	return nil
}

// Status returns the current status.
func (c *Connection) Status() app.ConnectionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// Apply makes new settings take effect. A running endpoint swaps its secret
// in place and restarts when the port changed. An enabled endpoint whose last
// start failed tries again, so choosing a free port recovers (FR-006). A
// stopped one stays stopped.
func (c *Connection) Apply(ctx context.Context, settings domain.AssistantConnectionSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	c.mu.Lock()
	running := c.httpSrv != nil
	retry := !running && settings.Enabled && c.status.State == app.ConnectionFailed
	samePort := running && c.admission != nil && c.admission.port == settings.Port
	if samePort {
		c.admission.setSecret(settings.AccessSecret)
	}
	c.mu.Unlock()
	if retry {
		return c.start(settings)
	}
	if !running || samePort {
		return nil
	}
	if err := c.Stop(ctx); err != nil {
		return err
	}
	return c.start(settings)
}

// OnStatusChange registers fn to run after every status transition.
func (c *Connection) OnStatusChange(fn func(app.ConnectionStatus)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, fn)
}

func (c *Connection) setStatus(st app.ConnectionStatus) {
	c.mu.Lock()
	c.status = st
	c.mu.Unlock()
	c.notify(st)
}

func (c *Connection) notify(st app.ConnectionStatus) {
	c.mu.Lock()
	fns := slices.Clone(c.listeners)
	c.mu.Unlock()
	for _, fn := range fns {
		fn(st)
	}
}
