package mcpserver

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
)

const (
	// mrtrProtocolVersion is the first protocol revision whose clients answer
	// input requests by retrying the call (SEP-2322).
	mrtrProtocolVersion = "2026-07-28"
	confirmInputID      = "confirm"
	// confirmTokenTTL is how long an in-conversation confirmation stays open.
	confirmTokenTTL = 2 * time.Minute
	// usedNonceGrace keeps a used nonce past its token's expiry, so a replay
	// shortly after the expiry is still seen as a reuse (research R5).
	usedNonceGrace = time.Minute
	// desktopConfirmTimeout keeps the desktop dialog below the 60-second
	// request timeout common in MCP clients (research R5).
	desktopConfirmTimeout = 50 * time.Second
)

// confirmSchema is the form an in-conversation confirmation asks for.
var confirmSchema = &jsonschema.Schema{
	Type:       "object",
	Properties: map[string]*jsonschema.Schema{"confirm": {Type: "boolean", Title: "Confirm"}},
	Required:   []string{"confirm"},
}

var (
	errConfirmDeclined   = &toolError{Code: codeDeclined, Message: "the GM declined the confirmation; nothing was changed"}
	errConfirmTimedOut   = &toolError{Code: codeTimedOut, Message: "the GM did not answer the confirmation in time; nothing was changed, you may offer to try again"}
	errConfirmUnverified = &toolError{Code: codeDeclined, Message: "confirmation could not be verified", unverified: true}
)

// confirmer asks the GM to confirm destructive tool calls (FR-015): in
// conversation when the client can elicit, otherwise in the desktop app.
// Callers never hold the app gate while it waits.
type confirmer struct {
	key    []byte
	router *app.ConfirmationRouter

	mu   sync.Mutex
	now  func() time.Time
	used map[string]time.Time // nonce -> when it may be forgotten
	// open is cancelled while the connection is stopped, which closes every
	// desktop confirmation as declined.
	open      context.Context
	closeOpen context.CancelFunc
}

func newConfirmer(router *app.ConfirmationRouter) *confirmer {
	key := make([]byte, sha256.Size)
	_, _ = rand.Read(key) // crypto/rand.Read never fails (Go 1.24+).
	c := &confirmer{key: key, router: router, now: time.Now, used: make(map[string]time.Time)}
	c.resume()
	return c
}

// resume lets desktop confirmations wait again after cancelPending.
func (c *confirmer) resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.open == nil || c.open.Err() != nil {
		c.open, c.closeOpen = context.WithCancel(context.Background())
	}
}

// cancelPending closes the open desktop confirmations as declined and
// declines new ones until resume. The connection calls it when it stops.
func (c *confirmer) cancelPending() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeOpen()
}

// confirmToken is the signed payload of a requestState.
type confirmToken struct {
	Tool    string `json:"t"`
	Args    string `json:"a"` // SHA-256 of the canonical arguments JSON
	Expires int64  `json:"e"` // Unix milliseconds
	Nonce   string `json:"n"`
}

// require asks the GM to confirm cr for the call req with the decoded
// arguments args. It returns a non-nil result when the client must answer an
// input request first; ok reports that the GM confirmed and the operation may
// run; otherwise err says why not (declined or timed_out).
func (c *confirmer) require(ctx context.Context, req *mcp.CallToolRequest, args any, cr app.ConfirmationRequest) (*mcp.CallToolResult, bool, error) {
	if !canElicit(req) {
		if err := c.desktop(ctx, cr); err != nil {
			return nil, false, err
		}
		return nil, true, nil
	}
	digest, err := argsDigest(args)
	if err != nil {
		return nil, false, err
	}
	tool := req.Params.Name
	if req.Params.RequestState == "" {
		return c.ask(tool, digest, cr), false, nil
	}
	if err := c.verify(tool, digest, req.Params.RequestState); err != nil {
		return nil, false, err
	}
	if !accepted(req.Params.InputResponses) {
		return nil, false, errConfirmDeclined
	}
	return nil, true, nil
}

