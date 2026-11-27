package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/domain"
)

type turnResult struct {
	Encounter        Encounter `json:"encounter"`
	Round            int       `json:"round"`
	CurrentCombatant Combatant `json:"current_combatant"`
}

func (r turnResult) Summary() string {
	return fmt.Sprintf("Round %d: %s's turn.", r.Round, r.CurrentCombatant.Name)
}

type damageArgs struct {
	CombatantID string              `json:"combatant_id"`
	DamageType  domain.DamageType   `json:"damage_type"`
	Location    domain.BodyLocation `json:"location"`
	Amount      int                 `json:"amount" jsonschema:"raw damage before DR, 0 or more"`
}

type damageResult struct {
	Combatant Combatant `json:"combatant"`
	Raw       int       `json:"raw"`
	Applied   int       `json:"applied" jsonschema:"damage after DR or immunity"`
	Defeated  bool      `json:"defeated"`
}

func (r damageResult) Summary() string {
	s := fmt.Sprintf("%s took %d of %d damage, HP %d/%d.", r.Combatant.Name, r.Applied, r.Raw, r.Combatant.HP, r.Combatant.MaxHP)
	if r.Defeated {
		s += " Defeated."
	}
	return s
}

type healArgs struct {
	CombatantID string `json:"combatant_id"`
	Amount      int    `json:"amount" jsonschema:"HP to restore, 0 or more"`
}

type healResult struct {
	Combatant Combatant `json:"combatant"`
	Healed    int       `json:"healed" jsonschema:"HP actually restored; healing stops at max HP"`
}

func (r healResult) Summary() string {
	return fmt.Sprintf("%s healed %d, HP %d/%d.", r.Combatant.Name, r.Healed, r.Combatant.HP, r.Combatant.MaxHP)
}

type deltaArgs struct {
	Delta int `json:"delta" jsonschema:"non-zero change: positive adds, negative spends"`
}

type apResult struct {
	PartyAP int  `json:"party_ap"`
	Capped  bool `json:"capped" jsonschema:"true when the 6-point cap cut the addition"`
}

func (r apResult) Summary() string {
	if r.Capped {
		return fmt.Sprintf("Party AP %d (capped at %d).", r.PartyAP, domain.MaxPartyAP)
	}
	return fmt.Sprintf("Party AP %d.", r.PartyAP)
}

type threatResult struct {
	GMThreat int `json:"gm_threat"`
}

func (r threatResult) Summary() string { return fmt.Sprintf("GM Threat %d.", r.GMThreat) }

func registerCombatTools(s *mcp.Server, d *deps) {
	addTool(s, d, toolDef{Name: "advance_turn", Title: "Advance the turn", Effect: effectModify,
		Description: "Advance the active encounter to the next combatant in initiative order; after the last one a new round starts. Returns whose turn it is now."},
		d.advanceTurn)
	addTool(s, d, toolDef{Name: "apply_damage", Title: "Apply damage", Effect: effectModify,
		Description: "Apply raw damage of a type to a body location of a combatant in the active encounter. Applied damage is raw minus DR at that location, or 0 when the combatant is immune to the type. " +
			"Poison uses the global poison DR. Torso-only creatures always take the hit on the torso. HP stops at 0 and the combatant becomes defeated."},
		d.applyDamage)
	addTool(s, d, toolDef{Name: "heal", Title: "Heal", Effect: effectModify,
		Description: "Restore HP to a combatant in the active encounter. Healing stops at max HP. Use it to correct a mistaken damage call."},
		d.heal)
	addTool(s, d, toolDef{Name: "change_party_ap", Title: "Change Party AP", Effect: effectModify,
		Description: "Add (positive delta) or spend (negative delta) shared Party AP of the active campaign. Party AP is capped at 6; spending more than is available is rejected."},
		d.changePartyAP)
	addTool(s, d, toolDef{Name: "change_gm_threat", Title: "Change GM Threat", Effect: effectModify,
		Description: "Add (positive delta) or spend (negative delta) GM Threat of the active campaign. Threat has no cap; spending more than is available is rejected."},
		d.changeGMThreat)
}

