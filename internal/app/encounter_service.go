package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) GetEncounter(ctx context.Context) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.Get(ctx)
}

func (s *Service) ListEncounters(ctx context.Context) ([]domain.EncounterSummary, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.List(ctx)
}

func (s *Service) ListPartyMembers(ctx context.Context) ([]domain.Combatant, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.ListPartyMembers(ctx)
}

func (s *Service) GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.GetEncounterByID(ctx, encounterID)
}

func (s *Service) ActivateEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if err := s.repo.Activate(ctx, encounterID); err != nil {
		return nil, err
	}
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, "Encounter activated")
	return enc, nil
}

func (s *Service) RestartEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
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
	return restarted, nil
}

func (s *Service) DeleteEncounter(ctx context.Context, encounterID string) error {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return fmt.Errorf("encounter id is required")
	}
	return s.repo.SoftDelete(ctx, encounterID)
}

func (s *Service) CreateEncounter(ctx context.Context, id, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteCreateEncounter(ctx, CreateEncounterCommand{
		ID:         id,
		Name:       name,
		Combatants: combatants,
	})
}

func (s *Service) ExecuteCreateEncounter(ctx context.Context, cmd CreateEncounterCommand) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if len(cmd.Combatants) == 0 {
		return nil, fmt.Errorf("cannot create encounter without combatants")
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
	return enc, nil
}

func (s *Service) UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteUpdateEncounter(ctx, UpdateEncounterCommand{
		EncounterID: encounterID,
		Name:        name,
		Combatants:  combatants,
	})
}

func (s *Service) ExecuteUpdateEncounter(ctx context.Context, cmd UpdateEncounterCommand) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.EncounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("encounter name is required")
	}
	if len(cmd.Combatants) == 0 {
		return nil, fmt.Errorf("cannot update encounter without combatants")
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
	return enc, nil
}

func (s *Service) AdvanceTurn(ctx context.Context) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := enc.AdvanceTurn(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	if active := enc.ActiveCombatant(); active != nil {
		s.appendOperationLog(ctx, enc, fmt.Sprintf("Turn advanced -> %s", active.Name))
	}
	return enc, nil
}

func (s *Service) AddPartyAP(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.updateActiveEncounterResources(ctx, func(resources *domain.Resources) error {
		resources.AddPartyAP(v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Party AP %+d (total: %d)", v, enc.Resources.PartyAP))
	return enc, nil
}

func (s *Service) SpendPartyAP(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.updateActiveEncounterResources(ctx, func(resources *domain.Resources) error {
		return resources.SpendPartyAP(v)
	})
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Party AP -%d (total: %d)", v, enc.Resources.PartyAP))
	return enc, nil
}

func (s *Service) AddThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.updateActiveEncounterResources(ctx, func(resources *domain.Resources) error {
		resources.AddThreat(v)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("GM Threat %+d (total: %d)", v, enc.Resources.GMThreat))
	return enc, nil
}

func (s *Service) SpendThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.updateActiveEncounterResources(ctx, func(resources *domain.Resources) error {
		return resources.SpendThreat(v)
	})
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("GM Threat -%d (total: %d)", v, enc.Resources.GMThreat))
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
			return nil, err
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
	return domain.ValidateCombatant(c, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("combatant %q", label),
		RequireName: true,
		RequireSide: true,
		MinLevel:    0,
	})
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
