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

func (s *EncounterStore) Get(ctx context.Context) (*domain.Encounter, error) {
	ctx = normalizeContext(ctx)
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}

	encRow, err := s.q.GetLatestEncounterByCampaignID(ctx, campaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrEncounterNotInitialized
		}
		return nil, fmt.Errorf("read encounter: %w", err)
	}

	combatants, err := combatantsByEncounterID(ctx, s.q, encRow.ID)
	if err != nil {
		return nil, fmt.Errorf("read combatants: %w", err)
	}

	return encounterFromLatestRow(encRow, combatants), nil
}

func (s *EncounterStore) Save(ctx context.Context, enc *domain.Encounter) error {
	ctx = normalizeContext(ctx)
	if enc == nil {
		return fmt.Errorf("encounter cannot be nil")
	}

	if strings.TrimSpace(enc.CampaignID) == "" {
		campaignID, err := s.activeCampaignID(ctx)
		if err != nil {
			return err
		}
		enc.CampaignID = campaignID
	}

	return s.runInTx(ctx, func(qtx *dbgen.Queries) error {
		return saveEncounter(ctx, qtx, enc)
	})
}

func saveEncounter(ctx context.Context, qtx *dbgen.Queries, enc *domain.Encounter) error {
	if err := normalizeEncounterForSave(enc); err != nil {
		return err
	}
	ids, err := normalizedDictionaryIDs(ctx, qtx)
	if err != nil {
		return err
	}

	activeParty, err := activePartyCombatantsByID(ctx, qtx, enc.CampaignID)
	if err != nil {
		return fmt.Errorf("read campaign party characters: %w", err)
	}
	existingCombatants, err := combatantIDsByID(ctx, qtx, enc.ID)
	if err != nil {
		return fmt.Errorf("read encounter combatant ids: %w", err)
	}
	for i, c := range enc.Combatants {
		domain.NormalizeCombatantHP(&c)
		if c.Side == domain.SideParty {
			playerCharacterID := strings.TrimSpace(c.PlayerCharacterID)
			if playerCharacterID == "" {
				if _, ok := activeParty[c.ID]; ok {
					playerCharacterID = c.ID
				}
			}
			if partyCharacter, ok := activeParty[playerCharacterID]; ok {
				c.PlayerCharacterID = playerCharacterID
				if strings.TrimSpace(c.ID) == "" || c.ID == playerCharacterID {
					c.ID = uuid.NewString()
				}
				if _, existsInEncounter := existingCombatants[c.ID]; !existsInEncounter {
					partyCharacter.Active = c.Active
					partyCharacter.Defeated = partyCharacter.HP == 0
					partyCharacter.ID = c.ID
					partyCharacter.PlayerCharacterID = playerCharacterID
					c = partyCharacter
					enc.Combatants[i] = c
					continue
				}
				c.Side = domain.SideParty
				c.XP = 0
				c.Defeated = c.HP == 0
				if err = updateActivePlayerCharacter(ctx, qtx, playerCharacterID, c, false); err != nil {
					return fmt.Errorf("update campaign character %s: %w", playerCharacterID, err)
				}
				if err = upsertPlayerCharacterNormalizedStats(ctx, qtx, ids, playerCharacterID, c.Profile()); err != nil {
					return fmt.Errorf("sync campaign character stats %s: %w", playerCharacterID, err)
				}
			}
		}
		enc.Combatants[i] = c
	}

	if err = validateUniquePlayerCharacters(enc.Combatants); err != nil {
		return err
	}

	if err = qtx.UpsertEncounter(ctx, dbgen.UpsertEncounterParams{
		ID:         enc.ID,
		CampaignID: enc.CampaignID,
		Name:       enc.Name,
		Round:      int64(enc.Round),
		TurnIndex:  int64(enc.TurnIndex),
		PartyAp:    int64(enc.Resources.PartyAP),
		GmThreat:   int64(enc.Resources.GMThreat),
	}); err != nil {
		return fmt.Errorf("upsert encounter: %w", err)
	}

	if err = qtx.DeleteCombatantsByEncounterID(ctx, enc.ID); err != nil {
		return fmt.Errorf("clear combatants: %w", err)
	}

	for i, c := range enc.Combatants {
		domain.NormalizeCombatantHP(&c)
		enc.Combatants[i] = c
		if err = upsertCombatantNormalizedStats(ctx, qtx, ids, c.ID, c.Profile()); err != nil {
			return fmt.Errorf("sync normalized combatant stats %s: %w", c.ID, err)
		}
		if err = qtx.InsertCombatant(ctx, insertCombatantParams(enc.ID, i, c)); err != nil {
			return fmt.Errorf("insert combatant %s: %w", c.ID, err)
		}
	}

	if err = qtx.TouchCampaign(ctx, enc.CampaignID); err != nil {
		return fmt.Errorf("touch campaign: %w", err)
	}
	return nil
}

