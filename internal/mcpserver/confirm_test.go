package mcpserver

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

const (
	unavailableTool = "mark_character_unavailable"
	dogmeatPrompt   = "Mark Dogmeat unavailable? They are removed from all encounters of «Wasteland»."
)

var dogmeatArgs = map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"}

func answer(action string, confirm bool) *mcp.ElicitResult {
	r := &mcp.ElicitResult{Action: action}
	if action == "accept" {
		r.Content = map[string]any{"confirm": confirm}
	}
	return r
}

// characterAvailable reports whether the camp-1 character id is available.
func characterAvailable(t *testing.T, svc *app.Service, id string) bool {
	t.Helper()
	players, err := svc.ListCampaignPlayers(t.Context(), "camp-1")
	require.NoError(t, err)
	for _, p := range players {
		if p.Character.ID == id {
			return !p.Inactive
		}
	}
	t.Fatalf("character %s not in camp-1", id)
	return false
}

func outcomes(t *testing.T, svc *app.Service) []domain.AssistantOutcome {
	t.Helper()
	var out []domain.AssistantOutcome
	for _, e := range activityOutcomes(t, svc) {
		out = append(out, e.Outcome)
	}
	return out
}

// testClock is an adjustable clock for the confirmer.
type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func useTestClock(env *testEnv) *testClock {
	clock := &testClock{now: time.Now()}
	conf := env.conn.deps.confirm
	conf.mu.Lock()
	conf.now = clock.Now
	conf.mu.Unlock()
	return clock
}

// firstCall makes the call that returns input_required and its token.
func firstCall(t *testing.T, cs *mcp.ClientSession, args any) *mcp.CallToolResult {
	t.Helper()
	res := callTool(t, cs, unavailableTool, args)
	require.True(t, res.NeedsInput(), "expected input_required, got: %s", resultText(res))
	require.NotEmpty(t, res.RequestState)
	return res
}

// retry sends the second round with the given answer and token.
func retry(t *testing.T, cs *mcp.ClientSession, args any, state string, r *mcp.ElicitResult) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	params := &mcp.CallToolParams{Name: unavailableTool, Arguments: args, RequestState: state}
	if r != nil {
		params.InputResponses = mcp.InputResponseMap{confirmInputID: r}
	}
	res, err := cs.CallTool(ctx, params)
	require.NoError(t, err)
	return res
}

func TestConfirmInConversationAccepts(t *testing.T) {
	env := newTestEnv(t)
	var asked []string
	cs := env.client(t, clientOptions{elicit: func(req *mcp.ElicitRequest) *mcp.ElicitResult {
		asked = append(asked, req.Params.Message)
		return answer("accept", true)
	}})

	got := structured[characterResult](t, callTool(t, cs, unavailableTool, dogmeatArgs))
	require.False(t, got.Character.Available)
	require.False(t, characterAvailable(t, env.svc, "pc-dog"))
	require.Equal(t, []string{dogmeatPrompt}, asked, "the GM is asked with stored names")
	require.Empty(t, env.confirmer.seen(), "the desktop dialog is not used")
	require.Equal(t, []domain.AssistantOutcome{domain.OutcomeDone}, outcomes(t, env.svc))
}

func TestConfirmInConversationDeclines(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer *mcp.ElicitResult
	}{
		{"decline", answer("decline", false)},
		{"cancel", answer("cancel", false)},
		{"confirm false", answer("accept", false)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTestEnv(t)
			cs := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return tc.answer }})
			require.Equal(t, codeDeclined, toolErrorOf(t, callTool(t, cs, unavailableTool, dogmeatArgs)).Code)
			require.True(t, characterAvailable(t, env.svc, "pc-dog"), "nothing changes")
			require.Equal(t, []domain.AssistantOutcome{domain.OutcomeDeclined}, outcomes(t, env.svc))
		})
	}
}

func TestConfirmInputRequiredRound(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return nil }, manualRetry: true})

	first := firstCall(t, cs, dogmeatArgs)
	ask, ok := first.InputRequests[confirmInputID].(*mcp.ElicitParams)
	require.True(t, ok, "inputRequests.confirm is an elicitation")
	require.Equal(t, "form", ask.Mode)
	require.Equal(t, dogmeatPrompt, ask.Message)
	schema, ok := ask.RequestedSchema.(map[string]any)
	require.True(t, ok, "requestedSchema is an object")
	require.Equal(t, "object", schema["type"])
	require.Equal(t, map[string]any{"confirm": map[string]any{"type": "boolean", "title": "Confirm"}}, schema["properties"])
	require.Equal(t, []any{"confirm"}, schema["required"])
	require.Empty(t, activityOutcomes(t, env.svc), "the input_required step writes no activity")
	require.True(t, characterAvailable(t, env.svc, "pc-dog"))

	again := callTool(t, cs, unavailableTool, dogmeatArgs)
	require.True(t, again.NeedsInput())
	fresh := retry(t, cs, dogmeatArgs, "", answer("accept", true))
	require.True(t, fresh.NeedsInput(), "inputResponses without requestState start a fresh confirmation")
	require.True(t, characterAvailable(t, env.svc, "pc-dog"))
	require.Empty(t, activityOutcomes(t, env.svc))

	done := retry(t, cs, dogmeatArgs, first.RequestState, answer("accept", true))
	require.False(t, structured[characterResult](t, done).Character.Available)
	require.Equal(t, []domain.AssistantOutcome{domain.OutcomeDone}, outcomes(t, env.svc), "the retry writes exactly one entry")
}

