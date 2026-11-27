package mcpserver

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/obalunenko/fallout/internal/app"
	"github.com/obalunenko/fallout/internal/domain"
)

const (
	termsDifficulty = "Difficulty: average PC level = the party members' average level, rounded up; XP baseline = total monster XP ÷ number of party members; " +
		"encounter level = max(1, floor((XP baseline − 10) ÷ 10)); difference = encounter level − average PC level. " +
		"Bands by difference: below -2 Trivial, -2..-1 Simple, 0..1 Average, 2..5 Hard, above 5 Deadly; Unknown without party members."
	termsDraft = "Party members are character_id values from list_party_members (available characters of the active campaign). " +
		"Each monster entry has a template_id from list_monster_templates or inline monster stats (exactly one) and a quantity of 1 to 50; every copy becomes its own combatant. " +
		"Torso-only creatures are always hit on the torso. " + termsDR
	termsEncounterActive = "Saving an encounter makes it the active encounter (tracker behavior)."
)

type previewInput struct {
	PartyCharacterIDs []string       `json:"party_character_ids" jsonschema:"character_id values from list_party_members"`
	Monsters          []DraftMonster `json:"monsters"`
}

// Summary describes the difficulty in one line for people (FR-010).
func (d Difficulty) Summary() string {
	if d.UnavailableReason != "" {
		return fmt.Sprintf("Difficulty %s: %s.", d.Label, d.UnavailableReason)
	}
	return fmt.Sprintf("Difficulty %s: encounter level %d, average PC level %d (difference %+d), %d monster XP for %d party member(s).",
		d.Label, d.EncounterLevel, d.AveragePCLevel, d.Difference, d.TotalMonsterXP, d.PartyCount)
}

type createEncounterInput struct {
	Name              string         `json:"name"`
	PartyCharacterIDs []string       `json:"party_character_ids" jsonschema:"character_id values from list_party_members"`
	Monsters          []DraftMonster `json:"monsters"`
}

type updateEncounterInput struct {
	EncounterID          string         `json:"encounter_id"`
	Name                 *string        `json:"name,omitempty" jsonschema:"new name; omitted keeps the current name"`
	KeepCombatantIDs     []string       `json:"keep_combatant_ids" jsonschema:"combatant_id of every stored combatant to keep, with its current HP and defeated state; stored combatants left out are removed after the GM confirms"`
	AddPartyCharacterIDs []string       `json:"add_party_character_ids,omitempty" jsonschema:"character_id values from list_party_members to add"`
	AddMonsters          []DraftMonster `json:"add_monsters,omitempty" jsonschema:"monsters to add"`
}

type saveTemplateInput struct {
	Monster MonsterInput `json:"monster"`
}

type templateSaved struct {
	Template         MonsterTemplate `json:"template"`
	ReplacedExisting bool            `json:"replaced_existing" jsonschema:"true when a template with the same name (case-insensitive) was replaced"`
}

func (t templateSaved) Summary() string {
	verb := "Saved"
	if t.ReplacedExisting {
		verb = "Replaced"
	}
	return fmt.Sprintf("%s monster template «%s» (%s).", verb, t.Template.Name, t.Template.TemplateID)
}

type encounterDeleted struct {
	EncounterID string `json:"encounter_id"`
	Name        string `json:"name"`
}

func (e encounterDeleted) Summary() string {
	return fmt.Sprintf("Deleted encounter «%s» (%s).", e.Name, e.EncounterID)
}

