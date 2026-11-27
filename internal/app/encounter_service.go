package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) GetEncounter(ctx context.Context) (*domain.Encounter, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.Get(ctx)
}

func (s *Service) ListEncounters(ctx context.Context) ([]domain.EncounterSummary, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.List(ctx)
}

func (s *Service) ListPartyMembers(ctx context.Context) ([]domain.Combatant, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.ListPartyMembers(ctx)
}

func (s *Service) GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if strings.TrimSpace(encounterID) == "" {
		return nil, invalidf("encounter id is required")
	}
	return s.repo.GetEncounterByID(ctx, encounterID)
}

func (s *Service) ActivateEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if encounterID == "" {
		return nil, invalidf("encounter id is required")
	}
	if err := s.repo.Activate(ctx, encounterID); err != nil {
		return nil, err
	}
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, "Encounter activated")
	op.changed(ctx, encounterEvent(enc))
	return enc, nil
}

func (s *Service) RestartEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if encounterID == "" {
		return nil, invalidf("encounter id is required")
	}
	if err := s.repo.Activate(ctx, encounterID); err != nil {
		return nil, err
	}

	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	for i := range enc.Combatants {
		if enc.Combatants[i].Side != domain.SideNPC {
			continue
		}
		domain.NormalizeCombatantHP(&enc.Combatants[i])
		enc.Combatants[i].HP = enc.Combatants[i].MaxHP
		enc.Combatants[i].Defeated = false
	}
	restarted := domain.NewEncounter(enc.ID, enc.Name, enc.Combatants)
	restarted.CampaignID = enc.CampaignID
	restarted.Resources = enc.Resources
	if err := s.repo.Save(ctx, restarted); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, restarted, "Encounter restarted")
	s.bumpRevisions(ctx, encounterChangeKeys(restarted)...)
	op.changed(ctx, encounterEvent(restarted))
	return restarted, nil
}

func (s *Service) DeleteEncounter(ctx context.Context, encounterID string) error {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return err
	}
	defer op.end()
	if encounterID == "" {
		return invalidf("encounter id is required")
	}
	if err := s.repo.SoftDelete(ctx, encounterID); err != nil {
		return err
	}
	s.tombstone(ctx, encounterKey(encounterID))
	op.changed(ctx, ChangeEvent{Kinds: []domain.AssistantEntityKind{domain.EntityEncounter}, EncounterID: encounterID})
	return nil
}

func (s *Service) CreateEncounter(ctx context.Context, id, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteCreateEncounter(ctx, CreateEncounterCommand{
		ID:         id,
		Name:       name,
		Combatants: combatants,
	})
}

func (s *Service) ExecuteCreateEncounter(ctx context.Context, cmd CreateEncounterCommand) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.createEncounter(ctx, op, cmd)
}

// createEncounter creates an encounter in the active campaign. The caller
// holds the write gate.
func (s *Service) createEncounter(ctx context.Context, op *operation, cmd CreateEncounterCommand) (*domain.Encounter, error) {
	if len(cmd.Combatants) == 0 {
		return nil, invalidf("cannot create encounter without combatants")
	}
	activeCampaign, err := s.repo.GetActiveCampaign(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		cmd.ID = uuid.NewString()
	}
	if err := prepareEncounterCombatants(cmd.Combatants); err != nil {
		return nil, err
	}
	if err := s.resolveLinkedPartyCombatants(ctx, cmd.Combatants); err != nil {
		return nil, err
	}
	enc := domain.NewEncounter(cmd.ID, cmd.Name, cmd.Combatants)
	enc.CampaignID = activeCampaign.ID
	enc.Resources = activeCampaign.Resources
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Encounter created (%s)", cmd.Name))
	s.bumpRevisions(ctx, encounterChangeKeys(enc)...)
	op.changed(ctx, encounterEvent(enc))
	return enc, nil
}

