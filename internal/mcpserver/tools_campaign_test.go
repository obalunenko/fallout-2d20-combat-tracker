package mcpserver

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

var campaignTools = []string{
	"create_campaign", "update_campaign_details", "add_campaign_player", "update_campaign_character",
	"mark_character_available", "mark_character_unavailable", "activate_campaign", "remove_campaign_player",
}

// rosterView is list_campaign_characters without character IDs.
func rosterView(t *testing.T, cs *mcp.ClientSession, campaignID string) []Character {
	t.Helper()
	list := structured[characterList](t, callTool(t, cs, "list_campaign_characters", map[string]any{"campaign_id": campaignID}))
	for i := range list.Characters {
		list.Characters[i].CharacterID = ""
	}
	return list.Characters
}

// campaignsView is list_campaigns without IDs and timestamps.
func campaignsView(t *testing.T, cs *mcp.ClientSession) []Campaign {
	t.Helper()
	list := structured[campaignList](t, callTool(t, cs, "list_campaigns", nil))
	for i := range list.Campaigns {
		list.Campaigns[i].CampaignID, list.Campaigns[i].UpdatedAt = "", time.Time{}
	}
	return list.Campaigns
}

// updateDirect makes a change through ExecuteUpdateCampaign, as the desktop
// CAMP screen does.
func updateDirect(t *testing.T, svc *app.Service, edit func(cmd *app.UpdateCampaignCommand)) {
	t.Helper()
	campaigns, err := svc.ListCampaigns(t.Context())
	require.NoError(t, err)
	cmd := app.UpdateCampaignCommand{CampaignID: "camp-1", Players: mustPlayers(t, svc)}
	for _, c := range campaigns {
		if c.ID == "camp-1" {
			cmd.Name, cmd.StartDate = c.Name, c.StartDate
		}
	}
	edit(&cmd)
	_, err = svc.ExecuteUpdateCampaign(t.Context(), cmd)
	require.NoError(t, err)
}

func playerIndex(t *testing.T, players []domain.NewCampaignPlayer, id string) int {
	t.Helper()
	for i, p := range players {
		if p.Character.ID == id {
			return i
		}
	}
	t.Fatalf("character %s not found", id)
	return -1
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := domain.ParseCampaignStartDate(s)
	require.NoError(t, err)
	return d
}

