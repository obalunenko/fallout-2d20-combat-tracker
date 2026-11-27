package mcpserver

import (
	"fmt"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestReadToolsAtScale checks SC-007: every read tool answers in under one
// second with 50 campaigns, 200 encounters and 100 monster templates.
func TestReadToolsAtScale(t *testing.T) {
	if testing.Short() {
		t.Skip("seeding takes a while; skipped with -short")
	}
	env := newTestEnv(t)
	ctx := t.Context()
	start, err := domain.ParseCampaignStartDate("2026-01-01")
	require.NoError(t, err)
	for i := range 49 {
		_, err := env.svc.CreateCampaign(ctx, "", fmt.Sprintf("Campaign %02d", i), start, []domain.NewCampaignPlayer{
			{PlayerName: fmt.Sprintf("P%02d", i), Character: domain.Combatant{Name: fmt.Sprintf("Hero %02d", i), Side: domain.SideParty, Level: 2, HP: 8, MaxHP: 8}},
		})
		require.NoError(t, err)
	}
	_, err = env.svc.ActivateCampaign(ctx, "camp-1")
	require.NoError(t, err)

	templates := make([]domain.Combatant, 0, 100)
	for i := range 100 {
		templates = append(templates, domain.Combatant{Name: fmt.Sprintf("Monster %03d", i), Level: 1 + i%5, XP: 10 + i, HP: 10, MaxHP: 10})
	}
	_, err = env.svc.SaveMonsterTemplates(ctx, templates)
	require.NoError(t, err)

	var lastID string
	for i := range 200 {
		enc, err := env.svc.CreateEncounter(ctx, "", fmt.Sprintf("Fight %03d", i), []domain.Combatant{
			{ID: "pc-nora", PlayerCharacterID: "pc-nora", Name: "Nora", Side: domain.SideParty, Level: 3, HP: 12, MaxHP: 12},
			{Name: "Raider", Side: domain.SideNPC, Level: 2, XP: 20, HP: 9, MaxHP: 9},
		})
		require.NoError(t, err)
		lastID = enc.ID
	}

	cs := env.client(t, clientOptions{})
	args := map[string]map[string]any{
		"get_encounter":     {"encounter_id": lastID},
		"get_encounter_log": {"encounter_id": lastID},
	}
	for _, name := range readTools {
		began := time.Now()
		res := callTool(t, cs, name, args[name])
		require.False(t, res.IsError, "%s: %s", name, resultText(res))
		require.Less(t, time.Since(began), time.Second, "%s took too long", name)
	}
}