func (s *Service) UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteUpdateEncounter(ctx, UpdateEncounterCommand{
		EncounterID: encounterID,
		Name:        name,
		Combatants:  combatants,
	})
}

// ExecuteUpdateEncounter replaces the encounter's name and roster. When
// cmd.ExpectedRevision is set and the assistant changed or deleted the
// encounter after that revision, it saves nothing and returns
// domain.ErrStaleEdit or domain.ErrEntityDeleted.
func (s *Service) ExecuteUpdateEncounter(ctx context.Context, cmd UpdateEncounterCommand) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if err := s.checkRevision(encounterKey(cmd.EncounterID), cmd.ExpectedRevision); err != nil {
		return nil, err
	}
	return s.updateEncounter(ctx, op, cmd)
}

// updateEncounter replaces an encounter's name and roster. The caller holds
// the write gate.
func (s *Service) updateEncounter(ctx context.Context, op *operation, cmd UpdateEncounterCommand) (*domain.Encounter, error) {
	if strings.TrimSpace(cmd.EncounterID) == "" {
		return nil, invalidf("encounter id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, invalidf("encounter name is required")
	}
	if len(cmd.Combatants) == 0 {
		return nil, invalidf("cannot update encounter without combatants")
	}
	if err := prepareEncounterCombatants(cmd.Combatants); err != nil {
		return nil, err
	}
	if err := s.resolveLinkedPartyCombatants(ctx, cmd.Combatants); err != nil {
		return nil, err
	}
	enc, err := s.repo.UpdateEncounter(ctx, cmd.EncounterID, cmd.Name, cmd.Combatants)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Encounter updated (%s)", cmd.Name))
	s.bumpRevisions(ctx, encounterChangeKeys(enc)...)
	op.changed(ctx, encounterEvent(enc))
	return enc, nil
}

func (s *Service) AdvanceTurn(ctx context.Context) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := enc.AdvanceTurn(); err != nil {
		return nil, domain.InvalidInput(err)
	}
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	if active := enc.ActiveCombatant(); active != nil {
		s.appendOperationLog(ctx, enc, fmt.Sprintf("Turn advanced -> %s", active.Name))
	}
	s.bumpRevisions(ctx, encounterChangeKeys(enc)...)
	op.changed(ctx, encounterEvent(enc))
	return enc, nil
}

// AddPartyAP adds v party AP to the active campaign. capped reports that the
// maximum kept part of v out.
func (s *Service) AddPartyAP(ctx context.Context, v int) (enc *domain.Encounter, capped bool, err error) {
	enc, err = s.changeResources(ctx, func(resources *domain.Resources) error {
		capped = resources.AddPartyAP(v)
		return nil
	}, func(enc *domain.Encounter) string {
		return fmt.Sprintf("Party AP %+d (total: %d)", v, enc.Resources.PartyAP)
	})
	if err != nil {
		return nil, false, err
	}
	return enc, capped, nil
}

func (s *Service) SpendPartyAP(ctx context.Context, v int) (*domain.Encounter, error) {
	return s.changeResources(ctx, func(resources *domain.Resources) error {
		return resources.SpendPartyAP(v)
	}, func(enc *domain.Encounter) string {
		return fmt.Sprintf("Party AP -%d (total: %d)", v, enc.Resources.PartyAP)
	})
}

func (s *Service) AddThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	return s.changeResources(ctx, func(resources *domain.Resources) error {
		resources.AddThreat(v)
		return nil
	}, func(enc *domain.Encounter) string {
		return fmt.Sprintf("GM Threat %+d (total: %d)", v, enc.Resources.GMThreat)
	})
}

func (s *Service) SpendThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	return s.changeResources(ctx, func(resources *domain.Resources) error {
		return resources.SpendThreat(v)
	}, func(enc *domain.Encounter) string {
		return fmt.Sprintf("GM Threat -%d (total: %d)", v, enc.Resources.GMThreat)
	})
}

