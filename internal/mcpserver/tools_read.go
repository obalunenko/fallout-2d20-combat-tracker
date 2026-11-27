package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/domain"
)

const (
	defaultLogLimit      = 100
	maxLogLimit          = 500
	defaultActivityLimit = 20
	maxActivityLimit     = 200
)

const termsDR = "DR (damage resistance) is per body location for physical, energy and radiation damage; poison DR is global. Immune damage types deal no damage."

type campaignList struct {
	Campaigns []Campaign `json:"campaigns"`
}

func (c campaignList) Summary() string { return fmt.Sprintf("%d campaign(s).", len(c.Campaigns)) }

type activeCampaign struct {
	Initialized bool      `json:"initialized" jsonschema:"false when no campaign exists yet"`
	Campaign    *Campaign `json:"campaign"`
}

func (a activeCampaign) Summary() string {
	if a.Campaign == nil {
		return "No campaign yet."
	}
	return fmt.Sprintf("Active campaign: %s (Party AP %d, GM Threat %d).", a.Campaign.Name, a.Campaign.PartyAP, a.Campaign.GMThreat)
}

type campaignArg struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign to read; defaults to the active campaign"`
}

type characterList struct {
	CampaignID string      `json:"campaign_id"`
	Characters []Character `json:"characters"`
}

func (c characterList) Summary() string {
	return fmt.Sprintf("%d player character(s).", len(c.Characters))
}

type partyList struct {
	Members []Combatant `json:"members"`
}

func (p partyList) Summary() string {
	return fmt.Sprintf("%d available party member(s).", len(p.Members))
}

type encounterList struct {
	Encounters []EncounterSummary `json:"encounters"`
}

func (e encounterList) Summary() string { return fmt.Sprintf("%d encounter(s).", len(e.Encounters)) }

type encounterArg struct {
	EncounterID string `json:"encounter_id"`
}

// Summary describes the encounter in one line for people (FR-010).
func (e Encounter) Summary() string {
	return fmt.Sprintf("%s: round %d, %d combatant(s), difficulty %s.", e.Name, e.Round, len(e.Combatants), e.Difficulty.Label)
}

type activeEncounter struct {
	Encounter *Encounter `json:"encounter"`
}

func (a activeEncounter) Summary() string {
	if a.Encounter == nil {
		return "No active encounter."
	}
	return "Active encounter " + a.Encounter.Summary()
}

type logArg struct {
	EncounterID string `json:"encounter_id"`
	Limit       int    `json:"limit,omitempty" jsonschema:"maximum entries, 1 to 500, default 100"`
}

type logList struct {
	Entries []LogEntry `json:"entries" jsonschema:"newest first"`
}

func (l logList) Summary() string { return fmt.Sprintf("%d log entr(ies).", len(l.Entries)) }

type templateList struct {
	Templates []MonsterTemplate `json:"templates"`
}

func (t templateList) Summary() string {
	return fmt.Sprintf("%d monster template(s).", len(t.Templates))
}

type activityArg struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum entries, 1 to 200, default 20"`
}

type activityList struct {
	Entries []Activity `json:"entries" jsonschema:"newest first"`
}

func (a activityList) Summary() string {
	return fmt.Sprintf("%d assistant activity entr(ies).", len(a.Entries))
}