func registerEncounterTools(s *mcp.Server, d *deps) {
	addTool(s, d, toolDef{Name: "preview_encounter_difficulty", Title: "Preview encounter difficulty", Effect: effectRead,
		Description: "Compute the difficulty of a draft roster without saving anything, the same way the desktop encounter editor does and create_encounter stores it. " + termsDraft + " " + termsDifficulty},
		d.previewEncounterDifficulty)
	addTool(s, d, toolDef{Name: "create_encounter", Title: "Create an encounter", Effect: effectModify,
		Description: "Create an encounter in the active campaign from party members and monsters; it needs at least one combatant. Combatants are put in initiative order and Party AP and GM Threat come from the campaign. " +
			termsEncounterActive + " " + termsDraft + " " + termsDifficulty},
		d.createEncounter)
	addTool(s, d, toolDef{Name: "activate_encounter", Title: "Activate an encounter", Effect: effectModify, Idempotent: true,
		Description: "Make an encounter of the active campaign the active encounter, the one combat actions apply to. Logs \"Encounter activated\"."},
		d.activateEncounter)
	addTool(s, d, toolDef{Name: "save_monster_template", Title: "Save a monster template", Effect: effectModify, Idempotent: true,
		Description: "Save a monster as a template for later encounters, in the assistant and the desktop app. A template with the same name (case-insensitive) is replaced, and replaced_existing says so. " +
			"Level is 1 or more; current HP is 0 or more and no more than max HP. Torso-only creatures are always hit on the torso and use torso DR. " + termsDR},
		d.saveMonsterTemplate)
	addTool(s, d, toolDef{Name: "update_encounter", Title: "Update an encounter", Effect: effectDestroy,
		Description: "Replace an encounter's name and roster. List every stored combatant to keep in keep_combatant_ids: kept combatants keep their current HP and defeated state. " +
			"Stored combatants left out are removed, and the GM must confirm the removal; a rename or additions alone need no confirmation. " +
			"add_party_character_ids and add_monsters add combatants as in create_encounter. " + termsEncounterActive + " " + termsDraft},
		d.updateEncounter)
	addTool(s, d, toolDef{Name: "restart_encounter", Title: "Restart an encounter", Effect: effectDestroy,
		Description: "Restart an encounter. Only NPC HP (back to max HP) and NPC defeated state reset; party members keep their HP and defeated state. " +
			"The round goes back to 1 and the encounter becomes the active encounter. The GM must confirm."},
		d.restartEncounter)
	addTool(s, d, toolDef{Name: "delete_encounter", Title: "Delete an encounter", Effect: effectDestroy,
		Description: "Delete an encounter of the active campaign. It is no longer listed and cannot be read afterwards. The GM must confirm."},
		d.deleteEncounter)
}

// draftMonsters converts the monster entries of the argument field.
func draftMonsters(field string, in []DraftMonster) ([]app.DraftMonster, error) {
	out := make([]app.DraftMonster, 0, len(in))
	for i, m := range in {
		at := fmt.Sprintf("%s[%d]", field, i)
		if m.Quantity < 1 || m.Quantity > app.MaxDraftMonsterQuantity {
			return nil, invalidInput(at+".quantity", fmt.Sprintf("%s.quantity must be between 1 and %d", at, app.MaxDraftMonsterQuantity))
		}
		if (strings.TrimSpace(m.TemplateID) == "") == (m.Monster == nil) {
			return nil, invalidInput(at, at+" must set exactly one of template_id or monster")
		}
		dm := app.DraftMonster{TemplateID: m.TemplateID, Quantity: m.Quantity, Initiative: m.Initiative}
		if m.Monster != nil {
			c := monsterCombatant(*m.Monster)
			dm.Monster = &c
		}
		out = append(out, dm)
	}
	return out, nil
}

func (d *deps) previewEncounterDifficulty(ctx context.Context, _ *mcp.CallToolRequest, in previewInput) (*mcp.CallToolResult, Difficulty, error) {
	monsters, err := draftMonsters("monsters", in.Monsters)
	if err != nil {
		return nil, Difficulty{}, err
	}
	m, err := d.svc.PreviewEncounterDifficulty(ctx, app.EncounterDraft{PartyCharacterIDs: in.PartyCharacterIDs, Monsters: monsters})
	if err != nil {
		return nil, Difficulty{}, err
	}
	return nil, difficultyDTO(m), nil
}

func (d *deps) createEncounter(ctx context.Context, _ *mcp.CallToolRequest, in createEncounterInput) (res *mcp.CallToolResult, out Encounter, err error) {
	a := activity{Operation: "create_encounter", Kind: domain.EntityEncounter, Label: in.Name}
	defer func() { d.finish(ctx, a, res, err) }()

	if strings.TrimSpace(in.Name) == "" {
		return nil, out, invalidInput("name", "name is required")
	}
	monsters, err := draftMonsters("monsters", in.Monsters)
	if err != nil {
		return nil, out, err
	}
	enc, err := d.svc.ExecuteCreateEncounterFromDraft(ctx, app.EncounterDraft{Name: in.Name, PartyCharacterIDs: in.PartyCharacterIDs, Monsters: monsters})
	if err != nil {
		return nil, out, err
	}
	a.ID, a.Label = enc.ID, enc.Name
	return nil, encounterDTO(enc), nil
}

