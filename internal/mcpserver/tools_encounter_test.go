package mcpserver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

var encounterTools = []string{
	"preview_encounter_difficulty", "create_encounter", "activate_encounter", "save_monster_template",
	"update_encounter", "restart_encounter", "delete_encounter",
}

// mirelurkInput is an inline torso-only monster with per-location DR.
var mirelurkInput = map[string]any{
	"name": "Mirelurk", "level": 4, "xp": 45, "initiative": 3, "hp": 15, "max_hp": 15, "defense": 1, "torso_only": true,
	"resistance": map[string]any{
		"physical":  map[string]any{"head": 0, "torso": 4, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
		"energy":    map[string]any{"head": 0, "torso": 2, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
		"radiation": map[string]any{"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
		"poison":    1,
	},
	"immunity": map[string]any{"physical": false, "energy": false, "radiation": true, "poison": false},
}

func mirelurk() domain.Combatant {
	return domain.Combatant{
		Name: "Mirelurk", Side: domain.SideNPC, Level: 4, XP: 45, Initiative: 3, HP: 15, MaxHP: 15, Defense: 1, TorsoOnly: true,
		ResistPhysicalTorso: 4, ResistEnergyTorso: 2, ResistPoison: 1, ImmuneRadiation: true,
	}
}

// raiderTemplate returns the raider template seedEncounter saves.
func raiderTemplate(t *testing.T, svc *app.Service) domain.Combatant {
	t.Helper()
	templates, err := svc.ListMonsterTemplates(t.Context())
	require.NoError(t, err)
	for _, tpl := range templates {
		if tpl.Name == "Raider" {
			return tpl
		}
	}
	t.Fatal("no raider template")
	return domain.Combatant{}
}

// decideAll makes the desktop confirmer answer every request with decision.
func (f *fakeConfirmer) decideAll(decision app.ConfirmationDecision) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision { return decision }
}

// withoutIDs clears the generated identifiers of e, so encounters built in
// different trackers can be compared.
func withoutIDs(e Encounter) Encounter {
	e.EncounterID, e.CampaignID, e.CurrentCombatantID = "", "", ""
	e.Combatants = append([]Combatant(nil), e.Combatants...)
	for i := range e.Combatants {
		e.Combatants[i].CombatantID = ""
	}
	return e
}

func encounterView(t *testing.T, cs *mcp.ClientSession, id string) Encounter {
	t.Helper()
	return structured[Encounter](t, callTool(t, cs, "get_encounter", map[string]any{"encounter_id": id}))
}

func combatantIDs(e Encounter) []any {
	ids := make([]any, 0, len(e.Combatants))
	for _, c := range e.Combatants {
		ids = append(ids, c.CombatantID)
	}
	return ids
}

func combatantNamed(t *testing.T, e Encounter, name string) Combatant {
	t.Helper()
	for _, c := range e.Combatants {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no combatant %s in %s", name, e.Name)
	return Combatant{}
}

func TestCreateEncounterMatchesDesktop(t *testing.T) {
	viaTool, direct := newTestEnv(t), newTestEnv(t)
	cs, ds := viaTool.client(t, clientOptions{}), direct.client(t, clientOptions{})
	raider := domain.Combatant{Name: "Raider", Level: 2, XP: 20, Initiative: 7, HP: 9, MaxHP: 9, ResistPhysicalTorso: 1}
	for _, env := range []*testEnv{viaTool, direct} {
		_, err := env.svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{raider})
		require.NoError(t, err)
	}
	monsters := []any{
		map[string]any{"template_id": raiderTemplate(t, viaTool.svc).ID, "quantity": 3},
		map[string]any{"monster": mirelurkInput, "quantity": 1, "initiative": 4},
	}
	party := []string{"pc-nora", "pc-dog"}

	preview := structured[Difficulty](t, callTool(t, cs, "preview_encounter_difficulty", map[string]any{"party_character_ids": party, "monsters": monsters}))
	require.Empty(t, structured[encounterList](t, callTool(t, cs, "list_encounters", nil)).Encounters, "a preview stores nothing")
	created := structured[Encounter](t, callTool(t, cs, "create_encounter", map[string]any{"name": "Ambush", "party_character_ids": party, "monsters": monsters}))
	require.Len(t, created.Combatants, 6)
	require.Equal(t, preview, created.Difficulty, "the preview matches the saved roster")

	// The desktop editor's rows for the same roster: party members loaded
	// from the campaign, three raider copies, and the mirelurk row.
	members, err := direct.svc.ListPartyMembers(t.Context())
	require.NoError(t, err)
	combatants := append([]domain.Combatant(nil), members...)
	tpl := raiderTemplate(t, direct.svc)
	tpl.ID = ""
	combatants = append(combatants, tpl, tpl, tpl)
	m := mirelurk()
	m.Initiative = 4
	combatants = append(combatants, m)
	enc, err := direct.svc.ExecuteCreateEncounter(t.Context(), app.CreateEncounterCommand{Name: "Ambush", Combatants: combatants})
	require.NoError(t, err)

	require.Equal(t, withoutIDs(encounterView(t, ds, enc.ID)), withoutIDs(encounterView(t, cs, created.EncounterID)))
	want := structured[encounterList](t, callTool(t, ds, "list_encounters", nil)).Encounters[0].Difficulty
	require.Equal(t, want, structured[encounterList](t, callTool(t, cs, "list_encounters", nil)).Encounters[0].Difficulty)
	require.Equal(t, want, preview, "the preview matches the desktop difficulty")

	entry := activityOutcomes(t, viaTool.svc)[0]
	require.Equal(t, "create_encounter", entry.Operation)
	require.Equal(t, domain.EntityEncounter, entry.EntityKind)
	require.Equal(t, created.EncounterID, entry.EntityID)
	require.Equal(t, domain.OutcomeDone, entry.Outcome)
}

func TestUpdateEncounterConfirmsOnlyRemovals(t *testing.T) {
	env := newTestEnv(t)
	enc := seedEncounter(t, env.svc)
	cs := env.client(t, clientOptions{})
	raiderA := combatantNamed(t, encounterView(t, cs, enc.ID), "Raider A").CombatantID
	_, _, err := env.svc.ApplyDamage(t.Context(), raiderA, domain.DamagePhysical, domain.BodyTorso, 4)
	require.NoError(t, err)
	_, err = env.svc.CreateEncounter(t.Context(), "", "Later", []domain.Combatant{{Name: "Bug", Side: domain.SideNPC, Level: 1, XP: 5, HP: 3, MaxHP: 3}})
	require.NoError(t, err)
	before := encounterView(t, cs, enc.ID)

	// A rename plus additions keeps every stored combatant: no confirmation.
	updated := structured[Encounter](t, callTool(t, cs, "update_encounter", map[string]any{
		"encounter_id": enc.ID, "name": "Ambush at Dawn", "keep_combatant_ids": combatantIDs(before),
		"add_monsters": []any{map[string]any{"template_id": raiderTemplate(t, env.svc).ID, "quantity": 2}},
	}))
	require.Empty(t, env.confirmer.seen(), "no removal, no confirmation")
	require.Equal(t, "Ambush at Dawn", updated.Name)
	require.Len(t, updated.Combatants, 5)
	require.Equal(t, 5, combatantNamed(t, updated, "Raider A").HP, "kept combatants keep their HP")
	active := structured[activeEncounter](t, callTool(t, cs, "get_active_encounter", nil))
	require.Equal(t, enc.ID, active.Encounter.EncounterID, "saving an encounter makes it active (research R20)")
	stored := encounterView(t, cs, enc.ID)

	// Dropping a stored combatant asks the GM; a decline changes nothing.
	keep := []any{}
	for _, c := range updated.Combatants {
		if c.CombatantID != raiderA {
			keep = append(keep, c.CombatantID)
		}
	}
	args := map[string]any{"encounter_id": enc.ID, "keep_combatant_ids": keep}
	env.confirmer.decideAll(app.ConfirmationDeclined)
	require.Equal(t, codeDeclined, toolErrorOf(t, callTool(t, cs, "update_encounter", args)).Code)
	require.Equal(t, stored, encounterView(t, cs, enc.ID), "a declined update changes nothing")
	seen := env.confirmer.seen()
	require.Len(t, seen, 1)
	require.Equal(t, app.ConfirmationRequest{
		Operation: "update_encounter", EntityKind: domain.EntityEncounter, EntityID: enc.ID, EntityLabel: "Ambush at Dawn",
		Summary: "Update encounter «Ambush at Dawn»: remove 1 combatant(s) (Raider A)?",
	}, seen[0])

	env.confirmer.decideAll(app.ConfirmationAccepted)
	trimmed := structured[Encounter](t, callTool(t, cs, "update_encounter", args))
	require.Len(t, trimmed.Combatants, 4)
	require.NotContains(t, combatantIDs(trimmed), raiderA)
	require.Len(t, env.confirmer.seen(), 2)

	// An unknown kept combatant is refused before anyone is asked.
	te := toolErrorOf(t, callTool(t, cs, "update_encounter", map[string]any{"encounter_id": enc.ID, "keep_combatant_ids": []any{"missing"}}))
	require.Equal(t, codeNotFound, te.Code)
	require.Len(t, env.confirmer.seen(), 2)

	require.Equal(t, []domain.AssistantOutcome{
		domain.OutcomeRejected, domain.OutcomeDone, domain.OutcomeDeclined, domain.OutcomeDone,
	}, outcomes(t, env.svc), "one entry per call, newest first")

	// In conversation, only a removal returns input_required.
	mrtr := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return nil }, manualRetry: true})
	all := map[string]any{"encounter_id": enc.ID, "keep_combatant_ids": combatantIDs(trimmed)}
	require.False(t, callTool(t, mrtr, "update_encounter", all).NeedsInput())
	require.True(t, callTool(t, mrtr, "update_encounter", map[string]any{"encounter_id": enc.ID, "keep_combatant_ids": combatantIDs(trimmed)[1:]}).NeedsInput())
}