func registerReadTools(s *mcp.Server, d *deps) {
	addTool(s, d, toolDef{Name: "list_assistant_activity", Title: "List assistant activity", Effect: effectRead,
		Description: "List what the assistant did recently, newest first: each changing operation with its target and outcome (done, rejected, declined, timed_out), plus refused connection attempts. " +
			"There is no undo; use this to find a mistaken action and correct it with the opposite operation (for example heal the damage applied). Read-only: the log cannot be changed here."},
		d.listAssistantActivity)
	addTool(s, d, toolDef{Name: "list_campaigns", Title: "List campaigns", Effect: effectRead,
		Description: "List every campaign with its shared Party AP (0 to 6) and GM Threat, and which one is active."},
		d.listCampaigns)
	addTool(s, d, toolDef{Name: "get_active_campaign", Title: "Get the active campaign", Effect: effectRead,
		Description: "Get the active campaign. Returns initialized=false, not an error, when no campaign exists yet."},
		d.getActiveCampaign)
	addTool(s, d, toolDef{Name: "list_campaign_characters", Title: "List campaign characters", Effect: effectRead,
		Description: "List a campaign's player characters with level, HP, Defense, DR, immunities, S.P.E.C.I.A.L., notes and availability. " + termsDR},
		d.listCampaignCharacters)
	addTool(s, d, toolDef{Name: "list_party_members", Title: "List party members", Effect: effectRead,
		Description: "List the active campaign's available player characters that can join an encounter. Use their character_id in create_encounter."},
		d.listPartyMembers)
	addTool(s, d, toolDef{Name: "list_encounters", Title: "List encounters", Effect: effectRead,
		Description: "List encounters with round, combatant count and difficulty. Difficulty bands by encounter level minus average PC level: below -2 Trivial, -2..-1 Simple, 0..1 Average, 2..5 Hard, above 5 Deadly."},
		d.listEncounters)
	addTool(s, d, toolDef{Name: "get_encounter", Title: "Get an encounter", Effect: effectRead,
		Description: "Get one encounter: round, whose turn it is, Party AP, GM Threat, and every combatant in initiative order. " + termsDR},
		d.getEncounter)
	addTool(s, d, toolDef{Name: "get_active_encounter", Title: "Get the active encounter", Effect: effectRead,
		Description: "Get the active encounter, the one combat actions apply to, or null when there is none."},
		d.getActiveEncounter)
	addTool(s, d, toolDef{Name: "get_encounter_log", Title: "Get an encounter log", Effect: effectRead,
		Description: "Get an encounter's operation log, newest first. source tells whether the desktop app or the assistant performed each operation."},
		d.getEncounterLog)
	addTool(s, d, toolDef{Name: "list_monster_templates", Title: "List monster templates", Effect: effectRead,
		Description: "List saved monster templates. Use their template_id in create_encounter, update_encounter and preview_encounter_difficulty. Torso-only creatures are always hit on the torso."},
		d.listMonsterTemplates)
}

func (d *deps) activeCampaignID(ctx context.Context) (string, error) {
	c, err := d.svc.GetActiveCampaign(ctx)
	if errors.Is(err, domain.ErrCampaignNotInitialized) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return c.ID, nil
}

func (d *deps) listCampaigns(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, campaignList, error) {
	campaigns, err := d.svc.ListCampaigns(ctx)
	if err != nil {
		return nil, campaignList{}, err
	}
	activeID, err := d.activeCampaignID(ctx)
	if err != nil {
		return nil, campaignList{}, err
	}
	out := campaignList{Campaigns: make([]Campaign, 0, len(campaigns))}
	for _, c := range campaigns {
		out.Campaigns = append(out.Campaigns, campaignDTO(c, activeID))
	}
	return nil, out, nil
}

func (d *deps) getActiveCampaign(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, activeCampaign, error) {
	c, err := d.svc.GetActiveCampaign(ctx)
	if errors.Is(err, domain.ErrCampaignNotInitialized) {
		return nil, activeCampaign{}, nil
	}
	if err != nil {
		return nil, activeCampaign{}, err
	}
	dto := campaignDTO(*c, c.ID)
	return nil, activeCampaign{Initialized: true, Campaign: &dto}, nil
}

func (d *deps) listCampaignCharacters(ctx context.Context, _ *mcp.CallToolRequest, in campaignArg) (*mcp.CallToolResult, characterList, error) {
	id := strings.TrimSpace(in.CampaignID)
	if id == "" {
		c, err := d.svc.GetActiveCampaign(ctx)
		if err != nil {
			return nil, characterList{}, err
		}
		id = c.ID
	}
	players, err := d.svc.ListCampaignPlayers(ctx, id)
	if err != nil {
		return nil, characterList{}, err
	}
	out := characterList{CampaignID: id, Characters: make([]Character, 0, len(players))}
	for _, p := range players {
		out.Characters = append(out.Characters, characterDTO(p))
	}
	return nil, out, nil
}