func normalizeEncounterForSave(enc *domain.Encounter) error {
	if strings.TrimSpace(enc.ID) == "" {
		return fmt.Errorf("encounter id is required")
	}
	enc.Name = strings.TrimSpace(enc.Name)
	if enc.Name == "" {
		return fmt.Errorf("encounter name is required")
	}
	if enc.Round < 1 {
		enc.Round = 1
	}
	if enc.TurnIndex < 0 {
		enc.TurnIndex = 0
	}
	if enc.Resources.PartyAP < 0 {
		enc.Resources.PartyAP = 0
	}
	if enc.Resources.GMThreat < 0 {
		enc.Resources.GMThreat = 0
	}
	for i := range enc.Combatants {
		enc.Combatants[i].Name = strings.TrimSpace(enc.Combatants[i].Name)
		if enc.Combatants[i].Side == "" {
			enc.Combatants[i].Side = domain.SideNPC
		}
		domain.NormalizeCombatantHP(&enc.Combatants[i])
		if err := domain.ValidateCombatant(enc.Combatants[i], domain.CombatantValidationOptions{
			Label:       fmt.Sprintf("combatant %q", enc.Combatants[i].Name),
			RequireName: true,
			RequireSide: true,
			MinLevel:    0,
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateUniquePlayerCharacters(combatants []domain.Combatant) error {
	seen := make(map[string]struct{})
	for _, c := range combatants {
		if c.Side != domain.SideParty {
			continue
		}
		playerCharacterID := strings.TrimSpace(c.PlayerCharacterID)
		if playerCharacterID == "" {
			continue
		}
		if _, ok := seen[playerCharacterID]; ok {
			return fmt.Errorf("player character %s is already in this encounter", playerCharacterID)
		}
		seen[playerCharacterID] = struct{}{}
	}
	return nil
}

func combatantIDsByID(ctx context.Context, qtx *dbgen.Queries, encounterID string) (map[string]struct{}, error) {
	rows, err := qtx.ListCombatantIDsByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(rows))
	for _, id := range rows {
		result[id] = struct{}{}
	}
	return result, nil
}

func activePartyCombatantsByID(ctx context.Context, qtx *dbgen.Queries, campaignID string) (map[string]domain.Combatant, error) {
	rows, err := qtx.ListActivePartyCharactersByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	profiles, err := playerCharacterResistanceProfiles(ctx, qtx, campaignID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]domain.Combatant, len(rows))
	for _, row := range rows {
		if row.AvailabilityStatus == playerCharacterAvailabilityInactive {
			continue
		}
		combatant := partyCombatantFromRow(row)
		applyPlayerCharacterResistanceProfile(&combatant, profiles)
		result[row.ID] = combatant
	}
	return result, nil
}

func (s *EncounterStore) List(ctx context.Context) ([]domain.EncounterSummary, error) {
	ctx = normalizeContext(ctx)
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.q.ListEncounterSummariesByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list encounters: %w", err)
	}

	summaries := make([]domain.EncounterSummary, 0, len(rows))
	for _, r := range rows {
		encounter, err := encounterByIDByCampaign(ctx, s.q, campaignID, r.ID)
		if err != nil {
			return nil, fmt.Errorf("load encounter summary %s: %w", r.ID, err)
		}
		summaries = append(summaries, encounterSummaryFromRow(r, domain.EvaluateEncounterDifficulty(encounter.Combatants)))
	}
	return summaries, nil
}

func (s *EncounterStore) GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}
	return s.getEncounterByIDByCampaign(ctx, campaignID, encounterID)
}