func TestUpdateEncounterRefusesRosterChangedDuringConfirmation(t *testing.T) {
	env := newTestEnv(t)
	enc := seedEncounter(t, env.svc)
	cs := env.client(t, clientOptions{})
	env.confirmer.mu.Lock()
	env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision {
		// The desktop adds a combatant while the GM reads the dialog.
		stored, err := env.svc.GetEncounterByID(t.Context(), enc.ID)
		if err == nil {
			bug := domain.Combatant{Name: "Bug", Side: domain.SideNPC, Level: 1, XP: 5, HP: 3, MaxHP: 3}
			_, err = env.svc.UpdateEncounter(t.Context(), enc.ID, stored.Name, append(stored.Combatants, bug))
		}
		if err != nil {
			t.Error(err)
		}
		return app.ConfirmationAccepted
	}
	env.confirmer.mu.Unlock()

	ids := combatantIDs(encounterView(t, cs, enc.ID))
	te := toolErrorOf(t, callTool(t, cs, "update_encounter", map[string]any{"encounter_id": enc.ID, "keep_combatant_ids": ids[1:]}))
	require.Equal(t, codeInvalidInput, te.Code)
	require.Equal(t, "keep_combatant_ids", te.Field)
	after := encounterView(t, cs, enc.ID)
	require.Len(t, after.Combatants, 4, "neither the confirmed nor the unconfirmed combatant is removed")
	require.Equal(t, domain.OutcomeRejected, activityOutcomes(t, env.svc)[0].Outcome)
}

