package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error) {
	ctx, op, err := s.beginRead(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	return s.repo.ListMonsterTemplates(ctx)
}

func (s *Service) SaveMonsterTemplates(ctx context.Context, monsters []domain.Combatant) ([]domain.Combatant, error) {
	saved, err := s.ExecuteSaveMonsterTemplates(ctx, SaveMonsterTemplatesCommand{Monsters: monsters})
	if err != nil {
		return nil, err
	}
	templates := make([]domain.Combatant, len(saved))
	for i := range saved {
		templates[i] = saved[i].Template
	}
	return templates, nil
}

// SavedMonsterTemplate is a stored template and whether it replaced a
// template with the same name (case-insensitive).
type SavedMonsterTemplate struct {
	Template         domain.Combatant
	ReplacedExisting bool
}

// ExecuteSaveMonsterTemplates stores monsters as templates. A monster
// replaces the template with the same name (case-insensitive).
func (s *Service) ExecuteSaveMonsterTemplates(ctx context.Context, cmd SaveMonsterTemplatesCommand) ([]SavedMonsterTemplate, error) {
	ctx, op, err := s.beginWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer op.end()
	if len(cmd.Monsters) == 0 {
		return nil, invalidf("add at least one monster")
	}

	existing, err := s.repo.ListMonsterTemplates(ctx)
	if err != nil {
		return nil, err
	}
	stored := make(map[string]bool, len(existing))
	for _, t := range existing {
		stored[strings.ToLower(strings.TrimSpace(t.Name))] = true
	}
	saved := make([]SavedMonsterTemplate, 0, len(cmd.Monsters))
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
		saved = append(saved, SavedMonsterTemplate{Template: created, ReplacedExisting: stored[key]})
	}
	op.changed(ctx, ChangeEvent{Kinds: []domain.AssistantEntityKind{domain.EntityMonsterTemplate}})
	return saved, nil
}

func prepareMonsterTemplate(monster domain.Combatant) (domain.Combatant, error) {
	monster.Name = strings.TrimSpace(monster.Name)
	if monster.Name == "" {
		return domain.Combatant{}, invalidf("monster name is required")
	}
	if err := domain.ValidateCombatant(monster, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("monster %q", monster.Name),
		RequireName: true,
		MinLevel:    1,
	}); err != nil {
		return domain.Combatant{}, domain.InvalidInput(err)
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
