package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

const (
	termsCharacter = "Current HP is 0 or more and no more than max HP. S.P.E.C.I.A.L. attributes (strength, perception, endurance, charisma, intelligence, agility, luck) are each 1 or more; omitted means 1 for every attribute. " + termsDR
	termsPlayers   = "Players are matched by name, case-insensitively, so player names are unique in a campaign."
)

// playerInput is one player of a new campaign.
type playerInput struct {
	PlayerName string         `json:"player_name" jsonschema:"the player's name, unique in the campaign (case-insensitive)"`
	Character  CharacterInput `json:"character"`
	Notes      string         `json:"notes,omitempty" jsonschema:"free-form notes; line breaks are kept"`
	Special    *Special       `json:"special,omitempty" jsonschema:"S.P.E.C.I.A.L. attributes, each 1 or more; omitted means 1 for every attribute"`
	Available  *bool          `json:"available,omitempty" jsonschema:"false makes the character unavailable for encounters; default true"`
}

func newPlayer(name string, c CharacterInput, notes string, special *Special) domain.NewCampaignPlayer {
	p := domain.NewCampaignPlayer{PlayerName: name, Character: c.combatant(), Notes: notes}
	if special != nil {
		p.Special = specialValues(*special)
	}
	return p
}

type createCampaignInput struct {
	Name      string        `json:"name"`
	StartDate string        `json:"start_date" jsonschema:"in-game start date, YYYY-MM-DD"`
	Players   []playerInput `json:"players" jsonschema:"at least one player with a character"`
}

// campaignDetail is a campaign with its roster.
type campaignDetail struct {
	Campaign
	Characters []Character `json:"characters"`
}

func (c campaignDetail) Summary() string {
	return fmt.Sprintf("Campaign «%s» (%s) with %d character(s).", c.Name, c.CampaignID, len(c.Characters))
}

// Summary describes the campaign in one line for people (FR-010).
func (c Campaign) Summary() string {
	active := ""
	if c.IsActive {
		active = ", active"
	}
	return fmt.Sprintf("Campaign «%s» (%s), starts %s%s.", c.Name, c.CampaignID, c.StartDate, active)
}

type updateCampaignDetailsInput struct {
	CampaignID string  `json:"campaign_id"`
	Name       *string `json:"name,omitempty" jsonschema:"new campaign name"`
	StartDate  *string `json:"start_date,omitempty" jsonschema:"new start date, YYYY-MM-DD"`
}

type addPlayerInput struct {
	CampaignID string         `json:"campaign_id"`
	PlayerName string         `json:"player_name" jsonschema:"the player's name, unique in the campaign (case-insensitive)"`
	Character  CharacterInput `json:"character"`
	Notes      string         `json:"notes,omitempty" jsonschema:"free-form notes; line breaks are kept"`
	Special    *Special       `json:"special,omitempty" jsonschema:"S.P.E.C.I.A.L. attributes, each 1 or more; omitted means 1 for every attribute"`
}

type playerAdded struct {
	Campaign  Campaign  `json:"campaign"`
	Character Character `json:"character"`
}

func (p playerAdded) Summary() string {
	return fmt.Sprintf("Added %s (%s) to «%s» as %s.", p.Character.PlayerName, p.Character.Name, p.Campaign.Name, p.Character.CharacterID)
}

// characterPatchInput lists the character fields to change. Omitted fields
// keep their stored value.
type characterPatchInput struct {
	Name       *string     `json:"name,omitempty" jsonschema:"new character name; renaming creates a new character_id"`
	Level      *int        `json:"level,omitempty" jsonschema:"1 or more"`
	Initiative *int        `json:"initiative,omitempty" jsonschema:"0 or more"`
	HP         *int        `json:"hp,omitempty" jsonschema:"current HP, 0 or more and no more than max HP"`
	MaxHP      *int        `json:"max_hp,omitempty" jsonschema:"maximum HP, 1 or more"`
	Defense    *int        `json:"defense,omitempty" jsonschema:"0 or more"`
	Resistance *Resistance `json:"resistance,omitempty" jsonschema:"replaces every DR value: per body location for physical, energy and radiation, and global poison DR"`
	Immunity   *Immunity   `json:"immunity,omitempty" jsonschema:"replaces the four immunity flags"`
	Special    *Special    `json:"special,omitempty" jsonschema:"replaces all seven S.P.E.C.I.A.L. attributes, each 1 or more"`
	Notes      *string     `json:"notes,omitempty" jsonschema:"replaces the notes; line breaks are kept"`
}