func (s *EncounterStore) getEncounterByIDByCampaign(ctx context.Context, campaignID, encounterID string) (*domain.Encounter, error) {
	return encounterByIDByCampaign(ctx, s.q, campaignID, encounterID)
}

func encounterByIDByCampaign(ctx context.Context, qtx *dbgen.Queries, campaignID, encounterID string) (*domain.Encounter, error) {
	row, err := qtx.GetEncounterByIDByCampaignID(ctx, dbgen.GetEncounterByIDByCampaignIDParams{
		CampaignID:  campaignID,
		EncounterID: encounterID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrEncounterNotFound
		}
		return nil, fmt.Errorf("read encounter by id: %w", err)
	}
	combatants, err := combatantsByEncounterID(ctx, qtx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("read combatants: %w", err)
	}
	return encounterFromByIDRow(row, combatants), nil
}

func (s *EncounterStore) UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	existing, err := s.GetEncounterByID(ctx, encounterID)
	if err != nil {
		return nil, err
	}

	activeCombatantID := ""
	if active := existing.ActiveCombatant(); active != nil {
		activeCombatantID = active.ID
	}

	updated := domain.NewEncounter(encounterID, name, combatants)
	updated.CampaignID = existing.CampaignID
	updated.Round = existing.Round
	if updated.Round < 1 {
		updated.Round = 1
	}
	updated.Resources = existing.Resources
	if len(updated.Combatants) > 0 {
		nextTurnIndex := existing.TurnIndex
		if nextTurnIndex < 0 {
			nextTurnIndex = 0
		}
		if nextTurnIndex >= len(updated.Combatants) {
			nextTurnIndex = len(updated.Combatants) - 1
		}
		if activeCombatantID != "" {
			for i := range updated.Combatants {
				if updated.Combatants[i].ID == activeCombatantID {
					nextTurnIndex = i
					break
				}
			}
		}
		updated.TurnIndex = nextTurnIndex
		for i := range updated.Combatants {
			updated.Combatants[i].Active = i == updated.TurnIndex
		}
	}
	if err := s.Save(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *EncounterStore) ListPartyMembers(ctx context.Context) ([]domain.Combatant, error) {
	ctx = normalizeContext(ctx)
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}

	activeCharacters, err := s.q.ListActivePartyCharactersByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign characters: %w", err)
	}
	profiles, err := playerCharacterResistanceProfiles(ctx, s.q, campaignID)
	if err != nil {
		return nil, err
	}
	party := make([]domain.Combatant, 0, len(activeCharacters))
	for _, row := range activeCharacters {
		if row.AvailabilityStatus == playerCharacterAvailabilityInactive {
			continue
		}
		combatant := partyCombatantFromRow(row)
		applyPlayerCharacterResistanceProfile(&combatant, profiles)
		party = append(party, combatant)
	}
	return party, nil
}

func (s *EncounterStore) Activate(ctx context.Context, encounterID string) error {
	ctx = normalizeContext(ctx)
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return err
	}
	affected, err := s.q.ActivateEncounterByCampaign(ctx, dbgen.ActivateEncounterByCampaignParams{
		EncounterID: encounterID,
		CampaignID:  campaignID,
	})
	if err != nil {
		return fmt.Errorf("activate encounter: %w", err)
	}
	if affected == 0 {
		return domain.ErrEncounterNotFound
	}
	return nil
}

func (s *EncounterStore) SoftDelete(ctx context.Context, encounterID string) error {
	ctx = normalizeContext(ctx)
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return err
	}
	affected, err := s.q.SoftDeleteEncounterByCampaign(ctx, dbgen.SoftDeleteEncounterByCampaignParams{
		EncounterID: encounterID,
		CampaignID:  campaignID,
	})
	if err != nil {
		return fmt.Errorf("soft delete encounter: %w", err)
	}
	if affected == 0 {
		return domain.ErrEncounterNotFound
	}
	return nil
}
