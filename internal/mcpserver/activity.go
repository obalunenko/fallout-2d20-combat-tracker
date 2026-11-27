package mcpserver

import (
	"context"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/domain"
)

// activity describes what a modifying or destructive tool call did. Callers
// fill it as they go and record it once, for the final outcome.
type activity struct {
	Operation string
	Kind      domain.AssistantEntityKind
	ID        string
	Label     string
	Detail    string
}

// record writes one activity entry for the final outcome of a call. err is
// the call's error (nil when it succeeded). Detail strings never contain the
// access secret: nothing in the adapter has it except the admission check.
func (d *deps) record(ctx context.Context, a activity, err error) {
	entry := domain.AssistantActivityEntry{
		Operation:   a.Operation,
		EntityKind:  a.Kind,
		EntityID:    a.ID,
		EntityLabel: a.Label,
		Outcome:     domain.OutcomeDone,
		Detail:      a.Detail,
	}
	if err != nil {
		// The detail is what the assistant was told: an internal error keeps
		// its cause in the tracker log, never in the activity it can read.
		entry.Outcome = outcomeFor(err)
		entry.Detail = asToolError(err, func(string, ...any) {}).Message
	}
	d.svc.AppendAssistantActivity(context.WithoutCancel(ctx), entry)
}

// finish records the final outcome of a modifying or destructive call. A
// non-nil res asks the client for input first (input_required), which is not
// final, so finish records nothing for it (FR-024). Handlers call it from a
// deferred func over their named results.
func (d *deps) finish(ctx context.Context, a activity, res *mcp.CallToolResult, err error) {
	if res != nil {
		return
	}
	d.record(ctx, a, err)
}

func outcomeFor(err error) domain.AssistantOutcome {
	var te *toolError
	if errors.As(err, &te) {
		switch te.Code {
		case codeDeclined:
			if te.unverified {
				return domain.OutcomeRejected
			}
			return domain.OutcomeDeclined
		case codeTimedOut:
			return domain.OutcomeTimedOut
		}
	}
	return domain.OutcomeRejected
}