// apply copies the set fields onto p.
func (in characterPatchInput) apply(p *domain.NewCampaignPlayer) {
	c := &p.Character
	if in.Name != nil {
		c.Name = *in.Name
	}
	setInt(&c.Level, in.Level)
	setInt(&c.Initiative, in.Initiative)
	setInt(&c.HP, in.HP)
	setInt(&c.MaxHP, in.MaxHP)
	setInt(&c.Defense, in.Defense)
	if in.Resistance != nil || in.Immunity != nil {
		r, im := resistanceDTO(*c), immunityDTO(*c)
		if in.Resistance != nil {
			r = *in.Resistance
		}
		if in.Immunity != nil {
			im = *in.Immunity
		}
		applyResistance(c, r, im)
	}
	if in.Special != nil {
		p.Special = specialValues(*in.Special)
	}
	if in.Notes != nil {
		p.Notes = *in.Notes
	}
}

func setInt(dst, src *int) {
	if src != nil {
		*dst = *src
	}
}

type updateCharacterInput struct {
	CampaignID  string              `json:"campaign_id"`
	CharacterID string              `json:"character_id"`
	Patch       characterPatchInput `json:"patch" jsonschema:"the fields to change; omitted fields keep their value"`
}

type characterUpdated struct {
	Character           Character `json:"character"`
	ReplacedCharacterID string    `json:"replaced_character_id,omitempty" jsonschema:"the previous character_id when a rename created a new character"`
}

func (c characterUpdated) Summary() string {
	if c.ReplacedCharacterID != "" {
		return fmt.Sprintf("Updated %s; the rename created character_id %s, replacing %s.", c.Character.Name, c.Character.CharacterID, c.ReplacedCharacterID)
	}
	return fmt.Sprintf("Updated %s (%s).", c.Character.Name, c.Character.CharacterID)
}

type characterRef struct {
	CampaignID  string `json:"campaign_id"`
	CharacterID string `json:"character_id"`
}

type characterResult struct {
	Character Character `json:"character"`
}

func (c characterResult) Summary() string {
	state := "available"
	if !c.Character.Available {
		state = "unavailable"
	}
	return fmt.Sprintf("%s (%s) is %s.", c.Character.Name, c.Character.CharacterID, state)
}

type campaignRef struct {
	CampaignID string `json:"campaign_id"`
}

type playerRemoved struct {
	Campaign Campaign  `json:"campaign"`
	Removed  Character `json:"removed" jsonschema:"the removed player's character as it was stored"`
}

func (p playerRemoved) Summary() string {
	return fmt.Sprintf("Removed %s (%s) from «%s».", p.Removed.PlayerName, p.Removed.Name, p.Campaign.Name)
}

func registerCampaignTools(s *mcp.Server, d *deps) {
	addTool(s, d, toolDef{Name: "create_campaign", Title: "Create a campaign", Effect: effectModify,
		Description: "Create a campaign with its players and their characters. start_date is YYYY-MM-DD. Each character needs a name, level (1 or more), initiative, current HP, max HP and Defense. " + termsCharacter + " " + termsPlayers},
		d.createCampaign)
	addTool(s, d, toolDef{Name: "update_campaign_details", Title: "Update campaign details", Effect: effectModify, Idempotent: true,
		Description: "Rename a campaign or change its start date (YYYY-MM-DD). The roster is unchanged."},
		d.updateCampaignDetails)
	addTool(s, d, toolDef{Name: "add_campaign_player", Title: "Add a campaign player", Effect: effectModify,
		Description: "Add a player and their character to a campaign. A duplicate player name is rejected. " + termsPlayers + " " + termsCharacter},
		d.addCampaignPlayer)
	addTool(s, d, toolDef{Name: "update_campaign_character", Title: "Update a campaign character", Effect: effectModify, Idempotent: true,
		Description: "Change some fields of a campaign character; omitted fields keep their value. Renaming a character creates a new character_id: the result returns it, with the old id in replaced_character_id. Changes to level, HP, Defense and DR also reach the character's combatant in the active encounter. " + termsCharacter},
		d.updateCampaignCharacter)
	addTool(s, d, toolDef{Name: "mark_character_available", Title: "Mark a character available", Effect: effectModify, Idempotent: true,
		Description: "Make an unavailable character available again, so they can join new encounters. They do not rejoin past encounters."},
		d.markCharacterAvailable)
	addTool(s, d, toolDef{Name: "mark_character_unavailable", Title: "Mark a character unavailable", Effect: effectDestroy,
		Description: "Mark a character unavailable. Unavailable characters leave every encounter of the campaign and cannot join new ones until marked available. The GM must confirm."},
		d.markCharacterUnavailable)
	addTool(s, d, toolDef{Name: "activate_campaign", Title: "Activate a campaign", Effect: effectModify, Idempotent: true,
		Description: "Make a campaign the active one. New encounters are created in the active campaign, which holds the shared Party AP (0 to 6) and GM Threat."},
		d.activateCampaign)
	addTool(s, d, toolDef{Name: "remove_campaign_player", Title: "Remove a campaign player", Effect: effectDestroy,
		Description: "Remove a player and their character from a campaign. The character also leaves the campaign's encounters. A campaign keeps at least one player. The GM must confirm."},
		d.removeCampaignPlayer)
}

