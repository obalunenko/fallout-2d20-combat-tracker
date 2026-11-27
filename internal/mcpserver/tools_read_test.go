package mcpserver

import (
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

var readTools = []string{
	"list_campaigns", "get_active_campaign", "list_campaign_characters", "list_party_members",
	"list_encounters", "get_encounter", "get_active_encounter", "get_encounter_log", "list_monster_templates",
}

// seedEncounter creates an active encounter in the active campaign with Nora
// and two raiders, plus a saved raider template.
func seedEncounter(t *testing.T, svc *app.Service) *domain.Encounter {
	t.Helper()
	_, err := svc.SaveMonsterTemplates(t.Context(), []domain.Combatant{{Name: "Raider", Level: 2, XP: 20, Initiative: 7, HP: 9, MaxHP: 9, ResistPhysicalTorso: 1}})
	require.NoError(t, err)
	enc, err := svc.CreateEncounter(t.Context(), "", "Ambush", []domain.Combatant{
		{ID: "pc-nora", PlayerCharacterID: "pc-nora", Name: "Nora", Side: domain.SideParty, Level: 3, Initiative: 9, HP: 12, MaxHP: 12},
		{Name: "Raider A", Side: domain.SideNPC, Level: 2, XP: 20, Initiative: 7, HP: 9, MaxHP: 9},
		{Name: "Raider B", Side: domain.SideNPC, Level: 2, XP: 20, Initiative: 12, HP: 9, MaxHP: 9},
	})
	require.NoError(t, err)
	return enc
}

func TestReadToolsReturnTrackerData(t *testing.T) {
	env := newTestEnv(t)
	start, err := domain.ParseCampaignStartDate("2026-12-01")
	require.NoError(t, err)
	_, err = env.svc.CreateCampaign(t.Context(), "camp-2", "Пустошь", start, []domain.NewCampaignPlayer{
		{PlayerName: "Вера", Notes: "Строка 1\nСтрока 2", Character: domain.Combatant{ID: "pc-vera", Name: "Вера", Side: domain.SideParty, Level: 1, HP: 5, MaxHP: 5}},
	})
	require.NoError(t, err)
	_, err = env.svc.ActivateCampaign(t.Context(), "camp-1")
	require.NoError(t, err)
	enc := seedEncounter(t, env.svc)
	cs := env.client(t, clientOptions{})

	campaigns := structured[campaignList](t, callTool(t, cs, "list_campaigns", nil))
	require.Len(t, campaigns.Campaigns, 2)
	for _, c := range campaigns.Campaigns {
		require.Equal(t, c.CampaignID == "camp-1", c.IsActive, c.Name)
	}

	active := structured[activeCampaign](t, callTool(t, cs, "get_active_campaign", nil))
	require.True(t, active.Initialized)
	require.Equal(t, "camp-1", active.Campaign.CampaignID)

	chars := structured[characterList](t, callTool(t, cs, "list_campaign_characters", nil))
	require.Equal(t, "camp-1", chars.CampaignID, "defaults to the active campaign")
	require.Len(t, chars.Characters, 2)
	require.Equal(t, 1, chars.Characters[0].Special.Strength, "default S.P.E.C.I.A.L. is 1")
	require.True(t, chars.Characters[0].Available)

	other := structured[characterList](t, callTool(t, cs, "list_campaign_characters", map[string]any{"campaign_id": "camp-2"}))
	require.Equal(t, "Вера", other.Characters[0].Name, "Cyrillic names round-trip")
	require.Equal(t, "Строка 1\nСтрока 2", other.Characters[0].Notes, "line breaks round-trip")

	party := structured[partyList](t, callTool(t, cs, "list_party_members", nil))
	require.Len(t, party.Members, 2)

	summaries, err := env.svc.ListEncounters(t.Context())
	require.NoError(t, err)
	list := structured[encounterList](t, callTool(t, cs, "list_encounters", nil))
	require.Len(t, list.Encounters, len(summaries))
	require.Equal(t, encounterSummaryDTO(summaries[0]).Difficulty, list.Encounters[0].Difficulty, "same difficulty as the desktop list")

	got := structured[Encounter](t, callTool(t, cs, "get_encounter", map[string]any{"encounter_id": enc.ID}))
	require.Len(t, got.Combatants, 3)
	require.Equal(t, "Raider B", got.Combatants[0].Name, "initiative order")
	require.True(t, got.Combatants[got.TurnIndex].IsCurrentTurn)
	require.Equal(t, got.Combatants[got.TurnIndex].CombatantID, got.CurrentCombatantID)
	require.Equal(t, "pc-nora", got.Combatants[1].CharacterID)

	act := structured[activeEncounter](t, callTool(t, cs, "get_active_encounter", nil))
	require.Equal(t, enc.ID, act.Encounter.EncounterID)

	_, err = env.svc.ActivateEncounter(app.WithSource(t.Context(), domain.SourceAssistant), enc.ID)
	require.NoError(t, err)
	logs := structured[logList](t, callTool(t, cs, "get_encounter_log", map[string]any{"encounter_id": enc.ID}))
	require.Equal(t, "Encounter activated", logs.Entries[0].Message, "newest first")
	require.Equal(t, "assistant", logs.Entries[0].Source)
	require.Equal(t, "desktop", logs.Entries[len(logs.Entries)-1].Source)
	limited := structured[logList](t, callTool(t, cs, "get_encounter_log", map[string]any{"encounter_id": enc.ID, "limit": 1}))
	require.Len(t, limited.Entries, 1)

	templates := structured[templateList](t, callTool(t, cs, "list_monster_templates", nil))
	require.Len(t, templates.Templates, 1)
	require.Equal(t, 1, templates.Templates[0].Resistance.Physical.Torso)

	require.Equal(t, codeNotFound, toolErrorOf(t, callTool(t, cs, "get_encounter", map[string]any{"encounter_id": "missing"})).Code)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "get_encounter_log", map[string]any{"encounter_id": enc.ID, "limit": 501})).Code)
	require.Equal(t, codeInvalidInput, toolErrorOf(t, callTool(t, cs, "get_encounter", map[string]any{"encounter_id": enc.ID, "extra": 1})).Code, "unknown arguments are rejected")

	require.Empty(t, activityOutcomes(t, env.svc), "read tools write no activity")
}