// canElicit reports whether the client takes in-conversation confirmations:
// it speaks 2026-07-28 or later and declares form elicitation.
func canElicit(req *mcp.CallToolRequest) bool {
	if req.ProtocolVersion() < mrtrProtocolVersion {
		return false
	}
	caps := req.ClientCapabilities()
	if caps == nil || caps.Elicitation == nil {
		return false
	}
	// An empty elicitation capability means form elicitation.
	return caps.Elicitation.Form != nil || caps.Elicitation.URL == nil
}

// ask returns the input_required result with a fresh single-use token.
func (c *confirmer) ask(tool, digest string, cr app.ConfirmationRequest) *mcp.CallToolResult {
	c.mu.Lock()
	expires := c.now().Add(confirmTokenTTL)
	c.mu.Unlock()
	payload, _ := json.Marshal(confirmToken{Tool: tool, Args: digest, Expires: expires.UnixMilli(), Nonce: rand.Text()})
	enc := base64.RawURLEncoding
	return &mcp.CallToolResult{
		InputRequests: mcp.InputRequestMap{confirmInputID: &mcp.ElicitParams{
			Mode:            "form",
			Message:         cr.Summary,
			RequestedSchema: confirmSchema,
		}},
		RequestState: enc.EncodeToString(payload) + "." + enc.EncodeToString(c.sign(payload)),
	}
}

// verify checks a requestState in order: signature, tool and arguments,
// unused nonce, then expiry. It uses the nonce up either way.
func (c *confirmer) verify(tool, digest, state string) error {
	enc := base64.RawURLEncoding
	p64, m64, ok := strings.Cut(state, ".")
	if !ok {
		return errConfirmUnverified
	}
	payload, err1 := enc.DecodeString(p64)
	mac, err2 := enc.DecodeString(m64)
	if err1 != nil || err2 != nil || !hmac.Equal(mac, c.sign(payload)) {
		return errConfirmUnverified
	}
	var tok confirmToken
	if err := json.Unmarshal(payload, &tok); err != nil || tok.Tool != tool || tok.Args != digest {
		return errConfirmUnverified
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for n, forgetAt := range c.used {
		if now.After(forgetAt) {
			delete(c.used, n)
		}
	}
	if _, used := c.used[tok.Nonce]; used {
		return errConfirmUnverified
	}
	expires := time.UnixMilli(tok.Expires)
	c.used[tok.Nonce] = expires.Add(usedNonceGrace)
	if now.After(expires) {
		return errConfirmTimedOut
	}
	return nil
}

func (c *confirmer) sign(payload []byte) []byte {
	h := hmac.New(sha256.New, c.key)
	h.Write(payload)
	return h.Sum(nil)
}

// accepted reports whether the client answered the confirmation with
// accept and confirm=true.
func accepted(responses mcp.InputResponseMap) bool {
	r, ok := responses[confirmInputID].(*mcp.ElicitResult)
	if !ok || r.Action != "accept" {
		return false
	}
	v, _ := r.Content["confirm"].(bool)
	return v
}

// argsDigest hashes the decoded arguments. Marshaling the typed input makes
// the digest independent of key order and spacing in the request.
func argsDigest(args any) (string, error) {
	b, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("digest tool arguments: %w", err)
	}
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// desktop asks through the desktop app and waits up to 50 seconds. A
// cancelled or dropped request, or a connection that stops meanwhile, is a
// decline.
func (c *confirmer) desktop(ctx context.Context, cr app.ConfirmationRequest) error {
	c.mu.Lock()
	open := c.open
	c.mu.Unlock()
	req := requestContext(ctx)
	dctx, cancel := context.WithTimeout(ctx, desktopConfirmTimeout)
	defer cancel()
	defer context.AfterFunc(open, cancel)()
	defer context.AfterFunc(req, cancel)()
	decision, err := c.router.Confirm(dctx, cr)
	switch {
	case ctx.Err() != nil, req.Err() != nil, open.Err() != nil:
		return errConfirmDeclined
	case err != nil:
		return err
	}
	switch decision {
	case app.ConfirmationAccepted:
		return nil
	case app.ConfirmationTimedOut:
		return errConfirmTimedOut
	}
	if errors.Is(dctx.Err(), context.DeadlineExceeded) {
		return errConfirmTimedOut
	}
	return errConfirmDeclined
}