func TestRestartAndDeleteEncounterAlwaysConfirm(t *testing.T) {
	env := newTestEnv(t)
	enc := seedEncounter(t, env.svc)
	cs := env.client(t, clientOptions{})
	view := encounterView(t, cs, enc.ID)
	nora, raiderA := combatantNamed(t, view, "Nora").CombatantID, combatantNamed(t, view, "Raider A").CombatantID
	for range len(view.Combatants) {
		structured[turnResult](t, callTool(t, cs, "advance_turn", nil))
	}
	damage(t, cs, nora, "physical", "torso", 12)
	damage(t, cs, raiderA, "energy", "head", 9)
	before := encounterView(t, cs, enc.ID)
	require.Equal(t, 2, before.Round)
	ref := map[string]any{"encounter_id": enc.ID}

	env.confirmer.decideAll(app.ConfirmationDeclined)
	require.Equal(t, codeDeclined, toolErrorOf(t, callTool(t, cs, "restart_encounter", ref)).Code)
	require.Equal(t, codeDeclined, toolErrorOf(t, callTool(t, cs, "delete_encounter", ref)).Code)
	require.Equal(t, before, encounterView(t, cs, enc.ID), "declined confirmations change nothing")
	seen := env.confirmer.seen()
	require.Len(t, seen, 2)
	require.Equal(t, "Restart encounter «Ambush»? NPC HP resets to max and the round goes back to 1.", seen[0].Summary)
	require.Equal(t, "Delete encounter «Ambush» (round 2, 3 combatants)?", seen[1].Summary)
	require.Equal(t, app.ConfirmationRequest{
		Operation: "delete_encounter", EntityKind: domain.EntityEncounter, EntityID: enc.ID, EntityLabel: "Ambush", Summary: seen[1].Summary,
	}, seen[1])

	env.confirmer.decideAll(app.ConfirmationAccepted)
	restarted := structured[Encounter](t, callTool(t, cs, "restart_encounter", ref))
	require.Equal(t, 1, restarted.Round)
	for _, c := range restarted.Combatants {
		if c.Side == string(domain.SideNPC) {
			require.Equal(t, c.MaxHP, c.HP, c.Name)
			require.False(t, c.Defeated, c.Name)
		}
	}
	gotNora := combatantNamed(t, restarted, "Nora")
	require.Zero(t, gotNora.HP, "party HP is not reset")
	require.True(t, gotNora.Defeated, "party defeated state is not reset")

	require.Equal(t, encounterDeleted{EncounterID: enc.ID, Name: "Ambush"}, structured[encounterDeleted](t, callTool(t, cs, "delete_encounter", ref)))
	require.Len(t, env.confirmer.seen(), 4)
	require.Empty(t, structured[encounterList](t, callTool(t, cs, "list_encounters", nil)).Encounters, "deleted encounters are not listed")
	for _, tool := range []string{"get_encounter", "delete_encounter", "restart_encounter", "activate_encounter"} {
		require.Equal(t, codeNotFound, toolErrorOf(t, callTool(t, cs, tool, ref)).Code, tool)
	}
	require.Len(t, env.confirmer.seen(), 4, "nothing is confirmed for a deleted encounter")

	var ops []string
	for _, e := range activityOutcomes(t, env.svc)[:7] {
		ops = append(ops, e.Operation+":"+string(e.Outcome))
	}
	require.Equal(t, []string{
		"activate_encounter:rejected", "restart_encounter:rejected", "delete_encounter:rejected",
		"delete_encounter:done", "restart_encounter:done", "delete_encounter:declined", "restart_encounter:declined",
	}, ops)
}