func (d *deps) activateEncounter(ctx context.Context, _ *mcp.CallToolRequest, in encounterArg) (res *mcp.CallToolResult, out Encounter, err error) {
	a := activity{Operation: "activate_encounter", Kind: domain.EntityEncounter, ID: in.EncounterID}
	defer func() { d.finish(ctx, a, res, err) }()

	if strings.TrimSpace(in.EncounterID) == "" {
		return nil, out, invalidInput("encounter_id", "encounter_id is required")
	}
	enc, err := d.svc.ActivateEncounter(ctx, in.EncounterID)
	if err != nil {
		return nil, out, err
	}
	a.Label = enc.Name
	return nil, encounterDTO(enc), nil
}

func (d *deps) saveMonsterTemplate(ctx context.Context, _ *mcp.CallToolRequest, in saveTemplateInput) (res *mcp.CallToolResult, out templateSaved, err error) {
	a := activity{Operation: "save_monster_template", Kind: domain.EntityMonsterTemplate, Label: in.Monster.Name}
	defer func() { d.finish(ctx, a, res, err) }()

	saved, err := d.svc.ExecuteSaveMonsterTemplates(ctx, app.SaveMonsterTemplatesCommand{Monsters: []domain.Combatant{monsterCombatant(in.Monster)}})
	if err != nil {
		return nil, out, err
	}
	if len(saved) != 1 {
		return nil, out, fmt.Errorf("saving one monster template returned %d templates", len(saved))
	}
	t := saved[0]
	a.ID, a.Label = t.Template.ID, t.Template.Name
	if t.ReplacedExisting {
		a.Detail = "replaced the existing template"
	}
	return nil, templateSaved{Template: monsterTemplateDTO(t.Template), ReplacedExisting: t.ReplacedExisting}, nil
}

func (d *deps) updateEncounter(ctx context.Context, req *mcp.CallToolRequest, in updateEncounterInput) (res *mcp.CallToolResult, out Encounter, err error) {
	a := activity{Operation: "update_encounter", Kind: domain.EntityEncounter, ID: in.EncounterID}
	defer func() { d.finish(ctx, a, res, err) }()

	draft := app.EncounterDraft{PartyCharacterIDs: in.AddPartyCharacterIDs, KeepCombatantIDs: in.KeepCombatantIDs}
	if in.Name != nil {
		if strings.TrimSpace(*in.Name) == "" {
			return nil, out, invalidInput("name", "name must not be empty")
		}
		draft.Name = *in.Name
	}
	if draft.Monsters, err = draftMonsters("add_monsters", in.AddMonsters); err != nil {
		return nil, out, err
	}
	stored, err := d.storedEncounter(ctx, in.EncounterID)
	if err != nil {
		return nil, out, err
	}
	a.Label = stored.Name
	removed, err := removedCombatants(stored, in.KeepCombatantIDs)
	if err != nil {
		return nil, out, err
	}
	if len(removed) > 0 {
		names := make([]string, 0, len(removed))
		for _, c := range removed {
			names = append(names, c.Name)
		}
		var ok bool
		res, ok, err = d.confirm.require(ctx, req, in, app.ConfirmationRequest{
			Operation:   a.Operation,
			EntityKind:  a.Kind,
			EntityID:    stored.ID,
			EntityLabel: stored.Name,
			Summary:     fmt.Sprintf("Update encounter «%s»: remove %d combatant(s) (%s)?", stored.Name, len(removed), strings.Join(names, ", ")),
		})
		if !ok {
			return res, out, err
		}
		// The GM confirmed these removals only: refuse if the roster changed
		// while the confirmation was open.
		if err = d.sameRemovals(ctx, stored.ID, in.KeepCombatantIDs, removed); err != nil {
			return nil, out, err
		}
	}
	enc, err := d.svc.ExecuteUpdateEncounterFromDraft(ctx, stored.ID, draft)
	if err != nil {
		return nil, out, err
	}
	a.Label = enc.Name
	return nil, encounterDTO(enc), nil
}

