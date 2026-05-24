package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.ListMonsterTemplates(ctx)
}

func (s *Service) SaveMonsterTemplates(ctx context.Context, monsters []domain.Combatant) ([]domain.Combatant, error) {
	return s.ExecuteSaveMonsterTemplates(ctx, SaveMonsterTemplatesCommand{Monsters: monsters})
}

func (s *Service) ExecuteSaveMonsterTemplates(ctx context.Context, cmd SaveMonsterTemplatesCommand) ([]domain.Combatant, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if len(cmd.Monsters) == 0 {
		return nil, fmt.Errorf("add at least one monster")
	}

	saved := make([]domain.Combatant, 0, len(cmd.Monsters))
	seen := make(map[string]struct{}, len(cmd.Monsters))
	for i := range cmd.Monsters {
		monster, err := prepareMonsterTemplate(cmd.Monsters[i])
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(monster.Name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		created, err := s.repo.UpsertMonsterTemplate(ctx, monster)
		if err != nil {
			return nil, err
		}
		saved = append(saved, created)
	}
	return saved, nil
}

func prepareMonsterTemplate(monster domain.Combatant) (domain.Combatant, error) {
	monster.Name = strings.TrimSpace(monster.Name)
	if monster.Name == "" {
		return domain.Combatant{}, fmt.Errorf("monster name is required")
	}
	if err := domain.ValidateCombatant(monster, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("monster %q", monster.Name),
		RequireName: true,
		MinLevel:    1,
	}); err != nil {
		return domain.Combatant{}, err
	}
	if strings.TrimSpace(monster.ID) == "" {
		monster.ID = uuid.NewString()
	}
	monster.Side = domain.SideNPC
	monster.PlayerCharacterID = ""
	monster.Active = false
	monster.Defeated = false
	domain.NormalizeCombatantHP(&monster)
	return monster, nil
}