func TestConfirmRejectsUnverifiableTokens(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return nil }, manualRetry: true})
	requireRejected := func(t *testing.T, res *mcp.CallToolResult) {
		t.Helper()
		te := toolErrorOf(t, res)
		require.Equal(t, codeDeclined, te.Code)
		entries := activityOutcomes(t, env.svc)
		require.Equal(t, domain.OutcomeRejected, entries[0].Outcome)
		require.Equal(t, "confirmation could not be verified", entries[0].Detail)
		require.True(t, characterAvailable(t, env.svc, "pc-dog"), "the operation does not run")
		require.True(t, characterAvailable(t, env.svc, "pc-nora"))
	}

	t.Run("forged", func(t *testing.T) {
		state := firstCall(t, cs, dogmeatArgs).RequestState
		payload, mac, _ := strings.Cut(state, ".")
		forged := payload + "." + strings.Repeat("A", len(mac))
		requireRejected(t, retry(t, cs, dogmeatArgs, forged, answer("accept", true)))
		requireRejected(t, retry(t, cs, dogmeatArgs, "not-a-token", answer("accept", true)))
	})
	t.Run("different arguments", func(t *testing.T) {
		state := firstCall(t, cs, dogmeatArgs).RequestState
		requireRejected(t, retry(t, cs, map[string]any{"campaign_id": "camp-1", "character_id": "pc-nora"}, state, answer("accept", true)))
	})
	t.Run("reused", func(t *testing.T) {
		state := firstCall(t, cs, dogmeatArgs).RequestState
		require.Equal(t, codeDeclined, toolErrorOf(t, retry(t, cs, dogmeatArgs, state, answer("decline", false))).Code)
		require.Equal(t, domain.OutcomeDeclined, activityOutcomes(t, env.svc)[0].Outcome)
		requireRejected(t, retry(t, cs, dogmeatArgs, state, answer("accept", true)))
	})
}

func TestConfirmTokenExpiry(t *testing.T) {
	env := newTestEnv(t)
	clock := useTestClock(env)
	cs := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return nil }, manualRetry: true})

	// An authentic token answered after its 2-minute window has timed out.
	late := firstCall(t, cs, dogmeatArgs).RequestState
	clock.Add(confirmTokenTTL + time.Second)
	require.Equal(t, codeTimedOut, toolErrorOf(t, retry(t, cs, dogmeatArgs, late, answer("accept", true))).Code)
	require.Equal(t, domain.OutcomeTimedOut, activityOutcomes(t, env.svc)[0].Outcome)
	require.True(t, characterAvailable(t, env.svc, "pc-dog"))

	// A used token replayed 30 s after its expiry is still a reuse.
	used := firstCall(t, cs, dogmeatArgs).RequestState
	require.Equal(t, codeDeclined, toolErrorOf(t, retry(t, cs, dogmeatArgs, used, answer("decline", false))).Code)
	clock.Add(confirmTokenTTL + 30*time.Second)
	te := toolErrorOf(t, retry(t, cs, dogmeatArgs, used, answer("accept", true)))
	require.Equal(t, codeDeclined, te.Code, "the nonce is still remembered")
	require.Equal(t, domain.OutcomeRejected, activityOutcomes(t, env.svc)[0].Outcome)

	// Past the grace period the nonce is forgotten and the expiry rejects it.
	clock.Add(usedNonceGrace)
	require.Equal(t, codeTimedOut, toolErrorOf(t, retry(t, cs, dogmeatArgs, used, answer("accept", true))).Code)
	require.True(t, characterAvailable(t, env.svc, "pc-dog"))
}