func TestSaveMonsterTemplateReportsReplacement(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})

	first := structured[templateSaved](t, callTool(t, cs, "save_monster_template", map[string]any{"monster": mirelurkInput}))
	require.False(t, first.ReplacedExisting)
	require.Equal(t, monsterTemplateDTO(func() domain.Combatant { m := mirelurk(); m.ID = first.Template.TemplateID; return m }()), first.Template)

	again := map[string]any{}
	for k, v := range mirelurkInput {
		again[k] = v
	}
	again["name"], again["hp"], again["max_hp"] = " MIRELURK ", 20, 20
	second := structured[templateSaved](t, callTool(t, cs, "save_monster_template", map[string]any{"monster": again}))
	require.True(t, second.ReplacedExisting, "names match case-insensitively")
	require.Equal(t, first.Template.TemplateID, second.Template.TemplateID)
	templates := structured[templateList](t, callTool(t, cs, "list_monster_templates", nil)).Templates
	require.Len(t, templates, 1)
	require.Equal(t, 20, templates[0].MaxHP)

	entries := activityOutcomes(t, env.svc)
	require.Len(t, entries, 2)
	require.Equal(t, domain.EntityMonsterTemplate, entries[0].EntityKind)
	require.Equal(t, first.Template.TemplateID, entries[0].EntityID)
	require.Equal(t, "replaced the existing template", entries[0].Detail)
	require.Empty(t, entries[1].Detail)

	te := toolErrorOf(t, callTool(t, cs, "save_monster_template", map[string]any{"monster": map[string]any{
		"name": "Bad", "level": 1, "xp": 0, "initiative": 1, "hp": 9, "max_hp": 3, "defense": 0,
		"resistance": mirelurkInput["resistance"], "immunity": mirelurkInput["immunity"],
	}}))
	require.Equal(t, codeInvalidInput, te.Code)
	require.Equal(t, domain.OutcomeRejected, activityOutcomes(t, env.svc)[0].Outcome)
}