func TestReadToolsOnEmptyTracker(t *testing.T) {
	env := newTestEnvWith(t, newEmptyService(t))
	cs := env.client(t, clientOptions{})

	active := structured[activeCampaign](t, callTool(t, cs, "get_active_campaign", nil))
	require.False(t, active.Initialized, "no campaign yet is a result, not an error")
	require.Nil(t, active.Campaign)

	require.Nil(t, structured[activeEncounter](t, callTool(t, cs, "get_active_encounter", nil)).Encounter)
	require.Empty(t, structured[campaignList](t, callTool(t, cs, "list_campaigns", nil)).Campaigns)
	require.Equal(t, codeNoCampaign, toolErrorOf(t, callTool(t, cs, "list_campaign_characters", nil)).Code)
}

func TestReadToolAnnotations(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	tools := map[string]*mcp.Tool{}
	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		tools[tool.Name] = tool
	}
	for _, name := range readTools {
		tool, ok := tools[name]
		require.True(t, ok, name)
		a := tool.Annotations
		require.True(t, a.ReadOnlyHint, name)
		require.True(t, a.IdempotentHint, name)
		require.NotNil(t, a.DestructiveHint, name)
		require.False(t, *a.DestructiveHint, name)
		require.NotNil(t, a.OpenWorldHint, name)
		require.False(t, *a.OpenWorldHint, name)
		require.NotEmpty(t, tool.Description, name)
		require.NotNil(t, tool.OutputSchema, name)
	}
}

// schemaProperty finds the first property called name anywhere in schema.
func schemaProperty(t *testing.T, schema any, name string) map[string]any {
	t.Helper()
	b, err := json.Marshal(schema)
	require.NoError(t, err)
	var root map[string]any
	require.NoError(t, json.Unmarshal(b, &root))
	var find func(map[string]any) map[string]any
	find = func(s map[string]any) map[string]any {
		props, _ := s["properties"].(map[string]any)
		if p, ok := props[name].(map[string]any); ok {
			return p
		}
		for _, p := range props {
			if m, ok := p.(map[string]any); ok {
				if found := find(m); found != nil {
					return found
				}
			}
		}
		if items, ok := s["items"].(map[string]any); ok {
			return find(items)
		}
		return nil
	}
	p := find(root)
	require.NotNil(t, p, "property %s", name)
	return p
}

func TestToolSchemasCarryDomainBounds(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	tools := map[string]*mcp.Tool{}
	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		tools[tool.Name] = tool
	}

	quantity := schemaProperty(t, tools["create_encounter"].InputSchema, "quantity")
	require.InDelta(t, 1, quantity["minimum"], 0)
	require.InDelta(t, app.MaxDraftMonsterQuantity, quantity["maximum"], 0)
	require.InDelta(t, 1, schemaProperty(t, tools["create_encounter"].InputSchema, "max_hp")["minimum"], 0)
	require.InDelta(t, 0, schemaProperty(t, tools["apply_damage"].InputSchema, "amount")["minimum"], 0)
	require.InDelta(t, 1, schemaProperty(t, tools["apply_damage"].InputSchema, "combatant_id")["minLength"], 0)
	require.InDelta(t, 1, schemaProperty(t, tools["list_assistant_activity"].InputSchema, "limit")["minimum"], 0)
	keep := schemaProperty(t, tools["update_encounter"].InputSchema, "keep_combatant_ids")
	require.InDelta(t, 1, keep["items"].(map[string]any)["minLength"], 0)
	require.NotContains(t, schemaProperty(t, tools["update_encounter"].InputSchema, "initiative"), "minimum", "unbounded fields stay unbounded")

	require.ElementsMatch(t, []any{"party", "npc"}, schemaProperty(t, tools["get_encounter"].OutputSchema, "side")["enum"])
	require.ElementsMatch(t, []any{"desktop", "assistant"}, schemaProperty(t, tools["get_encounter_log"].OutputSchema, "source")["enum"])
	require.NotContains(t, schemaProperty(t, tools["get_encounter"].OutputSchema, "hp"), "minimum", "outputs carry no bounds")
}