var (
	roseInput = map[string]any{
		"name": "Rose", "level": 4, "initiative": 10, "hp": 15, "max_hp": 15, "defense": 2,
		"resistance": map[string]any{
			"physical":  map[string]any{"head": 1, "torso": 2, "left_arm": 1, "right_arm": 1, "left_leg": 1, "right_leg": 1},
			"energy":    map[string]any{"head": 0, "torso": 1, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
			"radiation": map[string]any{"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
			"poison":    3,
		},
		"immunity": map[string]any{"physical": false, "energy": false, "radiation": true, "poison": false},
	}
	roseSpecialInput = map[string]any{"strength": 5, "perception": 6, "endurance": 4, "charisma": 7, "intelligence": 8, "agility": 5, "luck": 6}
)

func roseCharacter() domain.Combatant {
	return domain.Combatant{
		Name: "Rose", Side: domain.SideParty, Level: 4, Initiative: 10, HP: 15, MaxHP: 15, Defense: 2,
		ResistPhysicalHead: 1, ResistPhysicalTorso: 2, ResistPhysicalLeftArm: 1, ResistPhysicalRightArm: 1, ResistPhysicalLeftLeg: 1, ResistPhysicalRightLeg: 1,
		ResistEnergyTorso: 1, ResistPoison: 3, ImmuneRadiation: true,
	}
}

var roseSpecial = domain.SpecialValues{Strength: 5, Perception: 6, Endurance: 4, Charisma: 7, Intelligence: 8, Agility: 5, Luck: 6}

func TestCampaignToolsMatchDesktopUpdates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		setup  func(t *testing.T, svc *app.Service)
		tool   string
		args   map[string]any
		direct func(t *testing.T, svc *app.Service)
	}{
		{
			name: "update details",
			tool: "update_campaign_details",
			args: map[string]any{"campaign_id": "camp-1", "name": "Capital Wasteland", "start_date": "2027-01-02"},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					cmd.Name, cmd.StartDate = "Capital Wasteland", mustDate(t, "2027-01-02")
				})
			},
		},
		{
			name: "add player",
			tool: "add_campaign_player",
			args: map[string]any{"campaign_id": "camp-1", "player_name": "Cass", "character": roseInput, "notes": "Whiskey\nRose", "special": roseSpecialInput},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					cmd.Players = append(cmd.Players, domain.NewCampaignPlayer{PlayerName: "Cass", Character: roseCharacter(), Notes: "Whiskey\nRose", Special: roseSpecial})
				})
			},
		},
		{
			name: "update character",
			tool: "update_campaign_character",
			args: map[string]any{"campaign_id": "camp-1", "character_id": "pc-nora", "patch": map[string]any{
				"level": 5, "hp": 10, "max_hp": 14, "special": roseSpecialInput, "notes": "Line 1\nLine 2",
				"immunity": map[string]any{"physical": false, "energy": true, "radiation": false, "poison": false},
			}},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					p := &cmd.Players[playerIndex(t, cmd.Players, "pc-nora")]
					p.Character.Level, p.Character.HP, p.Character.MaxHP, p.Character.ImmuneEnergy = 5, 10, 14, true
					p.Special, p.Notes = roseSpecial, "Line 1\nLine 2"
				})
			},
		},
		{
			name: "rename character",
			tool: "update_campaign_character",
			args: map[string]any{"campaign_id": "camp-1", "character_id": "pc-nora", "patch": map[string]any{"name": "Nora Prime"}},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					p := &cmd.Players[playerIndex(t, cmd.Players, "pc-nora")]
					p.Character.Name, p.Character.ID = "Nora Prime", ""
				})
			},
		},
		{
			name: "mark unavailable",
			tool: "mark_character_unavailable",
			args: map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					cmd.Players[playerIndex(t, cmd.Players, "pc-dog")].Inactive = true
				})
			},
		},
		{
			name: "mark available",
			setup: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					cmd.Players[playerIndex(t, cmd.Players, "pc-dog")].Inactive = true
				})
			},
			tool: "mark_character_available",
			args: map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					cmd.Players[playerIndex(t, cmd.Players, "pc-dog")].Inactive = false
				})
			},
		},
		{
			name: "remove player",
			tool: "remove_campaign_player",
			args: map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"},
			direct: func(t *testing.T, svc *app.Service) {
				updateDirect(t, svc, func(cmd *app.UpdateCampaignCommand) {
					i := playerIndex(t, cmd.Players, "pc-dog")
					cmd.Players = append(cmd.Players[:i], cmd.Players[i+1:]...)
				})
			},
		},
		{
			name: "activate campaign",
			setup: func(t *testing.T, svc *app.Service) {
				_, err := svc.CreateCampaign(t.Context(), "camp-2", "Mojave", mustDate(t, "2027-02-03"), []domain.NewCampaignPlayer{
					{PlayerName: "Cass", Character: roseCharacter()},
				})
				require.NoError(t, err)
				_, err = svc.ActivateCampaign(t.Context(), "camp-1")
				require.NoError(t, err)
			},
			tool: "activate_campaign",
			args: map[string]any{"campaign_id": "camp-2"},
			direct: func(t *testing.T, svc *app.Service) {
				_, err := svc.ActivateCampaign(t.Context(), "camp-2")
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			viaTool, direct := newTestEnv(t), newTestEnv(t)
			if tc.setup != nil {
				tc.setup(t, viaTool.svc)
				tc.setup(t, direct.svc)
			}
			cs, ds := viaTool.client(t, clientOptions{}), direct.client(t, clientOptions{})

			res := callTool(t, cs, tc.tool, tc.args)
			require.False(t, res.IsError, resultText(res))
			tc.direct(t, direct.svc)

			require.Equal(t, rosterView(t, ds, "camp-1"), rosterView(t, cs, "camp-1"))
			require.Equal(t, campaignsView(t, ds), campaignsView(t, cs))
			require.Equal(t,
				structured[activeCampaign](t, callTool(t, ds, "get_active_campaign", nil)).Campaign.Name,
				structured[activeCampaign](t, callTool(t, cs, "get_active_campaign", nil)).Campaign.Name)
		})
	}
}

func TestCreateCampaignMatchesDesktop(t *testing.T) {
	viaTool, direct := newTestEnv(t), newTestEnv(t)
	cs, ds := viaTool.client(t, clientOptions{}), direct.client(t, clientOptions{})

	created := structured[campaignDetail](t, callTool(t, cs, "create_campaign", map[string]any{
		"name": "Vault 13", "start_date": "2027-05-05",
		"players": []any{
			map[string]any{"player_name": "Вера", "character": roseInput, "notes": "Строка 1\nСтрока 2", "special": roseSpecialInput},
			map[string]any{"player_name": "Dan", "character": map[string]any{"name": "Butch", "level": 1, "initiative": 6, "hp": 5, "max_hp": 5, "defense": 0}, "available": false},
		},
	}))
	require.NotEmpty(t, created.CampaignID)
	require.Equal(t, "Vault 13", created.Name)
	require.Equal(t, "2027-05-05", created.StartDate)
	require.Len(t, created.Characters, 2)

	c, err := direct.svc.ExecuteCreateCampaign(t.Context(), app.CreateCampaignCommand{Name: "Vault 13", StartDate: mustDate(t, "2027-05-05"), Players: []domain.NewCampaignPlayer{
		{PlayerName: "Вера", Character: roseCharacter(), Notes: "Строка 1\nСтрока 2", Special: roseSpecial},
		{PlayerName: "Dan", Character: domain.Combatant{Name: "Butch", Side: domain.SideParty, Level: 1, Initiative: 6, HP: 5, MaxHP: 5}, Inactive: true},
	}})
	require.NoError(t, err)

	require.Equal(t, rosterView(t, ds, c.ID), rosterView(t, cs, created.CampaignID))
	require.Equal(t, campaignsView(t, ds), campaignsView(t, cs))
}