func parseStartDate(value string) (time.Time, error) {
	t, err := domain.ParseCampaignStartDate(value)
	if err != nil {
		return time.Time{}, invalidInput("start_date", "start_date must be a date in YYYY-MM-DD form")
	}
	return t, nil
}

func (d *deps) createCampaign(ctx context.Context, _ *mcp.CallToolRequest, in createCampaignInput) (res *mcp.CallToolResult, out campaignDetail, err error) {
	a := activity{Operation: "create_campaign", Kind: domain.EntityCampaign, Label: in.Name}
	defer func() { d.finish(ctx, a, res, err) }()

	start, err := parseStartDate(in.StartDate)
	if err != nil {
		return nil, out, err
	}
	if len(in.Players) == 0 {
		return nil, out, invalidInput("players", "players must list at least one player")
	}
	players := make([]domain.NewCampaignPlayer, 0, len(in.Players))
	for _, p := range in.Players {
		player := newPlayer(p.PlayerName, p.Character, p.Notes, p.Special)
		player.Inactive = p.Available != nil && !*p.Available
		players = append(players, player)
	}
	c, err := d.svc.ExecuteCreateCampaign(ctx, app.CreateCampaignCommand{Name: in.Name, StartDate: start, Players: players})
	if err != nil {
		return nil, out, err
	}
	a.ID = c.ID
	stored, err := d.storedCampaign(ctx, c.ID)
	if err != nil {
		return nil, out, err
	}
	campaign, err := d.campaignOut(ctx, stored)
	if err != nil {
		return nil, out, err
	}
	roster, err := d.svc.ListCampaignPlayers(ctx, c.ID)
	if err != nil {
		return nil, out, err
	}
	out = campaignDetail{Campaign: campaign, Characters: make([]Character, 0, len(roster))}
	for _, p := range roster {
		out.Characters = append(out.Characters, characterDTO(p))
	}
	return nil, out, nil
}

func (d *deps) updateCampaignDetails(ctx context.Context, _ *mcp.CallToolRequest, in updateCampaignDetailsInput) (res *mcp.CallToolResult, out Campaign, err error) {
	a := activity{Operation: "update_campaign_details", Kind: domain.EntityCampaign, ID: in.CampaignID}
	defer func() { d.finish(ctx, a, res, err) }()

	if in.Name == nil && in.StartDate == nil {
		return nil, out, invalidInput("", "set name or start_date to change")
	}
	patch := &app.CampaignDetailsPatch{Name: in.Name}
	if in.StartDate != nil {
		var start time.Time
		if start, err = parseStartDate(*in.StartDate); err != nil {
			return nil, out, err
		}
		patch.StartDate = &start
	}
	r, err := d.svc.ExecuteCampaignRosterPatch(ctx, app.CampaignRosterPatch{CampaignID: in.CampaignID, Details: patch})
	if err != nil {
		return nil, out, err
	}
	a.Label = r.Campaign.Name
	out, err = d.campaignOut(ctx, r.Campaign)
	return nil, out, err
}