// changeResources applies update to the active campaign's shared resources
// under the write gate and logs logMessage on success.
func (s *Service) changeResources(ctx context.Context, update func(*domain.Resources) error, logMessage func(*domain.Encounter) string) (*domain.Encounter, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	enc, err := s.updateActiveEncounterResources(ctx, update)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, logMessage(enc))
	op.changed(ctx, ChangeEvent{
		Kinds:       []domain.AssistantEntityKind{domain.EntityCampaign, domain.EntityEncounter},
		CampaignID:  enc.CampaignID,
		EncounterID: enc.ID,
	})
	return enc, nil
}

func (s *Service) updateActiveEncounterResources(ctx context.Context, update func(*domain.Resources) error) (*domain.Encounter, error) {
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	resources := enc.Resources
	if update != nil {
		if err := update(&resources); err != nil {
			return nil, domain.InvalidInput(err)
		}
	}
	if err := s.repo.UpdateCampaignResources(ctx, enc.CampaignID, resources); err != nil {
		return nil, err
	}
	enc.Resources = resources
	return enc, nil
}

func prepareEncounterCombatants(combatants []domain.Combatant) error {
	for i := range combatants {
		combatants[i].Name = strings.TrimSpace(combatants[i].Name)
		if combatants[i].Side == "" {
			combatants[i].Side = domain.SideNPC
		}
		if err := validateEncounterCombatant(i, combatants[i]); err != nil {
			return err
		}
		if strings.TrimSpace(combatants[i].ID) == "" {
			combatants[i].ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&combatants[i])
	}
	return nil
}

func validateEncounterCombatant(index int, c domain.Combatant) error {
	label := strings.TrimSpace(c.Name)
	if label == "" {
		label = strings.TrimSpace(c.ID)
	}
	if label == "" {
		label = fmt.Sprintf("#%d", index+1)
	}
	return domain.InvalidInput(domain.ValidateCombatant(c, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("combatant %q", label),
		RequireName: true,
		RequireSide: true,
		MinLevel:    0,
	}))
}

func (s *Service) resolveLinkedPartyCombatants(ctx context.Context, combatants []domain.Combatant) error {
	if !hasLinkedPartyCombatant(combatants) {
		return nil
	}

	partyMembers, err := s.repo.ListPartyMembers(ctx)
	if err != nil {
		return err
	}
	partyByID := make(map[string]domain.Combatant, len(partyMembers)*2)
	for _, member := range partyMembers {
		if id := strings.TrimSpace(member.PlayerCharacterID); id != "" {
			partyByID[id] = member
		}
		if id := strings.TrimSpace(member.ID); id != "" {
			partyByID[id] = member
		}
	}

	for i := range combatants {
		playerCharacterID := linkedPlayerCharacterID(combatants[i])
		if playerCharacterID == "" {
			continue
		}
		member, ok := partyByID[playerCharacterID]
		if !ok {
			continue
		}

		encounterCombatantID := strings.TrimSpace(combatants[i].ID)
		member.PlayerCharacterID = playerCharacterID
		member.Active = combatants[i].Active
		if encounterCombatantID != "" && encounterCombatantID != playerCharacterID {
			member.ID = encounterCombatantID
		} else {
			member.ID = playerCharacterID
		}
		combatants[i] = member
	}
	return nil
}

func hasLinkedPartyCombatant(combatants []domain.Combatant) bool {
	for _, combatant := range combatants {
		if linkedPlayerCharacterID(combatant) != "" {
			return true
		}
	}
	return false
}

func linkedPlayerCharacterID(combatant domain.Combatant) string {
	if combatant.Side != domain.SideParty {
		return ""
	}
	if playerCharacterID := strings.TrimSpace(combatant.PlayerCharacterID); playerCharacterID != "" {
		return playerCharacterID
	}
	return strings.TrimSpace(combatant.ID)
}
