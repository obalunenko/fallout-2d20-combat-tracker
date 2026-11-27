package mcpserver

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

var combatTools = []string{"advance_turn", "apply_damage", "heal", "change_party_ap", "change_gm_threat"}

// seedCombat creates an active encounter with Nora, a robot with per-location
// DR, global poison DR and radiation immunity, and a torso-only bug.
func seedCombat(t *testing.T, svc *app.Service) map[string]string {
	t.Helper()
	enc, err := svc.CreateEncounter(t.Context(), "", "Arena", []domain.Combatant{
		{ID: "pc-nora", PlayerCharacterID: "pc-nora", Name: "Nora", Side: domain.SideParty, Level: 3, Initiative: 9, HP: 12, MaxHP: 12},
		{Name: "Robot", Side: domain.SideNPC, Level: 3, XP: 30, Initiative: 5, HP: 20, MaxHP: 20,
			ResistPhysicalHead: 2, ResistPhysicalTorso: 5, ResistEnergyLeftArm: 1, ResistPoison: 3, ImmuneRadiation: true},
		{Name: "Bug", Side: domain.SideNPC, Level: 1, XP: 10, Initiative: 3, HP: 10, MaxHP: 10, TorsoOnly: true, ResistPhysicalTorso: 4},
	})
	require.NoError(t, err)
	ids := map[string]string{}
	for _, c := range enc.Combatants {
		ids[c.Name] = c.ID
	}
	return ids
}

func damage(t *testing.T, cs *mcp.ClientSession, id, typ, loc string, amount int) damageResult {
	t.Helper()
	return structured[damageResult](t, callTool(t, cs, "apply_damage", map[string]any{"combatant_id": id, "damage_type": typ, "location": loc, "amount": amount}))
}

func TestCombatToolsApplyTrackerRules(t *testing.T) {
	env := newTestEnv(t)
	ids := seedCombat(t, env.svc)
	cs := env.client(t, clientOptions{})

	turn := structured[turnResult](t, callTool(t, cs, "advance_turn", nil))
	require.Equal(t, "Robot", turn.CurrentCombatant.Name)
	require.Equal(t, 1, turn.Round)

	require.Equal(t, 3, damage(t, cs, ids["Robot"], "physical", "head", 5).Applied, "5 minus head DR 2")
	require.Equal(t, 3, damage(t, cs, ids["Robot"], "energy", "left_arm", 4).Applied, "4 minus left-arm energy DR 1")
	require.Equal(t, 0, damage(t, cs, ids["Robot"], "radiation", "torso", 9).Applied, "immune")
	require.Equal(t, 2, damage(t, cs, ids["Robot"], "poison", "left_leg", 5).Applied, "global poison DR 3")
	require.Equal(t, 2, damage(t, cs, ids["Bug"], "physical", "head", 6).Applied, "torso-only: torso DR 4 applies")

	down := damage(t, cs, ids["Bug"], "energy", "torso", 50)
	require.Equal(t, 0, down.Combatant.HP)
	require.True(t, down.Defeated)

	healed := structured[healResult](t, callTool(t, cs, "heal", map[string]any{"combatant_id": ids["Robot"], "amount": 100}))
	require.Equal(t, 8, healed.Healed, "healing stops at max HP (20-12 missing)")
	require.Equal(t, 20, healed.Combatant.HP)

	require.Equal(t, codeNotFound, toolErrorOf(t, callTool(t, cs, "apply_damage", map[string]any{"combatant_id": "ghost", "damage_type": "physical", "location": "torso", "amount": 1})).Code)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "apply_damage", map[string]any{"combatant_id": ids["Robot"], "damage_type": "fire", "location": "torso", "amount": 1})).Code)
	bad := toolErrorOf(t, callTool(t, cs, "apply_damage", map[string]any{"combatant_id": ids["Robot"], "damage_type": "physical", "location": "torso", "amount": -1}))
	require.Equal(t, codeInvalidInput, bad.Code)
	require.Equal(t, "amount", bad.Field)

	ap := structured[apResult](t, callTool(t, cs, "change_party_ap", map[string]any{"delta": 3}))
	require.Equal(t, apResult{PartyAP: 3}, ap)
	ap = structured[apResult](t, callTool(t, cs, "change_party_ap", map[string]any{"delta": 5}))
	require.Equal(t, apResult{PartyAP: 6, Capped: true}, ap)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "change_party_ap", map[string]any{"delta": -7})).Code, "not enough AP")
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "change_party_ap", map[string]any{"delta": 0})).Code)
	require.Equal(t, 2, structured[threatResult](t, callTool(t, cs, "change_gm_threat", map[string]any{"delta": 2})).GMThreat)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "change_gm_threat", map[string]any{"delta": -5})).Code)

	enc, err := env.svc.GetEncounter(t.Context())
	require.NoError(t, err)
	require.Equal(t, 6, enc.Resources.PartyAP, "a rejected spend leaves AP unchanged")
	require.Equal(t, 2, enc.Resources.GMThreat)

	logs, err := env.svc.ListEncounterLogs(t.Context(), enc.ID)
	require.NoError(t, err)
	require.Equal(t, domain.SourceAssistant, logs[0].Source, "assistant combat actions are tagged")

	entries := activityOutcomes(t, env.svc)
	require.NotEmpty(t, entries)
	require.Equal(t, "change_gm_threat", entries[0].Operation)
	require.Equal(t, domain.OutcomeRejected, entries[0].Outcome)
}

func TestCombatToolsMatchDesktopActions(t *testing.T) {
	viaTool, viaApp := newTestEnv(t), newTestService(t)
	toolIDs, appIDs := seedCombat(t, viaTool.svc), seedCombat(t, viaApp)
	cs := viaTool.client(t, clientOptions{})

	got := damage(t, cs, toolIDs["Robot"], "physical", "torso", 9)
	enc, applied, err := viaApp.ApplyDamage(t.Context(), appIDs["Robot"], domain.DamagePhysical, domain.BodyTorso, 9)
	require.NoError(t, err)
	require.Equal(t, applied, got.Applied)
	want, err := combatantIn(enc, appIDs["Robot"])
	require.NoError(t, err)
	got.Combatant.CombatantID, want.CombatantID = "", ""
	require.Equal(t, want, got.Combatant, "same stored result as the desktop action")
}

func TestCombatToolsNeedAnActiveEncounter(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	args := map[string]map[string]any{
		"apply_damage":     {"combatant_id": "x", "damage_type": "physical", "location": "torso", "amount": 1},
		"heal":             {"combatant_id": "x", "amount": 1},
		"change_party_ap":  {"delta": 1},
		"change_gm_threat": {"delta": 1},
	}
	for _, name := range combatTools {
		require.Equal(t, codeNoActiveEncounter, toolErrorOf(t, callTool(t, cs, name, args[name])).Code, name)
	}
}
