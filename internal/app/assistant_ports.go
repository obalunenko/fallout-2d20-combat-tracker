package app

import (
	"context"
	"sync"

	"github.com/obalunenko/fallout/internal/domain"
)

// ConnectionState is the state of the assistant connection.
type ConnectionState string

const (
	ConnectionStopped ConnectionState = "stopped"
	ConnectionRunning ConnectionState = "running"
	ConnectionFailed  ConnectionState = "failed"
)

// ConnectionStatus describes the assistant connection for the desktop app.
type ConnectionStatus struct {
	State ConnectionState
	// Address is the endpoint URL while running, for example
	// http://127.0.0.1:47823/mcp.
	Address string
	// Reason explains a failed state, for example "port 47823 is in use".
	Reason string
}

// AssistantConnection is the assistant (MCP) endpoint as the desktop app sees
// it. internal/mcpserver implements it; internal/ui/fyneui consumes it.
type AssistantConnection interface {
	// StartIfEnabled starts the connection when the stored settings enable it.
	StartIfEnabled(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Status() ConnectionStatus
	// Apply makes settings take effect: it restarts on a port change and swaps
	// the access secret in place.
	Apply(ctx context.Context, settings domain.AssistantConnectionSettings) error
	// OnStatusChange registers fn to run after every status transition.
	OnStatusChange(fn func(ConnectionStatus))
}

// ConfirmationRequest asks the GM to confirm a destructive assistant
// operation. Its text comes from stored state, never from the assistant.
type ConfirmationRequest struct {
	Operation   string
	EntityKind  domain.AssistantEntityKind
	EntityID    string
	EntityLabel string
	Summary     string
}

// ConfirmationDecision is the GM's answer to a ConfirmationRequest.
type ConfirmationDecision string

const (
	ConfirmationAccepted ConfirmationDecision = "accepted"
	ConfirmationDeclined ConfirmationDecision = "declined"
	ConfirmationTimedOut ConfirmationDecision = "timed_out"
)

// DesktopConfirmer shows a confirmation in the desktop app. It returns when
// the GM answers, when its deadline passes (timed out), or when ctx is
// cancelled (declined).
type DesktopConfirmer interface {
	Confirm(ctx context.Context, req ConfirmationRequest) (ConfirmationDecision, error)
}

// ConfirmationRouter forwards confirmations to the desktop confirmer once the
// UI has registered one. Until then every confirmation is declined. It breaks
// the construction cycle between the MCP adapter and the Fyne UI.
type ConfirmationRouter struct {
	mu      sync.RWMutex
	desktop DesktopConfirmer
}

// NewConfirmationRouter returns a router with no desktop confirmer.
func NewConfirmationRouter() *ConfirmationRouter { return &ConfirmationRouter{} }

// SetDesktop registers the desktop confirmer.
func (r *ConfirmationRouter) SetDesktop(c DesktopConfirmer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.desktop = c
}

// Confirm asks the registered desktop confirmer, or declines when there is
// none.
func (r *ConfirmationRouter) Confirm(ctx context.Context, req ConfirmationRequest) (ConfirmationDecision, error) {
	r.mu.RLock()
	c := r.desktop
	r.mu.RUnlock()
	if c == nil {
		return ConfirmationDeclined, nil
	}
	return c.Confirm(ctx, req)
}
