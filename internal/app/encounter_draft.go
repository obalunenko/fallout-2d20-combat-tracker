package app

import (
	"context"
	"strings"

	"github.com/obalunenko/fallout/internal/domain"
)

// MaxDraftMonsterQuantity is the largest number of copies one DraftMonster
// may add to a roster.
const MaxDraftMonsterQuantity = 50

// EncounterDraft describes an encounter roster by reference: party members
// by player character ID, monsters by template or inline stats, and, for an
// update, the stored combatants to keep.
type EncounterDraft struct {
	// Name is the encounter name. An update keeps the stored name when it is
	// empty.
	Name string
	// PartyCharacterIDs are player characters of the active campaign that
	// are available for encounters.
	PartyCharacterIDs []string
	Monsters          []DraftMonster
	// KeepCombatantIDs are stored combatants an update keeps with their
	// current state, such as HP and defeated. Stored combatants not listed
	// are removed.
	KeepCombatantIDs []string
}

// DraftMonster adds Quantity copies of one monster to a draft. Set exactly
// one of TemplateID and Monster.
type DraftMonster struct {
	TemplateID string
	Monster    *domain.Combatant
	// Quantity is the number of copies, 1 to MaxDraftMonsterQuantity. Each
	// copy becomes its own combatant.
	Quantity int
	// Initiative, when set, replaces the monster's initiative.
	Initiative *int
}

// PreviewEncounterDifficulty returns the difficulty of draft's roster
// without storing anything. It expands the draft as
// ExecuteCreateEncounterFromDraft does.
func (s *Service) PreviewEncounterDifficulty(ctx context.Context, draft EncounterDraft) (domain.EncounterDifficultyMetrics, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return domain.EncounterDifficultyMetrics{}, err
	}
	defer op.end()
	combatants, err := s.expandDraft(ctx, draft, nil)
	if err != nil {
		return domain.EncounterDifficultyMetrics{}, err
	}
	return domain.EvaluateEncounterDifficulty(combatants), nil
}

// ExecuteCreateEncounterFromDraft expands draft and creates the encounter in
// the active campaign as ExecuteCreateEncounter does, under one write gate
// hold.
func (s *Service) ExecuteCreateEncounterFromDraft(ctx context.Context, draft EncounterDraft) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	name := strings.TrimSpace(draft.Name)
	if name == "" {
		return nil, invalidf("encounter name is required")
	}
	combatants, err := s.expandDraft(ctx, draft, nil)
	if err != nil {
		return nil, err
	}
	return s.createEncounter(ctx, op, CreateEncounterCommand{Name: name, Combatants: combatants})
}

// ExecuteUpdateEncounterFromDraft replaces the name and roster of the stored
// encounter as ExecuteUpdateEncounter does, under one write gate hold. The
// roster is the kept stored combatants plus the draft's additions.
func (s *Service) ExecuteUpdateEncounterFromDraft(ctx context.Context, encounterID string, draft EncounterDraft) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if strings.TrimSpace(encounterID) == "" {
		return nil, invalidf("encounter id is required")
	}
	stored, err := s.repo.GetEncounterByID(ctx, encounterID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(draft.Name)
	if name == "" {
		name = stored.Name
	}
	combatants, err := s.expandDraft(ctx, draft, stored)
	if err != nil {
		return nil, err
	}
	return s.updateEncounter(ctx, op, UpdateEncounterCommand{EncounterID: stored.ID, Name: name, Combatants: combatants})
}

// expandDraft returns draft's roster: the kept combatants of stored (nil
// when creating), then the party members, then Quantity copies of each
// monster with empty IDs, as the desktop editor collects its rows. The
// caller holds the gate.
func (s *Service) expandDraft(ctx context.Context, draft EncounterDraft, stored *domain.Encounter) ([]domain.Combatant, error) {
	combatants, err := keptCombatants(stored, draft.KeepCombatantIDs)
	if err != nil {
		return nil, err
	}
	party, err := s.draftPartyMembers(ctx, draft.PartyCharacterIDs, combatants)
	if err != nil {
		return nil, err
	}
	combatants = append(combatants, party...)

	var templates map[string]domain.Combatant
	for i, m := range draft.Monsters {
		if m.Quantity < 1 || m.Quantity > MaxDraftMonsterQuantity {
			return nil, invalidf("monster #%d: quantity must be between 1 and %d", i+1, MaxDraftMonsterQuantity)
		}
		templateID := strings.TrimSpace(m.TemplateID)
		if (templateID == "") == (m.Monster == nil) {
			return nil, invalidf("monster #%d: exactly one of a template ID or an inline monster is required", i+1)
		}
		var monster domain.Combatant
		if m.Monster != nil {
			monster = *m.Monster
		} else {
			if templates == nil {
				if templates, err = s.templatesByID(ctx); err != nil {
					return nil, err
				}
			}
			t, ok := templates[templateID]
			if !ok {
				return nil, notFoundf("monster template %q not found", templateID)
			}
			monster = t
		}
		monster.ID, monster.PlayerCharacterID, monster.Side = "", "", domain.SideNPC
		monster.Active, monster.Defeated = false, false
		if m.Initiative != nil {
			monster.Initiative = *m.Initiative
		}
		for range m.Quantity {
			combatants = append(combatants, monster)
		}
	}
	return combatants, nil
}

// keptCombatants returns the combatants of stored listed in ids, in stored
// order and with their stored state.
func keptCombatants(stored *domain.Encounter, ids []string) ([]domain.Combatant, error) {
	missing := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, invalidf("kept combatant id is required")
		}
		if missing[id] {
			return nil, invalidf("duplicate kept combatant %q", id)
		}
		missing[id] = true
	}
	var kept []domain.Combatant
	if stored != nil {
		for _, c := range stored.Combatants {
			if missing[c.ID] {
				kept = append(kept, c)
				delete(missing, c.ID)
			}
		}
	}
	for _, id := range ids {
		if id = strings.TrimSpace(id); missing[id] {
			return nil, notFoundf("combatant %q not found in the encounter", id)
		}
	}
	return kept, nil
}

// draftPartyMembers returns the available party members with the given
// player character IDs. A character already in roster is a duplicate.
func (s *Service) draftPartyMembers(ctx context.Context, characterIDs []string, roster []domain.Combatant) ([]domain.Combatant, error) {
	if len(characterIDs) == 0 {
		return nil, nil
	}
	members, err := s.repo.ListPartyMembers(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.Combatant, len(members))
	for _, m := range members {
		byID[m.PlayerCharacterID] = m
	}
	seen := make(map[string]bool, len(roster)+len(characterIDs))
	for _, c := range roster {
		if id := linkedPlayerCharacterID(c); id != "" {
			seen[id] = true
		}
	}
	party := make([]domain.Combatant, 0, len(characterIDs))
	for _, id := range characterIDs {
		id = strings.TrimSpace(id)
		member, ok := byID[id]
		if !ok {
			return nil, notFoundf("party character %q not found among the active campaign's available characters", id)
		}
		if seen[id] {
			return nil, invalidf("duplicate party character %q: a character joins an encounter once", id)
		}
		seen[id] = true
		party = append(party, member)
	}
	return party, nil
}

func (s *Service) templatesByID(ctx context.Context) (map[string]domain.Combatant, error) {
	templates, err := s.repo.ListMonsterTemplates(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]domain.Combatant, len(templates))
	for _, t := range templates {
		byID[t.ID] = t
	}
	return byID, nil
}