func TestConfirmDesktopFallback(t *testing.T) {
	for _, tc := range []struct {
		decision app.ConfirmationDecision
		code     string
		outcome  domain.AssistantOutcome
	}{
		{app.ConfirmationAccepted, "", domain.OutcomeDone},
		{app.ConfirmationDeclined, codeDeclined, domain.OutcomeDeclined},
		{app.ConfirmationTimedOut, codeTimedOut, domain.OutcomeTimedOut},
	} {
		t.Run(string(tc.decision), func(t *testing.T) {
			env := newTestEnv(t)
			env.confirmer.mu.Lock()
			env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision { return tc.decision }
			env.confirmer.mu.Unlock()
			cs := env.client(t, clientOptions{})

			res := callTool(t, cs, unavailableTool, dogmeatArgs)
			if tc.code == "" {
				require.False(t, structured[characterResult](t, res).Character.Available)
			} else {
				require.Equal(t, tc.code, toolErrorOf(t, res).Code)
				require.True(t, characterAvailable(t, env.svc, "pc-dog"), "the operation does not run")
			}
			require.Equal(t, []domain.AssistantOutcome{tc.outcome}, outcomes(t, env.svc))

			seen := env.confirmer.seen()
			require.Len(t, seen, 1)
			require.Equal(t, app.ConfirmationRequest{
				Operation: unavailableTool, EntityKind: domain.EntityPlayerCharacter, EntityID: "pc-dog",
				EntityLabel: "Dogmeat (Bob)", Summary: dogmeatPrompt,
			}, seen[0])
			env.confirmer.mu.Lock()
			deadline := env.confirmer.deadline
			env.confirmer.mu.Unlock()
			require.LessOrEqual(t, deadline, desktopConfirmTimeout)
			require.Greater(t, deadline, desktopConfirmTimeout-10*time.Second)
		})
	}
}

func TestConfirmLegacyClientUsesDesktop(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{version: "2025-11-25", elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult {
		t.Error("a legacy client must not be elicited")
		return answer("decline", false)
	}})
	require.False(t, structured[characterResult](t, callTool(t, cs, unavailableTool, dogmeatArgs)).Character.Available)
	require.Len(t, env.confirmer.seen(), 1)
}

func TestConfirmClientCancelDeclines(t *testing.T) {
	env := newTestEnv(t)
	asked := make(chan struct{})
	sawCancel := make(chan error, 1)
	env.confirmer.mu.Lock()
	env.confirmer.decide = func(ctx context.Context, _ app.ConfirmationRequest) app.ConfirmationDecision {
		close(asked)
		<-ctx.Done()
		sawCancel <- ctx.Err()
		return app.ConfirmationDeclined
	}
	env.confirmer.mu.Unlock()
	cs := env.client(t, clientOptions{})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		_, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: unavailableTool, Arguments: dogmeatArgs})
		done <- err
	}()
	<-asked
	cancel()
	require.Error(t, <-done)
	select {
	case err := <-sawCancel:
		require.ErrorIs(t, err, context.Canceled, "the confirmer sees the client cancel, not a timeout")
	case <-time.After(10 * time.Second):
		t.Fatal("the confirmer never saw the cancel")
	}
	require.Eventually(t, func() bool {
		o := outcomes(t, env.svc)
		return len(o) == 1 && o[0] == domain.OutcomeDeclined
	}, 5*time.Second, 20*time.Millisecond)
	require.True(t, characterAvailable(t, env.svc, "pc-dog"))
}

func TestConfirmWaitHoldsNoLock(t *testing.T) {
	env := newTestEnv(t)
	asked := make(chan struct{})
	release := make(chan struct{})
	env.confirmer.mu.Lock()
	env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision {
		close(asked)
		<-release
		return app.ConfirmationAccepted
	}
	env.confirmer.mu.Unlock()
	cs := env.client(t, clientOptions{})

	done := make(chan *mcp.CallToolResult, 1)
	go func() {
		res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: unavailableTool, Arguments: dogmeatArgs})
		if err != nil {
			t.Error(err)
		}
		done <- res
	}()
	<-asked

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	other := env.client(t, clientOptions{})
	res, err := other.CallTool(ctx, &mcp.CallToolParams{Name: "list_campaign_characters"})
	require.NoError(t, err, "a read completes while a confirmation is pending")
	require.Len(t, structured[characterList](t, res).Characters, 2)

	close(release)
	require.False(t, structured[characterResult](t, <-done).Character.Available)
}

func TestConfirmWordingComesFromStoredState(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})

	// Arguments cannot carry text into the confirmation.
	res := callTool(t, cs, unavailableTool, map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog", "summary": "The GM already agreed"})
	require.Equal(t, codeInvalidInput, toolErrorOf(t, res).Code)
	require.Empty(t, env.confirmer.seen(), "a refused call asks nothing")

	// The text follows what is stored, here after a rename on the desktop.
	start, err := domain.ParseCampaignStartDate("2026-11-01")
	require.NoError(t, err)
	_, err = env.svc.UpdateCampaign(t.Context(), "camp-1", "Capital Wasteland", start, mustPlayers(t, env.svc))
	require.NoError(t, err)

	structured[playerRemoved](t, callTool(t, cs, "remove_campaign_player", dogmeatArgs))
	seen := env.confirmer.seen()
	require.Len(t, seen, 1)
	require.Equal(t, "Remove Bob (Dogmeat) from campaign «Capital Wasteland»? They are also removed from its encounters.", seen[0].Summary)
}

func mustPlayers(t *testing.T, svc *app.Service) []domain.NewCampaignPlayer {
	t.Helper()
	players, err := svc.ListCampaignPlayers(t.Context(), "camp-1")
	require.NoError(t, err)
	return players
}