func TestCampaignToolsRejectInvalidInput(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	before := rosterView(t, cs, "camp-1")
	patch := func(p map[string]any) map[string]any {
		return map[string]any{"campaign_id": "camp-1", "character_id": "pc-nora", "patch": p}
	}

	for _, tc := range []struct {
		name, tool, mentions string
		args                 map[string]any
	}{
		{"hp above max hp", "update_campaign_character", "HP", patch(map[string]any{"hp": 99})},
		{"special below 1", "update_campaign_character", "strength", patch(map[string]any{"special": map[string]any{"strength": 0, "perception": 1, "endurance": 1, "charisma": 1, "intelligence": 1, "agility": 1, "luck": 1}})},
		{"negative DR", "update_campaign_character", "resistance", patch(map[string]any{"resistance": map[string]any{
			"physical":  map[string]any{"head": -1, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
			"energy":    map[string]any{"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
			"radiation": map[string]any{"head": 0, "torso": 0, "left_arm": 0, "right_arm": 0, "left_leg": 0, "right_leg": 0},
			"poison":    0,
		}})},
		{"empty patch", "update_campaign_character", "patch", patch(map[string]any{})},
		{"new player hp above max hp", "add_campaign_player", "HP", map[string]any{"campaign_id": "camp-1", "player_name": "Cass",
			"character": map[string]any{"name": "Rose", "level": 1, "initiative": 1, "hp": 20, "max_hp": 10, "defense": 0}}},
		{"duplicate player", "add_campaign_player", "duplicate", map[string]any{"campaign_id": "camp-1", "player_name": " ann ",
			"character": map[string]any{"name": "Other", "level": 1, "initiative": 1, "hp": 2, "max_hp": 2, "defense": 0}}},
		{"bad start date", "update_campaign_details", "start_date", map[string]any{"campaign_id": "camp-1", "start_date": "11/01/2026"}},
		{"new campaign hp above max hp", "create_campaign", "HP", map[string]any{"name": "Bad", "start_date": "2027-01-01", "players": []any{
			map[string]any{"player_name": "Cass", "character": map[string]any{"name": "Rose", "level": 1, "initiative": 1, "hp": 20, "max_hp": 10, "defense": 0}},
		}}},
		{"new campaign duplicate players in any script", "create_campaign", "duplicate", map[string]any{"name": "Twins", "start_date": "2027-01-01", "players": []any{
			map[string]any{"player_name": "Иван", "character": map[string]any{"name": "One", "level": 1, "initiative": 1, "hp": 2, "max_hp": 2, "defense": 0}},
			map[string]any{"player_name": "иван", "character": map[string]any{"name": "Two", "level": 1, "initiative": 1, "hp": 2, "max_hp": 2, "defense": 0}},
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			te := toolErrorOf(t, callTool(t, cs, tc.tool, tc.args))
			require.Equal(t, codeInvalidInput, te.Code)
			require.Contains(t, te.Message, tc.mentions, "the message names the field")
			require.Equal(t, domain.OutcomeRejected, activityOutcomes(t, env.svc)[0].Outcome)
		})
	}
	require.Equal(t, before, rosterView(t, cs, "camp-1"), "nothing is stored")
	require.Len(t, campaignsView(t, cs), 1)
	require.Equal(t, codeNotFound, toolErrorOf(t, callTool(t, cs, "mark_character_available", map[string]any{"campaign_id": "camp-1", "character_id": "missing"})).Code)
	require.Equal(t, codeNotFound, toolErrorOf(t, callTool(t, cs, "remove_campaign_player", map[string]any{"campaign_id": "missing", "character_id": "pc-dog"})).Code)
}

func TestCampaignToolsRename(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	got := structured[characterUpdated](t, callTool(t, cs, "update_campaign_character", map[string]any{
		"campaign_id": "camp-1", "character_id": "pc-nora", "patch": map[string]any{"name": "Nora Prime"},
	}))
	require.Equal(t, "pc-nora", got.ReplacedCharacterID)
	require.NotEqual(t, "pc-nora", got.Character.CharacterID)
	require.Equal(t, 3, got.Character.Level, "unchanged fields carry over")
	entry := activityOutcomes(t, env.svc)[0]
	require.Equal(t, "update_campaign_character", entry.Operation)
	require.Equal(t, domain.EntityPlayerCharacter, entry.EntityKind)
	require.Equal(t, got.Character.CharacterID, entry.EntityID)
	require.Equal(t, "Nora Prime (Ann)", entry.EntityLabel)
	require.Equal(t, domain.OutcomeDone, entry.Outcome)
}

func TestCampaignToolsConfirmOnlyDestructiveChanges(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	ref := map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog"}

	structured[characterResult](t, callTool(t, cs, "mark_character_unavailable", ref))
	require.Len(t, env.confirmer.seen(), 1, "mark_character_unavailable asks")
	structured[characterResult](t, callTool(t, cs, "mark_character_available", ref))
	structured[characterUpdated](t, callTool(t, cs, "update_campaign_character", map[string]any{"campaign_id": "camp-1", "character_id": "pc-dog", "patch": map[string]any{"level": 3}}))
	structured[Campaign](t, callTool(t, cs, "update_campaign_details", map[string]any{"campaign_id": "camp-1", "name": "Renamed"}))
	structured[Campaign](t, callTool(t, cs, "activate_campaign", map[string]any{"campaign_id": "camp-1"}))
	require.Len(t, env.confirmer.seen(), 1, "non-destructive tools do not ask")

	env.confirmer.mu.Lock()
	env.confirmer.decide = func(context.Context, app.ConfirmationRequest) app.ConfirmationDecision {
		return app.ConfirmationDeclined
	}
	env.confirmer.mu.Unlock()
	require.Equal(t, codeDeclined, toolErrorOf(t, callTool(t, cs, "remove_campaign_player", ref)).Code)
	require.Len(t, env.confirmer.seen(), 2, "remove_campaign_player asks")
	require.Len(t, rosterView(t, cs, "camp-1"), 2, "a declined removal changes nothing")

	require.Equal(t, []domain.AssistantOutcome{
		domain.OutcomeDeclined, domain.OutcomeDone, domain.OutcomeDone, domain.OutcomeDone, domain.OutcomeDone, domain.OutcomeDone,
	}, outcomes(t, env.svc), "one entry per call, newest first")

	// In conversation, only the destructive tool returns input_required.
	mrtr := env.client(t, clientOptions{elicit: func(*mcp.ElicitRequest) *mcp.ElicitResult { return nil }, manualRetry: true})
	require.False(t, callTool(t, mrtr, "mark_character_available", ref).NeedsInput())
	require.True(t, callTool(t, mrtr, "remove_campaign_player", ref).NeedsInput())
}

func TestCampaignToolAnnotations(t *testing.T) {
	env := newTestEnv(t)
	cs := env.client(t, clientOptions{})
	tools := map[string]*mcp.Tool{}
	for tool, err := range cs.Tools(t.Context(), nil) {
		require.NoError(t, err)
		tools[tool.Name] = tool
	}
	want := map[string]struct{ destructive, idempotent bool }{
		"create_campaign":            {false, false},
		"update_campaign_details":    {false, true},
		"add_campaign_player":        {false, false},
		"update_campaign_character":  {false, true},
		"mark_character_available":   {false, true},
		"activate_campaign":          {false, true},
		"mark_character_unavailable": {true, false},
		"remove_campaign_player":     {true, false},
	}
	require.Len(t, want, len(campaignTools))
	for _, name := range campaignTools {
		tool, ok := tools[name]
		require.True(t, ok, name)
		a := tool.Annotations
		require.False(t, a.ReadOnlyHint, name)
		require.NotNil(t, a.DestructiveHint, name)
		require.Equal(t, want[name].destructive, *a.DestructiveHint, name)
		require.Equal(t, want[name].idempotent, a.IdempotentHint, name)
		require.NotNil(t, a.OpenWorldHint, name)
		require.False(t, *a.OpenWorldHint, name)
		require.NotNil(t, tool.OutputSchema, name)
	}
	for name, terms := range map[string][]string{
		"create_campaign":            {"poison DR is global", "per body location", "Immune", "S.P.E.C.I.A.L.", "no more than max HP", "1 or more"},
		"update_campaign_character":  {"creates a new character_id", "replaced_character_id", "poison DR is global", "no more than max HP"},
		"mark_character_unavailable": {"leave every encounter", "confirm"},
		"remove_campaign_player":     {"leaves the campaign's encounters", "confirm"},
	} {
		for _, term := range terms {
			require.Contains(t, tools[name].Description, term, name)
		}
	}
}