func TestActivateEncounterLogsAssistantSource(t *testing.T) {
	env := newTestEnv(t)
	enc := seedEncounter(t, env.svc)
	_, err := env.svc.CreateEncounter(t.Context(), "", "Later", []domain.Combatant{{Name: "Bug", Side: domain.SideNPC, Level: 1, XP: 5, HP: 3, MaxHP: 3}})
	require.NoError(t, err)
	cs := env.client(t, clientOptions{})

	got := structured[Encounter](t, callTool(t, cs, "activate_encounter", map[string]any{"encounter_id": enc.ID}))
	require.Equal(t, enc.ID, got.EncounterID)
	require.Equal(t, enc.ID, structured[activeEncounter](t, callTool(t, cs, "get_active_encounter", nil)).Encounter.EncounterID)
	entries := structured[logList](t, callTool(t, cs, "get_encounter_log", map[string]any{"encounter_id": enc.ID})).Entries
	require.Equal(t, "Encounter activated", entries[0].Message)
	require.Equal(t, string(domain.SourceAssistant), entries[0].Source)

	entry := activityOutcomes(t, env.svc)[0]
	require.Equal(t, "activate_encounter", entry.Operation)
	require.Equal(t, enc.ID, entry.EntityID)
	require.Equal(t, "Ambush", entry.EntityLabel)
	require.Equal(t, domain.OutcomeDone, entry.Outcome)
}

func TestEncounterToolsRejectInvalidInput(t *testing.T) {
	env := newTestEnv(t)
	_, err := env.svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{{Name: "Raider", Level: 2, XP: 20, Initiative: 7, HP: 9, MaxHP: 9}})
	require.NoError(t, err)
	raider := raiderTemplate(t, env.svc).ID
	cs := env.client(t, clientOptions{})
	create := func(party []string, monsters ...any) map[string]any {
		if party == nil {
			party = []string{}
		}
		return map[string]any{"name": "Draft", "party_character_ids": party, "monsters": append([]any{}, monsters...)}
	}

	for _, tc := range []struct {
		name, code, field, mentions string
		args                        map[string]any
	}{
		{"zero quantity", codeInvalidInput, "monsters[0].quantity", "between 1 and 50", create(nil, map[string]any{"template_id": raider, "quantity": 0})},
		{"quantity above 50", codeInvalidInput, "monsters[1].quantity", "between 1 and 50", create(nil, map[string]any{"template_id": raider, "quantity": 1}, map[string]any{"template_id": raider, "quantity": 51})},
		{"template and monster", codeInvalidInput, "monsters[0]", "exactly one of template_id or monster", create(nil, map[string]any{"template_id": raider, "monster": mirelurkInput, "quantity": 1})},
		{"neither template nor monster", codeInvalidInput, "monsters[0]", "exactly one of template_id or monster", create(nil, map[string]any{"quantity": 1})},
		{"unknown template", codeNotFound, "", "missing", create(nil, map[string]any{"template_id": "missing", "quantity": 1})},
		{"unknown party character", codeNotFound, "", "missing", create([]string{"missing"})},
		{"empty roster", codeInvalidInput, "", "cannot create encounter without combatants", create(nil)},
		{"inline monster hp above max hp", codeInvalidInput, "", "HP", create(nil, map[string]any{"monster": func() map[string]any {
			m := map[string]any{}
			for k, v := range mirelurkInput {
				m[k] = v
			}
			m["hp"] = 99
			return m
		}(), "quantity": 1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := toolErrorOf(t, callTool(t, cs, "create_encounter", tc.args))
			require.Equal(t, tc.code, te.Code)
			require.Equal(t, tc.field, te.Field)
			require.Contains(t, te.Message, tc.mentions)
			require.Equal(t, domain.OutcomeRejected, activityOutcomes(t, env.svc)[0].Outcome)
			if tc.name != "empty roster" && tc.name != "inline monster hp above max hp" {
				delete(tc.args, "name")
				require.Equal(t, tc.code, toolErrorOf(t, callTool(t, cs, "preview_encounter_difficulty", tc.args)).Code, "the preview expands the same way")
			}
		})
	}
	te := toolErrorOf(t, callTool(t, cs, "create_encounter", map[string]any{"name": " ", "party_character_ids": []string{"pc-nora"}, "monsters": []any{}}))
	require.Equal(t, codeInvalidInput, te.Code)
	require.Equal(t, "name", te.Field)

	require.Empty(t, structured[encounterList](t, callTool(t, cs, "list_encounters", nil)).Encounters, "nothing is stored")
	require.Len(t, activityOutcomes(t, env.svc), 9, "previews record no activity")
	require.Empty(t, env.confirmer.seen())
}

