package mcpserver

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/domain"
)

// Tool error codes (contracts/mcp-tools.md, "Error contract").
const (
	codeInvalidInput      = "invalid_input"
	codeNotFound          = "not_found"
	codeNoCampaign        = "no_campaign"
	codeNoActiveEncounter = "no_active_encounter"
	codeBusy              = "busy"
	codeDeclined          = "declined"
	codeTimedOut          = "timed_out"
	codeInternal          = "internal"
)

// toolError is the error a tool returns to the assistant.
type toolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	// unverified marks a declined confirmation whose answer could not be
	// verified; the activity log records it as rejected.
	unverified bool
}

func (e *toolError) Error() string { return e.Message }

func invalidInput(field, message string) *toolError {
	return &toolError{Code: codeInvalidInput, Message: message, Field: field}
}

func notFound(message string) *toolError { return &toolError{Code: codeNotFound, Message: message} }

// asToolError maps an application error to a tool error by its kind, never by
// its wording: errors marked domain.ErrInvalidInput or domain.ErrNotFound pass
// their message through verbatim, because it already names the field or the
// identifier (SC-005). Anything unmarked is an internal error: logf receives
// the detail and the assistant gets a generic message.
func asToolError(err error, logf func(string, ...any)) *toolError {
	var te *toolError
	switch {
	case errors.As(err, &te):
		return te
	case errors.Is(err, domain.ErrCampaignNotInitialized):
		return &toolError{Code: codeNoCampaign, Message: "no campaign exists yet; create one first"}
	case errors.Is(err, domain.ErrEncounterNotInitialized):
		return &toolError{Code: codeNoActiveEncounter, Message: "there is no active encounter; activate one first"}
	case errors.Is(err, domain.ErrEncounterNotFound):
		return notFound("encounter not found")
	case errors.Is(err, domain.ErrCampaignNotFound):
		return notFound("campaign not found")
	case errors.Is(err, domain.ErrTrackerBusy):
		return &toolError{Code: codeBusy, Message: "the tracker is busy, try again"}
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return &toolError{Code: codeBusy, Message: "the request was cancelled or timed out, try again"}
	case errors.Is(err, domain.ErrNotFound):
		return notFound(err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return &toolError{Code: codeInvalidInput, Message: err.Error()}
	}
	logf("mcp tool internal error: %v", err)
	return &toolError{Code: codeInternal, Message: "internal error; see the tracker log"}
}

// errorResult builds the tool result for te. The JSON error object travels as
// the text content, not as structuredContent, so clients that check
// structuredContent against the output schema do not reject it.
func errorResult(te *toolError) *mcp.CallToolResult {
	b, _ := json.Marshal(struct {
		Error *toolError `json:"error"`
	}{te})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}
