package mcpserver

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

func zeroDR() map[string]any {
	return map[string]any{"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0}
}

func zeroResistance() map[string]any {
	return map[string]any{"physical": zeroDR(), "energy": zeroDR(), "radiation": zeroDR(), "poison": 0}
}

func noImmunity() map[string]any {
	return map[string]any{"physical": false, "energy": false, "radiation": false, "poison": false}
}

func character(name string) map[string]any {
	return map[string]any{"name": name, "level": 1, "initiative": 5, "hp": 5, "max_hp": 5, "defense": 0}
}

// TestEveryChangingToolWritesOneActivityEntry checks SC-010: each modifying
// or destructive call writes exactly one entry with its final outcome, and
// reads write none.
func TestEveryChangingToolWritesOneActivityEntry(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	count := func() int { return len(activityOutcomes(t, env.svc)) }

	expect := func(name string, args any, outcome domain.AssistantOutcome) *mcp.CallToolResult {
		t.Helper()
		before := count()
		res := callTool(t, cs, name, args)
		entries := activityOutcomes(t, env.svc)
		require.Len(t, entries, before+1, "%s writes exactly one entry", name)
		require.Equal(t, name, entries[0].Operation, "newest first")
		require.Equal(t, outcome, entries[0].Outcome, "%s: %s", name, resultText(res))
		return res
	}

	expect("create_campaign", map[string]any{"name": "Second", "start_date": "2026-12-01",
		"players": []any{map[string]any{"player_name": "Cid", "character": character("Cid")}}}, domain.OutcomeDone)
	expect("update_campaign_details", map[string]any{"campaign_id": "camp-1", "name": "Renamed"}, domain.OutcomeDone)
	added := structured[playerAdded](t, expect("add_campaign_player", map[string]any{"campaign_id": "camp-1", "player_name": "Eve", "character": character("Eve")}, domain.OutcomeDone))
	expect("update_campaign_character", map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog", "patch": map[string]any{"level": 3}}, domain.OutcomeDone)
	expect("mark_character_unavailable", map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"}, domain.OutcomeDone)
	expect("mark_character_available", map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"}, domain.OutcomeDone)
	expect("activate_campaign", map[string]any{"campaign_id": "camp-1"}, domain.OutcomeDone)

	tmpl := structured[map[string]any](t, expect("save_monster_template", map[string]any{"monster": map[string]any{
		"name": "Gecko", "level": 1, "xp": 5, "initiative": 4, "hp": 6, "max_hp": 6, "defense": 0,
		"resistance": zeroResistance(), "immunity": noImmunity(),
	}}, domain.OutcomeDone))
	templateID := tmpl["template"].(map[string]any)["template_id"].(string)

	enc := structured[Encounter](t, expect("create_encounter", map[string]any{"name": "Act", "party_character_ids": []string{"pc-nora"},
		"monsters": []any{map[string]any{"template_id": templateID, "quantity": 2}}}, domain.OutcomeDone))
	expect("activate_encounter", map[string]any{"encounter_id": enc.EncounterID}, domain.OutcomeDone)
	expect("advance_turn", nil, domain.OutcomeDone)
	gecko := enc.Combatants[len(enc.Combatants)-1].CombatantID
	hit := expect("apply_damage", map[string]any{"combatant_id": gecko, "damage_type": "physical", "location": "torso", "amount": 2}, domain.OutcomeDone)
	require.False(t, hit.IsError)
	entries := activityOutcomes(t, env.svc)
	require.Equal(t, domain.EntityCombatant, entries[0].EntityKind)
	require.Equal(t, gecko, entries[0].EntityID)
	require.Equal(t, "Gecko", entries[0].EntityLabel)

	expect("heal", map[string]any{"combatant_id": gecko, "amount": 1}, domain.OutcomeDone)
	expect("change_party_ap", map[string]any{"delta": 1}, domain.OutcomeDone)
	expect("change_gm_threat", map[string]any{"delta": 1}, domain.OutcomeDone)

	current, err := env.svc.GetEncounterByID(t.Context(), enc.EncounterID)
	require.NoError(t, err)
	keep := make([]string, 0, len(current.Combatants))
	for _, c := range current.Combatants {
		keep = append(keep, c.ID)
	}
	expect("update_encounter", map[string]any{"encounter_id": enc.EncounterID, "name": "Act 2", "keep_combatant_ids": keep}, domain.OutcomeDone)

	env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision {
		return app.ConfirmationDeclined
	}
	expect("restart_encounter", map[string]any{"encounter_id": enc.EncounterID}, domain.OutcomeDeclined)
	env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision {
		return app.ConfirmationTimedOut
	}
	expect("restart_encounter", map[string]any{"encounter_id": enc.EncounterID}, domain.OutcomeTimedOut)
	env.confirmer.decide = nil
	expect("delete_encounter", map[string]any{"encounter_id": enc.EncounterID}, domain.OutcomeDone)
	expect("remove_campaign_player", map[string]any{"campaign_id": "camp-1", "character_id": added.Character.CharacterID}, domain.OutcomeDone)
	expect("change_party_ap", map[string]any{"delta": 0}, domain.OutcomeRejected)

	before := count()
	for _, name := range []string{"list_campaigns", "list_encounters", "get_active_campaign", "list_monster_templates", "list_assistant_activity"} {
		callTool(t, cs, name, nil)
	}
	require.Equal(t, before, count(), "read tools write no activity")
	require.NotContains(t, fmt.Sprintf("%+v", activityOutcomes(t, env.svc)), string(testSecret))
}

func TestListAssistantActivityIsReadOnlyAndBounded(t *testing.T) {
	env := newTestEnv(t)
	for i := range 5 {
		env.svc.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{Operation: fmt.Sprintf("op-%d", i), Outcome: domain.OutcomeDone})
	}
	cs := env.client(t, clientOptions{})

	all := structured[activityList](t, callTool(t, cs, "list_assistant_activity", nil))
	require.Len(t, all.Entries, 5)
	require.Equal(t, "op-4", all.Entries[0].Operation, "newest first")
	require.Len(t, structured[activityList](t, callTool(t, cs, "list_assistant_activity", map[string]any{"limit": 2})).Entries, 2)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "list_assistant_activity", map[string]any{"limit": 201})).Code)

	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		require.NotContains(t, tool.Name, "clear", "the assistant cannot clear the log")
	}
}