func (d *deps) restartEncounter(ctx context.Context, req *mcp.CallToolRequest, in encounterArg) (res *mcp.CallToolResult, out Encounter, err error) {
	a := activity{Operation: "restart_encounter", Kind: domain.EntityEncounter, ID: in.EncounterID}
	defer func() { d.finish(ctx, a, res, err) }()

	stored, err := d.storedEncounter(ctx, in.EncounterID)
	if err != nil {
		return nil, out, err
	}
	a.Label = stored.Name
	res, ok, err := d.confirm.require(ctx, req, in, app.ConfirmationRequest{
		Operation:   a.Operation,
		EntityKind:  a.Kind,
		EntityID:    stored.ID,
		EntityLabel: stored.Name,
		Summary:     fmt.Sprintf("Restart encounter «%s»? NPC HP resets to max and the round goes back to 1.", stored.Name),
	})
	if !ok {
		return res, out, err
	}
	enc, err := d.svc.RestartEncounter(ctx, stored.ID)
	if err != nil {
		return nil, out, err
	}
	return nil, encounterDTO(enc), nil
}

func (d *deps) deleteEncounter(ctx context.Context, req *mcp.CallToolRequest, in encounterArg) (res *mcp.CallToolResult, out encounterDeleted, err error) {
	a := activity{Operation: "delete_encounter", Kind: domain.EntityEncounter, ID: in.EncounterID}
	defer func() { d.finish(ctx, a, res, err) }()

	stored, err := d.storedEncounter(ctx, in.EncounterID)
	if err != nil {
		return nil, out, err
	}
	a.Label = stored.Name
	res, ok, err := d.confirm.require(ctx, req, in, app.ConfirmationRequest{
		Operation:   a.Operation,
		EntityKind:  a.Kind,
		EntityID:    stored.ID,
		EntityLabel: stored.Name,
		Summary:     fmt.Sprintf("Delete encounter «%s» (round %d, %d combatants)?", stored.Name, stored.Round, len(stored.Combatants)),
	})
	if !ok {
		return res, out, err
	}
	if err = d.svc.DeleteEncounter(ctx, stored.ID); err != nil {
		return nil, out, err
	}
	return nil, encounterDeleted{EncounterID: stored.ID, Name: stored.Name}, nil
}

// storedEncounter reads the encounter that confirmation text is built from.
func (d *deps) storedEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	if strings.TrimSpace(encounterID) == "" {
		return nil, invalidInput("encounter_id", "encounter_id is required")
	}
	return d.svc.GetEncounterByID(ctx, encounterID)
}

// removedCombatants returns the combatants of enc missing from keep. An ID in
// keep that enc does not have is not found.
func removedCombatants(enc *domain.Encounter, keep []string) ([]domain.Combatant, error) {
	for _, id := range keep {
		if !slices.ContainsFunc(enc.Combatants, func(c domain.Combatant) bool { return c.ID == strings.TrimSpace(id) }) {
			return nil, notFound(fmt.Sprintf("combatant %q not found in encounter «%s»", id, enc.Name))
		}
	}
	var removed []domain.Combatant
	for _, c := range enc.Combatants {
		if !slices.ContainsFunc(keep, func(id string) bool { return strings.TrimSpace(id) == c.ID }) {
			removed = append(removed, c)
		}
	}
	return removed, nil
}

// sameRemovals reports an error unless keep still removes exactly the
// confirmed combatants from the stored encounter.
func (d *deps) sameRemovals(ctx context.Context, encounterID string, keep []string, confirmed []domain.Combatant) error {
	enc, err := d.svc.GetEncounterByID(ctx, encounterID)
	if err != nil {
		return err
	}
	removed, err := removedCombatants(enc, keep)
	if err != nil {
		return err
	}
	if !slices.EqualFunc(removed, confirmed, func(x, y domain.Combatant) bool { return x.ID == y.ID }) {
		return invalidInput("keep_combatant_ids", "the encounter's combatants changed while the GM was confirming; nothing was changed. Read the encounter again and retry")
	}
	return nil
}