func TestEncounterToolAnnotations(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	tools := map[string]*mcp.Tool{}
	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		tools[tool.Name] = tool
	}
	want := map[string]struct{ readOnly, destructive, idempotent bool }{
		"preview_encounter_difficulty": {true, false, true},
		"create_encounter":             {false, false, false},
		"activate_encounter":           {false, false, true},
		"save_monster_template":        {false, false, true},
		"update_encounter":             {false, true, false},
		"restart_encounter":            {false, true, false},
		"delete_encounter":             {false, true, false},
	}
	require.Len(t, want, len(encounterTools))
	for _, name := range encounterTools {
		tool, ok := tools[name]
		require.True(t, ok, name)
		a := tool.Annotations
		require.Equal(t, want[name].readOnly, a.ReadOnlyHint, name)
		require.NotNil(t, a.DestructiveHint, name)
		require.Equal(t, want[name].destructive, *a.DestructiveHint, name)
		require.Equal(t, want[name].idempotent, a.IdempotentHint, name)
		require.NotNil(t, a.OpenWorldHint, name)
		require.False(t, *a.OpenWorldHint, name)
		require.NotNil(t, tool.OutputSchema, name)
	}

	difficulty := []string{"floor((XP baseline − 10) ÷ 10)", "rounded up", "below -2 Trivial", "above 5 Deadly", "Torso-only", "poison DR is global"}
	for name, terms := range map[string][]string{
		"preview_encounter_difficulty": append([]string{"without saving anything"}, difficulty...),
		"create_encounter":             append([]string{"Saving an encounter makes it the active encounter (tracker behavior)."}, difficulty...),
		"update_encounter":             {"Saving an encounter makes it the active encounter (tracker behavior).", "keep their current HP and defeated state", "confirm"},
		"restart_encounter":            {"Only NPC HP", "defeated state", "party members keep", "confirm"},
		"delete_encounter":             {"confirm"},
		"save_monster_template":        {"case-insensitive", "replaced_existing", "torso", "per body location"},
	} {
		for _, term := range terms {
			require.Contains(t, tools[name].Description, term, name)
		}
	}
	for _, name := range []string{"preview_encounter_difficulty", "create_encounter", "update_encounter", "save_monster_template"} {
		schema, err := json.Marshal(tools[name].InputSchema)
		require.NoError(t, err)
		require.Contains(t, string(schema), "tracked on the torso only", name)
		require.Contains(t, string(schema), "DR per body location", name)
	}
}