func (d *deps) advanceTurn(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (res *mcp.CallToolResult, out turnResult, err error) {
	act := activity{Operation: "advance_turn", Kind: domain.EntityEncounter}
	defer func() { d.record(ctx, act, err) }()
	enc, err := d.svc.AdvanceTurn(ctx)
	if err != nil {
		return nil, turnResult{}, err
	}
	dto := encounterDTO(enc)
	out = turnResult{Encounter: dto, Round: enc.Round}
	if c := enc.ActiveCombatant(); c != nil {
		out.CurrentCombatant = combatantDTO(*c, true)
	}
	act.ID, act.Label, act.Detail = enc.ID, enc.Name, out.Summary()
	return nil, out, nil
}

func (d *deps) applyDamage(ctx context.Context, _ *mcp.CallToolRequest, in damageArgs) (res *mcp.CallToolResult, out damageResult, err error) {
	act := activity{Operation: "apply_damage", Kind: domain.EntityCombatant, ID: in.CombatantID}
	defer func() { d.record(ctx, act, err) }()
	if strings.TrimSpace(in.CombatantID) == "" {
		return nil, damageResult{}, invalidInput("combatant_id", "combatant_id is required")
	}
	if in.Amount < 0 {
		return nil, damageResult{}, invalidInput("amount", "amount must be 0 or more")
	}
	enc, applied, err := d.svc.ApplyDamage(ctx, in.CombatantID, in.DamageType, in.Location, in.Amount)
	if err != nil {
		return nil, damageResult{}, err
	}
	c, err := combatantIn(enc, in.CombatantID)
	if err != nil {
		return nil, damageResult{}, err
	}
	out = damageResult{Combatant: c, Raw: in.Amount, Applied: applied, Defeated: c.Defeated}
	act.Label = c.Name
	act.Detail = fmt.Sprintf("%d %s → %s, applied %d", in.Amount, in.DamageType, in.Location, applied)
	return nil, out, nil
}

func (d *deps) heal(ctx context.Context, _ *mcp.CallToolRequest, in healArgs) (res *mcp.CallToolResult, out healResult, err error) {
	act := activity{Operation: "heal", Kind: domain.EntityCombatant, ID: in.CombatantID}
	defer func() { d.record(ctx, act, err) }()
	if strings.TrimSpace(in.CombatantID) == "" {
		return nil, healResult{}, invalidInput("combatant_id", "combatant_id is required")
	}
	if in.Amount < 0 {
		return nil, healResult{}, invalidInput("amount", "amount must be 0 or more")
	}
	enc, healed, err := d.svc.Heal(ctx, in.CombatantID, in.Amount)
	if err != nil {
		return nil, healResult{}, err
	}
	c, err := combatantIn(enc, in.CombatantID)
	if err != nil {
		return nil, healResult{}, err
	}
	act.Label, act.Detail = c.Name, fmt.Sprintf("healed %d", healed)
	return nil, healResult{Combatant: c, Healed: healed}, nil
}

func (d *deps) changePartyAP(ctx context.Context, _ *mcp.CallToolRequest, in deltaArgs) (res *mcp.CallToolResult, out apResult, err error) {
	act := activity{Operation: "change_party_ap", Kind: domain.EntityCampaign, Detail: fmt.Sprintf("delta %+d", in.Delta)}
	defer func() { d.record(ctx, act, err) }()
	if in.Delta == 0 {
		return nil, apResult{}, invalidInput("delta", "delta must not be 0")
	}
	var (
		enc    *domain.Encounter
		capped bool
	)
	if in.Delta > 0 {
		enc, capped, err = d.svc.AddPartyAP(ctx, in.Delta)
	} else {
		enc, err = d.svc.SpendPartyAP(ctx, -in.Delta)
	}
	if err != nil {
		return nil, apResult{}, err
	}
	act.ID = enc.CampaignID
	out = apResult{PartyAP: enc.Resources.PartyAP, Capped: capped}
	act.Detail = fmt.Sprintf("delta %+d, total %d", in.Delta, out.PartyAP)
	return nil, out, nil
}

func (d *deps) changeGMThreat(ctx context.Context, _ *mcp.CallToolRequest, in deltaArgs) (res *mcp.CallToolResult, out threatResult, err error) {
	act := activity{Operation: "change_gm_threat", Kind: domain.EntityCampaign, Detail: fmt.Sprintf("delta %+d", in.Delta)}
	defer func() { d.record(ctx, act, err) }()
	if in.Delta == 0 {
		return nil, threatResult{}, invalidInput("delta", "delta must not be 0")
	}
	var enc *domain.Encounter
	if in.Delta > 0 {
		enc, err = d.svc.AddThreat(ctx, in.Delta)
	} else {
		enc, err = d.svc.SpendThreat(ctx, -in.Delta)
	}
	if err != nil {
		return nil, threatResult{}, err
	}
	act.ID = enc.CampaignID
	act.Detail = fmt.Sprintf("delta %+d, total %d", in.Delta, enc.Resources.GMThreat)
	return nil, threatResult{GMThreat: enc.Resources.GMThreat}, nil
}

func combatantIn(enc *domain.Encounter, id string) (Combatant, error) {
	for i, c := range enc.Combatants {
		if c.ID == id {
			return combatantDTO(c, i == enc.TurnIndex), nil
		}
	}
	return Combatant{}, notFound(fmt.Sprintf("combatant %q not found", id))
}
