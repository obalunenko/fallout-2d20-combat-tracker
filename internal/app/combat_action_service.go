package app

import (
	"context"
	"fmt"

	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) ApplyDamage(ctx context.Context, combatantID string, damageType domain.DamageType, location domain.BodyLocation, amount int) (*domain.Encounter, int, error) {
	return s.ExecuteApplyDamage(ctx, ApplyDamageCommand{
		CombatantID: combatantID,
		DamageType:  damageType,
		Location:    location,
		Amount:      amount,
	})
}

func (s *Service) ExecuteApplyDamage(ctx context.Context, cmd ApplyDamageCommand) (*domain.Encounter, int, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if cmd.CombatantID == "" {
		return nil, 0, fmt.Errorf("combatant id is required")
	}
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, 0, err
	}

	applied, err := enc.ApplyDamage(cmd.CombatantID, cmd.DamageType, cmd.Location, cmd.Amount)
	if err != nil {
		return nil, 0, err
	}

	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, 0, err
	}
	targetLabel := cmd.CombatantID
	if combatant := findCombatantByID(enc, cmd.CombatantID); combatant != nil {
		targetLabel = combatant.Name
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Damage -> %s type:%s location:%s raw:%d applied:%d", targetLabel, cmd.DamageType, cmd.Location, cmd.Amount, applied))
	return enc, applied, nil
}

func (s *Service) Heal(ctx context.Context, combatantID string, amount int) (*domain.Encounter, int, error) {
	return s.ExecuteHeal(ctx, HealCommand{
		CombatantID: combatantID,
		Amount:      amount,
	})
}

func (s *Service) ExecuteHeal(ctx context.Context, cmd HealCommand) (*domain.Encounter, int, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if cmd.CombatantID == "" {
		return nil, 0, fmt.Errorf("combatant id is required")
	}
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, 0, err
	}

	healed, err := enc.Heal(cmd.CombatantID, cmd.Amount)
	if err != nil {
		return nil, 0, err
	}

	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, 0, err
	}
	targetLabel := cmd.CombatantID
	if combatant := findCombatantByID(enc, cmd.CombatantID); combatant != nil {
		targetLabel = combatant.Name
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Heal -> %s value:%d", targetLabel, healed))
	return enc, healed, nil
}

func findCombatantByID(enc *domain.Encounter, combatantID string) *domain.Combatant {
	if enc == nil || combatantID == "" {
		return nil
	}
	for i := range enc.Combatants {
		if enc.Combatants[i].ID == combatantID {
			return &enc.Combatants[i]
		}
	}
	return nil
}