// TestCorrectionThroughActivityLog covers US4 acceptance scenario 7: in a
// new conversation the assistant finds its last damage call in the activity
// log and reverses it with heal.
func TestCorrectionThroughActivityLog(t *testing.T) {
	env := newTestEnv(t)
	ids := seedCombat(t, env.svc)
	first := env.client(t, clientOptions{})
	hit := damage(t, first, ids["Robot"], "physical", "head", 6)
	require.Equal(t, 4, hit.Applied)

	second := env.client(t, clientOptions{}) // a new conversation
	log := structured[activityList](t, callTool(t, second, "list_assistant_activity", map[string]any{"limit": 1}))
	last := log.Entries[0]
	require.Equal(t, "apply_damage", last.Operation)
	require.Equal(t, ids["Robot"], last.EntityID)
	require.Contains(t, last.Detail, "applied 4")

	healed := structured[healResult](t, callTool(t, second, "heal", map[string]any{"combatant_id": last.EntityID, "amount": 4}))
	require.Equal(t, 20, healed.Combatant.HP, "the opposite operation restores the HP")
}

func TestActivityDetailNeverCarriesInternalErrors(t *testing.T) {
	env := newTestEnv(t)
	d := &deps{svc: env.svc, logf: func(string, ...any) {}}
	cause := fmt.Errorf("exec: UNIQUE constraint failed: player_characters.name")

	d.record(t.Context(), activity{Operation: "create_campaign"}, cause)
	d.record(t.Context(), activity{Operation: "create_campaign"}, domain.InvalidInput(fmt.Errorf("campaign name is required")))

	entries := activityOutcomes(t, env.svc)
	require.Len(t, entries, 2)
	require.Equal(t, "campaign name is required", entries[0].Detail, "a user error keeps its message")
	require.Equal(t, "internal error; see the tracker log", entries[1].Detail)
	require.NotContains(t, fmt.Sprintf("%+v", entries), "UNIQUE")
}