func (d *deps) addCampaignPlayer(ctx context.Context, _ *mcp.CallToolRequest, in addPlayerInput) (res *mcp.CallToolResult, out playerAdded, err error) {
	a := activity{Operation: "add_campaign_player", Kind: domain.EntityPlayerCharacter, Label: characterLabel(in.Character.Name, in.PlayerName)}
	defer func() { d.finish(ctx, a, res, err) }()

	player := newPlayer(in.PlayerName, in.Character, in.Notes, in.Special)
	r, err := d.svc.ExecuteCampaignRosterPatch(ctx, app.CampaignRosterPatch{CampaignID: in.CampaignID, AddPlayer: &player})
	if err != nil {
		return nil, out, err
	}
	a.ID = r.Player.Character.ID
	campaign, err := d.campaignOut(ctx, r.Campaign)
	if err != nil {
		return nil, out, err
	}
	return nil, playerAdded{Campaign: campaign, Character: characterDTO(*r.Player)}, nil
}

func (d *deps) updateCampaignCharacter(ctx context.Context, _ *mcp.CallToolRequest, in updateCharacterInput) (res *mcp.CallToolResult, out characterUpdated, err error) {
	a := activity{Operation: "update_campaign_character", Kind: domain.EntityPlayerCharacter, ID: in.CharacterID}
	defer func() { d.finish(ctx, a, res, err) }()

	if in.Patch == (characterPatchInput{}) {
		return nil, out, invalidInput("patch", "patch must change at least one field")
	}
	r, err := d.svc.ExecuteCampaignRosterPatch(ctx, app.CampaignRosterPatch{
		CampaignID:      in.CampaignID,
		UpdateCharacter: &app.CharacterPatch{CharacterID: in.CharacterID, Edit: in.Patch.apply},
	})
	if err != nil {
		return nil, out, err
	}
	a.ID, a.Label = r.Player.Character.ID, characterLabel(r.Player.Character.Name, r.Player.PlayerName)
	if r.ReplacedCharacterID != "" {
		a.Detail = "renamed; replaces character_id " + r.ReplacedCharacterID
	}
	return nil, characterUpdated{Character: characterDTO(*r.Player), ReplacedCharacterID: r.ReplacedCharacterID}, nil
}

func (d *deps) markCharacterAvailable(ctx context.Context, _ *mcp.CallToolRequest, in characterRef) (res *mcp.CallToolResult, out characterResult, err error) {
	a := activity{Operation: "mark_character_available", Kind: domain.EntityPlayerCharacter, ID: in.CharacterID}
	defer func() { d.finish(ctx, a, res, err) }()

	out, err = d.setAvailability(ctx, in, true, &a)
	return nil, out, err
}

func (d *deps) markCharacterUnavailable(ctx context.Context, req *mcp.CallToolRequest, in characterRef) (res *mcp.CallToolResult, out characterResult, err error) {
	a := activity{Operation: "mark_character_unavailable", Kind: domain.EntityPlayerCharacter, ID: in.CharacterID}
	defer func() { d.finish(ctx, a, res, err) }()

	campaign, player, _, err := d.storedCharacter(ctx, in)
	if err != nil {
		return nil, out, err
	}
	a.Label = characterLabel(player.Character.Name, player.PlayerName)
	res, ok, err := d.confirm.require(ctx, req, in, app.ConfirmationRequest{
		Operation:   a.Operation,
		EntityKind:  a.Kind,
		EntityID:    in.CharacterID,
		EntityLabel: a.Label,
		Summary:     fmt.Sprintf("Mark %s unavailable? They are removed from all encounters of «%s».", player.Character.Name, campaign.Name),
	})
	if !ok {
		return res, out, err
	}
	out, err = d.setAvailability(ctx, in, false, &a)
	return nil, out, err
}

func (d *deps) setAvailability(ctx context.Context, in characterRef, available bool, a *activity) (characterResult, error) {
	r, err := d.svc.ExecuteCampaignRosterPatch(ctx, app.CampaignRosterPatch{
		CampaignID:      in.CampaignID,
		SetAvailability: &app.CharacterAvailabilityPatch{CharacterID: in.CharacterID, Available: available},
	})
	if err != nil {
		return characterResult{}, err
	}
	a.Label = characterLabel(r.Player.Character.Name, r.Player.PlayerName)
	return characterResult{Character: characterDTO(*r.Player)}, nil
}

