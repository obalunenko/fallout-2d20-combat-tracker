package mcpserver

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestAssistantAndDesktopWritesInterleaveSafely runs assistant tool calls
// next to direct service calls (as the desktop UI makes them). No update may
// be lost and nothing may fail as busy under a normal load (US6 acceptance
// scenario 4). Run it with -race.
func TestAssistantAndDesktopWritesInterleaveSafely(t *testing.T) {
	env := newTestEnv(t)
	ids := seedCombat(t, env.svc)
	robot := ids["Robot"]
	cs := env.client(t, clientOptions{})

	const assistantHits, desktopHits, turns = 8, 8, 6
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, assistantHits+desktopHits+turns)
	for range assistantHits {
		wg.Go(func() {
			// Head DR 2: each 3-point hit applies 1. No require here: this
			// runs on another goroutine.
			res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "apply_damage", Arguments: map[string]any{
				"combatant_id": robot, "damage_type": "physical", "location": "head", "amount": 3,
			}})
			if err == nil && res.IsError {
				err = &toolError{Code: "tool", Message: resultText(res)}
			}
			errs <- err
		})
	}
	for range desktopHits {
		wg.Go(func() {
			_, _, err := env.svc.ApplyDamage(ctx, robot, domain.DamagePhysical, domain.BodyHead, 3)
			errs <- err
		})
	}
	for range turns {
		wg.Go(func() {
			_, err := env.svc.AdvanceTurn(ctx)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	enc, err := env.svc.GetEncounter(t.Context())
	require.NoError(t, err)
	got, err := combatantIn(enc, robot)
	require.NoError(t, err)
	require.Equal(t, 20-(assistantHits+desktopHits), got.HP, "every hit from both sides is kept")
	require.Equal(t, 1+turns/3, enc.Round, "three combatants: six advances end on round 3")
}