func (d *deps) listPartyMembers(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, partyList, error) {
	members, err := d.svc.ListPartyMembers(ctx)
	if err != nil {
		return nil, partyList{}, err
	}
	out := partyList{Members: make([]Combatant, 0, len(members))}
	for _, m := range members {
		out.Members = append(out.Members, combatantDTO(m, false))
	}
	return nil, out, nil
}

func (d *deps) listEncounters(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, encounterList, error) {
	summaries, err := d.svc.ListEncounters(ctx)
	if err != nil {
		return nil, encounterList{}, err
	}
	out := encounterList{Encounters: make([]EncounterSummary, 0, len(summaries))}
	for _, s := range summaries {
		out.Encounters = append(out.Encounters, encounterSummaryDTO(s))
	}
	return nil, out, nil
}

func (d *deps) getEncounter(ctx context.Context, _ *mcp.CallToolRequest, in encounterArg) (*mcp.CallToolResult, Encounter, error) {
	if strings.TrimSpace(in.EncounterID) == "" {
		return nil, Encounter{}, invalidInput("encounter_id", "encounter_id is required")
	}
	enc, err := d.svc.GetEncounterByID(ctx, in.EncounterID)
	if err != nil {
		return nil, Encounter{}, err
	}
	return nil, encounterDTO(enc), nil
}

func (d *deps) getActiveEncounter(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, activeEncounter, error) {
	enc, err := d.svc.GetEncounter(ctx)
	if errors.Is(err, domain.ErrEncounterNotInitialized) || errors.Is(err, domain.ErrCampaignNotInitialized) {
		return nil, activeEncounter{}, nil
	}
	if err != nil {
		return nil, activeEncounter{}, err
	}
	dto := encounterDTO(enc)
	return nil, activeEncounter{Encounter: &dto}, nil
}

// resolveLimit returns limit, or def when limit is 0, and rejects a value
// outside 1 to maxLimit.
func resolveLimit(limit, def, maxLimit int) (int, error) {
	if limit == 0 {
		limit = def
	}
	if limit < 1 || limit > maxLimit {
		return 0, invalidInput("limit", fmt.Sprintf("limit must be between 1 and %d", maxLimit))
	}
	return limit, nil
}

func (d *deps) getEncounterLog(ctx context.Context, _ *mcp.CallToolRequest, in logArg) (*mcp.CallToolResult, logList, error) {
	limit, err := resolveLimit(in.Limit, defaultLogLimit, maxLogLimit)
	if err != nil {
		return nil, logList{}, err
	}
	if strings.TrimSpace(in.EncounterID) == "" {
		return nil, logList{}, invalidInput("encounter_id", "encounter_id is required")
	}
	// Reject unknown or deleted encounters instead of returning an empty log.
	if _, err := d.svc.GetEncounterByID(ctx, in.EncounterID); err != nil {
		return nil, logList{}, err
	}
	logs, err := d.svc.ListEncounterLogs(ctx, in.EncounterID)
	if err != nil {
		return nil, logList{}, err
	}
	if len(logs) > limit {
		logs = logs[:limit]
	}
	out := logList{Entries: make([]LogEntry, 0, len(logs))}
	for _, l := range logs {
		out.Entries = append(out.Entries, logEntryDTO(l))
	}
	return nil, out, nil
}

func (d *deps) listMonsterTemplates(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, templateList, error) {
	templates, err := d.svc.ListMonsterTemplates(ctx)
	if err != nil {
		return nil, templateList{}, err
	}
	out := templateList{Templates: make([]MonsterTemplate, 0, len(templates))}
	for _, t := range templates {
		out.Templates = append(out.Templates, monsterTemplateDTO(t))
	}
	return nil, out, nil
}

func (d *deps) listAssistantActivity(ctx context.Context, _ *mcp.CallToolRequest, in activityArg) (*mcp.CallToolResult, activityList, error) {
	limit, err := resolveLimit(in.Limit, defaultActivityLimit, maxActivityLimit)
	if err != nil {
		return nil, activityList{}, err
	}
	entries, err := d.svc.ListAssistantActivity(ctx, limit)
	if err != nil {
		return nil, activityList{}, err
	}
	out := activityList{Entries: make([]Activity, 0, len(entries))}
	for _, e := range entries {
		out.Entries = append(out.Entries, activityDTO(e))
	}
	return nil, out, nil
}