func (d *deps) activateCampaign(ctx context.Context, _ *mcp.CallToolRequest, in campaignRef) (res *mcp.CallToolResult, out Campaign, err error) {
	a := activity{Operation: "activate_campaign", Kind: domain.EntityCampaign, ID: in.CampaignID}
	defer func() { d.finish(ctx, a, res, err) }()

	if strings.TrimSpace(in.CampaignID) == "" {
		return nil, out, invalidInput("campaign_id", "campaign_id is required")
	}
	c, err := d.svc.ActivateCampaign(ctx, in.CampaignID)
	if err != nil {
		return nil, out, err
	}
	a.Label = c.Name
	return nil, campaignDTO(*c, c.ID), nil
}

func (d *deps) removeCampaignPlayer(ctx context.Context, req *mcp.CallToolRequest, in characterRef) (res *mcp.CallToolResult, out playerRemoved, err error) {
	a := activity{Operation: "remove_campaign_player", Kind: domain.EntityPlayerCharacter, ID: in.CharacterID}
	defer func() { d.finish(ctx, a, res, err) }()

	campaign, player, players, err := d.storedCharacter(ctx, in)
	if err != nil {
		return nil, out, err
	}
	a.Label = characterLabel(player.Character.Name, player.PlayerName)
	if players == 1 {
		// Refuse before asking the GM about a removal that cannot run.
		return nil, out, invalidInput("character_id", "cannot remove the last player: a campaign needs at least one player")
	}
	res, ok, err := d.confirm.require(ctx, req, in, app.ConfirmationRequest{
		Operation:   a.Operation,
		EntityKind:  a.Kind,
		EntityID:    in.CharacterID,
		EntityLabel: a.Label,
		Summary:     fmt.Sprintf("Remove %s (%s) from campaign «%s»? They are also removed from its encounters.", player.PlayerName, player.Character.Name, campaign.Name),
	})
	if !ok {
		return res, out, err
	}
	r, err := d.svc.ExecuteCampaignRosterPatch(ctx, app.CampaignRosterPatch{CampaignID: in.CampaignID, RemovePlayer: &app.RemovePlayerPatch{CharacterID: in.CharacterID}})
	if err != nil {
		return nil, out, err
	}
	c, err := d.campaignOut(ctx, r.Campaign)
	if err != nil {
		return nil, out, err
	}
	return nil, playerRemoved{Campaign: c, Removed: characterDTO(player)}, nil
}

// campaignOut maps c, marking whether it is the active campaign.
func (d *deps) campaignOut(ctx context.Context, c domain.Campaign) (Campaign, error) {
	activeID, err := d.activeCampaignID(ctx)
	if err != nil {
		return Campaign{}, err
	}
	return campaignDTO(c, activeID), nil
}

// storedCharacter reads the campaign and character ref names, and the size
// of the roster. Confirmation text is built from this stored state, never
// from tool arguments.
func (d *deps) storedCharacter(ctx context.Context, ref characterRef) (domain.Campaign, domain.NewCampaignPlayer, int, error) {
	c, err := d.storedCampaign(ctx, ref.CampaignID)
	if err != nil {
		return domain.Campaign{}, domain.NewCampaignPlayer{}, 0, err
	}
	players, err := d.svc.ListCampaignPlayers(ctx, c.ID)
	if err != nil {
		return domain.Campaign{}, domain.NewCampaignPlayer{}, 0, err
	}
	for _, p := range players {
		if p.Character.ID == ref.CharacterID {
			return c, p, len(players), nil
		}
	}
	return domain.Campaign{}, domain.NewCampaignPlayer{}, 0, notFound(fmt.Sprintf("character %q not found in campaign %q", ref.CharacterID, ref.CampaignID))
}

// storedCampaign returns the stored campaign with the given ID.
func (d *deps) storedCampaign(ctx context.Context, campaignID string) (domain.Campaign, error) {
	if strings.TrimSpace(campaignID) == "" {
		return domain.Campaign{}, invalidInput("campaign_id", "campaign_id is required")
	}
	campaigns, err := d.svc.ListCampaigns(ctx)
	if err != nil {
		return domain.Campaign{}, err
	}
	for _, c := range campaigns {
		if c.ID == campaignID {
			return c, nil
		}
	}
	return domain.Campaign{}, domain.ErrCampaignNotFound
}

func characterLabel(character, player string) string {
	return fmt.Sprintf("%s (%s)", character, player)
}
