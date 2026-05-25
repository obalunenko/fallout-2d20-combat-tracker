package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

func (s *EncounterStore) ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error) {
	ctx = normalizeContext(ctx)
	rows, err := s.q.ListMonsterTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monster templates: %w", err)
	}
	monsters := make([]domain.Combatant, 0, len(rows))
	monsterIndexes := make(map[string]int, len(rows))
	for _, row := range rows {
		monsterIndexes[row.ID] = len(monsters)
		monsters = append(monsters, monsterTemplateFromRow(row))
	}
	if len(monsters) == 0 {
		return monsters, nil
	}

	profiles, err := monsterTemplateResistanceProfiles(ctx, s.q)
	if err != nil {
		return nil, err
	}
	for monsterID, profile := range profiles {
		idx, ok := monsterIndexes[monsterID]
		if !ok {
			continue
		}
		monsters[idx].SetResistanceProfile(profile)
	}
	return monsters, nil
}

func (s *EncounterStore) UpsertMonsterTemplate(ctx context.Context, monster domain.Combatant) (domain.Combatant, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(monster.Name) == "" {
		return domain.Combatant{}, fmt.Errorf("monster name is required")
	}
	if strings.TrimSpace(monster.ID) == "" {
		monster.ID = uuid.NewString()
	}
	monster.Side = domain.SideNPC
	monster.PlayerCharacterID = ""
	domain.NormalizeCombatantHP(&monster)

	if err := s.runInTx(ctx, func(qtx *dbgen.Queries) error {
		ids, err := normalizedDictionaryIDs(ctx, qtx)
		if err != nil {
			return err
		}
		templateID := monster.ID
		existingTemplateID, err := qtx.GetMonsterTemplateIDByName(ctx, monster.Name)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("get monster template id: %w", err)
		}
		if err == nil {
			templateID = existingTemplateID
		}
		monster.ID = templateID
		if err := upsertMonsterTemplateNormalizedStats(ctx, qtx, ids, templateID, monster.Profile()); err != nil {
			return fmt.Errorf("sync normalized monster template stats: %w", err)
		}
		if err := qtx.UpsertMonsterTemplate(ctx, upsertMonsterTemplateParams(templateID, monster)); err != nil {
			return fmt.Errorf("upsert monster template: %w", err)
		}
		return nil
	}); err != nil {
		return domain.Combatant{}, err
	}
	return monster, nil
}
