package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

func (s *EncounterStore) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	rows, err := s.q.ListActivePartyCharactersByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign players: %w", err)
	}
	profiles, err := playerCharacterResistanceProfiles(ctx, s.q, campaignID)
	if err != nil {
		return nil, err
	}
	specialValues, err := playerCharacterSpecialValuesByCampaign(ctx, s.q, campaignID)
	if err != nil {
		return nil, err
	}
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, r := range rows {
		player := campaignPlayerFromRow(r)
		applyPlayerCharacterResistanceProfile(&player.Character, profiles)
		values, ok := specialValues[player.Character.ID]
		if !ok {
			return nil, fmt.Errorf("player character %s has no SPECIAL values", player.Character.ID)
		}
		player.Special = values
		players = append(players, player)
	}
	return players, nil
}

func removeInactiveCampaignCharactersFromEncounters(ctx context.Context, qtx *dbgen.Queries, campaignID string) error {
	inactiveIDs, err := qtx.ListInactiveCurrentPlayerCharacterIDsByCampaignID(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("list inactive campaign characters: %w", err)
	}
	if len(inactiveIDs) == 0 {
		return nil
	}

	inactiveByID := make(map[string]struct{}, len(inactiveIDs))
	for _, id := range inactiveIDs {
		inactiveByID[id] = struct{}{}
	}

	encounterIDs, err := qtx.ListEncounterIDsByCampaignID(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("list campaign encounters: %w", err)
	}
	for _, encounterID := range encounterIDs {
		enc, err := encounterByIDByCampaign(ctx, qtx, campaignID, encounterID)
		if err != nil {
			return err
		}

		activeCombatantID := ""
		if active := enc.ActiveCombatant(); active != nil {
			activeCombatantID = active.ID
		}
		filtered := make([]domain.Combatant, 0, len(enc.Combatants))
		removed := false
		for _, c := range enc.Combatants {
			playerCharacterID := strings.TrimSpace(c.PlayerCharacterID)
			if c.Side == domain.SideParty && playerCharacterID != "" {
				if _, inactive := inactiveByID[playerCharacterID]; inactive {
					removed = true
					continue
				}
			}
			filtered = append(filtered, c)
		}
		if !removed {
			continue
		}

		updated := domain.NewEncounter(enc.ID, enc.Name, filtered)
		updated.CampaignID = enc.CampaignID
		updated.Round = enc.Round
		if updated.Round < 1 {
			updated.Round = 1
		}
		updated.Resources = enc.Resources
		if len(updated.Combatants) > 0 {
			updated.TurnIndex = min(enc.TurnIndex, len(updated.Combatants)-1)
			if updated.TurnIndex < 0 {
				updated.TurnIndex = 0
			}
			for i := range updated.Combatants {
				if updated.Combatants[i].ID == activeCombatantID {
					updated.TurnIndex = i
					break
				}
			}
			for i := range updated.Combatants {
				updated.Combatants[i].Active = i == updated.TurnIndex
			}
		}
		if err := saveEncounter(ctx, qtx, updated); err != nil {
			return fmt.Errorf("remove inactive party characters from encounter %s: %w", encounterID, err)
		}
	}
	return nil
}

func (s *EncounterStore) CreateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if startDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}

	if err := s.runInTx(ctx, func(qtx *dbgen.Queries) error {
		if err := qtx.EnsureAppStateRow(ctx); err != nil {
			return fmt.Errorf("ensure app state: %w", err)
		}
		if err := qtx.InsertCampaign(ctx, dbgen.InsertCampaignParams{
			ID:        campaignID,
			Name:      name,
			StartDate: formatCampaignStartDateForDB(startDate),
		}); err != nil {
			return fmt.Errorf("insert campaign: %w", err)
		}
		ids, err := normalizedDictionaryIDs(ctx, qtx)
		if err != nil {
			return err
		}
		specialIDs, err := specialAttributeIDs(ctx, qtx)
		if err != nil {
			return err
		}

		for _, p := range players {
			if err := normalizeCampaignPlayerForSave(&p); err != nil {
				return err
			}
			playerID := uuid.NewString()
			if err := qtx.InsertPlayer(ctx, dbgen.InsertPlayerParams{
				ID:         playerID,
				CampaignID: campaignID,
				Name:       strings.TrimSpace(p.PlayerName),
			}); err != nil {
				return fmt.Errorf("insert player: %w", err)
			}

			charID := strings.TrimSpace(p.Character.ID)
			if charID == "" {
				charID = uuid.NewString()
			}
			if err := upsertPlayerCharacterNormalizedStats(ctx, qtx, ids, charID, p.Character.Profile()); err != nil {
				return fmt.Errorf("sync normalized player character stats: %w", err)
			}
			if err := qtx.InsertPlayerCharacter(ctx, insertPlayerCharacterParams(charID, playerID, p)); err != nil {
				return fmt.Errorf("insert player character: %w", err)
			}
			if err := upsertPlayerCharacterSpecialValues(ctx, qtx, specialIDs, charID, p.Special); err != nil {
				return fmt.Errorf("sync player character SPECIAL values: %w", err)
			}
		}

		if _, activeErr := qtx.GetActiveCampaign(ctx); activeErr == sql.ErrNoRows {
			affected, err := qtx.SetActiveCampaign(ctx, campaignID)
			if err != nil {
				return fmt.Errorf("set active campaign: %w", err)
			}
			if affected == 0 {
				return domain.ErrCampaignNotFound
			}
		} else if activeErr != nil {
			return fmt.Errorf("check active campaign: %w", activeErr)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &domain.Campaign{
		ID:        campaignID,
		Name:      name,
		StartDate: startDate,
	}, nil
}

func (s *EncounterStore) UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if startDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if err := s.runInTx(ctx, func(qtx *dbgen.Queries) error {
		affected, err := qtx.UpdateCampaignByID(ctx, dbgen.UpdateCampaignByIDParams{
			Name:       name,
			StartDate:  formatCampaignStartDateForDB(startDate),
			CampaignID: campaignID,
		})
		if err != nil {
			return fmt.Errorf("update campaign: %w", err)
		}
		if affected == 0 {
			return domain.ErrCampaignNotFound
		}

		playerIDsByName, err := listCampaignPlayerIDsByName(ctx, qtx, campaignID)
		if err != nil {
			return fmt.Errorf("list campaign players: %w", err)
		}
		ids, err := normalizedDictionaryIDs(ctx, qtx)
		if err != nil {
			return err
		}
		specialIDs, err := specialAttributeIDs(ctx, qtx)
		if err != nil {
			return err
		}

		updatedPlayers := make(map[string]struct{}, len(players))
		savedPlayersByCharacterID := make(map[string]domain.NewCampaignPlayer, len(players))
		for _, p := range players {
			if err := normalizeCampaignPlayerForSave(&p); err != nil {
				return err
			}
			playerName := strings.TrimSpace(p.PlayerName)
			playerKey := normalizeNameKey(playerName)
			if playerKey == "" {
				return fmt.Errorf("player name is required")
			}
			if _, exists := updatedPlayers[playerKey]; exists {
				return fmt.Errorf("player %q already has an active character in this campaign", playerName)
			}
			updatedPlayers[playerKey] = struct{}{}

			playerID, exists := playerIDsByName[playerKey]
			if !exists {
				playerID = uuid.NewString()
				if err := qtx.InsertPlayer(ctx, dbgen.InsertPlayerParams{
					ID:         playerID,
					CampaignID: campaignID,
					Name:       playerName,
				}); err != nil {
					return fmt.Errorf("insert player: %w", err)
				}
				playerIDsByName[playerKey] = playerID
			}

			activeChar, err := getActivePlayerCharacterByPlayerID(ctx, qtx, playerID)
			if err != nil {
				return fmt.Errorf("load active player character: %w", err)
			}

			targetCharacterName := strings.TrimSpace(p.Character.Name)
			charID := activeChar.ID
			shouldInsertNewCharacter := activeChar.ID == "" || normalizeNameKey(activeChar.Name) != normalizeNameKey(targetCharacterName)
			if shouldInsertNewCharacter {
				if err := deactivateActiveCharactersByPlayerID(ctx, qtx, playerID); err != nil {
					return fmt.Errorf("deactivate player characters: %w", err)
				}
				charID = strings.TrimSpace(p.Character.ID)
				if charID == "" {
					charID = uuid.NewString()
				}
				if err := upsertPlayerCharacterNormalizedStats(ctx, qtx, ids, charID, p.Character.Profile()); err != nil {
					return fmt.Errorf("sync normalized player character stats: %w", err)
				}
				if err := qtx.InsertPlayerCharacter(ctx, insertPlayerCharacterParams(charID, playerID, p)); err != nil {
					return fmt.Errorf("insert player character: %w", err)
				}
			} else {
				if err := updateActivePlayerCharacter(ctx, qtx, charID, p); err != nil {
					return fmt.Errorf("update player character: %w", err)
				}
				if err := upsertPlayerCharacterNormalizedStats(ctx, qtx, ids, charID, p.Character.Profile()); err != nil {
					return fmt.Errorf("sync normalized player character stats: %w", err)
				}
			}
			if err := upsertPlayerCharacterSpecialValues(ctx, qtx, specialIDs, charID, p.Special); err != nil {
				return fmt.Errorf("sync player character SPECIAL values: %w", err)
			}
			p.Character.ID = charID
			if !p.Inactive {
				savedPlayersByCharacterID[charID] = p
			}
		}

		for playerKey, playerID := range playerIDsByName {
			if _, keep := updatedPlayers[playerKey]; keep {
				continue
			}
			if err := deactivateActiveCharactersByPlayerID(ctx, qtx, playerID); err != nil {
				return fmt.Errorf("deactivate removed campaign players: %w", err)
			}
		}
		if err := syncEffectiveActiveEncounterPlayerCharacters(ctx, qtx, ids, campaignID, savedPlayersByCharacterID); err != nil {
			return err
		}
		if err := removeInactiveCampaignCharactersFromEncounters(ctx, qtx, campaignID); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &domain.Campaign{
		ID:        campaignID,
		Name:      name,
		StartDate: startDate,
	}, nil
}

func syncEffectiveActiveEncounterPlayerCharacters(
	ctx context.Context,
	qtx *dbgen.Queries,
	ids dictionaryIDs,
	campaignID string,
	playersByCharacterID map[string]domain.NewCampaignPlayer,
) error {
	encounterID, err := qtx.GetEffectiveActiveEncounterIDByCampaignID(ctx, campaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("get effective active encounter: %w", err)
	}
	rows, err := qtx.ListLinkedCombatantsByEncounterID(ctx, encounterID)
	if err != nil {
		return fmt.Errorf("list active encounter linked combatants: %w", err)
	}
	for _, row := range rows {
		player, ok := playersByCharacterID[row.PlayerCharacterID]
		if !ok {
			continue
		}
		character := player.Character
		if err := qtx.UpdateLinkedCombatantSnapshotStatsByID(ctx, dbgen.UpdateLinkedCombatantSnapshotStatsByIDParams{
			Level:       int64(character.Level),
			Hp:          int64(character.HP),
			MaxHp:       int64(character.MaxHP),
			Defense:     int64(character.Defense),
			CombatantID: row.ID,
		}); err != nil {
			return fmt.Errorf("sync linked combatant stats %s: %w", row.ID, err)
		}
		if err := replaceStatProfileNormalizedResistances(
			ctx,
			qtx,
			ids,
			statProfileID(statProfileCombatantKind, row.ID),
			character.ResistanceProfile(),
		); err != nil {
			return fmt.Errorf("sync linked combatant resistance %s: %w", row.ID, err)
		}
		if err := qtx.UpdateCombatantDefeatedByID(ctx, dbgen.UpdateCombatantDefeatedByIDParams{
			Defeated:    boolToInt64(character.HP == 0),
			CombatantID: row.ID,
		}); err != nil {
			return fmt.Errorf("sync linked combatant defeated state %s: %w", row.ID, err)
		}
	}
	return nil
}

type playerCharacterRef struct {
	ID   string
	Name string
}

func listCampaignPlayerIDsByName(ctx context.Context, qtx *dbgen.Queries, campaignID string) (map[string]string, error) {
	rows, err := qtx.ListPlayerIDsAndNamesByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, row := range rows {
		result[normalizeNameKey(row.Name)] = row.ID
	}
	return result, nil
}

func getActivePlayerCharacterByPlayerID(ctx context.Context, qtx *dbgen.Queries, playerID string) (playerCharacterRef, error) {
	row, err := qtx.GetActivePlayerCharacterByPlayerID(ctx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return playerCharacterRef{}, nil
		}
		return playerCharacterRef{}, err
	}
	return playerCharacterRef{ID: row.ID, Name: row.Name}, nil
}

func deactivateActiveCharactersByPlayerID(ctx context.Context, qtx *dbgen.Queries, playerID string) error {
	return qtx.DeactivateActiveCharactersByPlayerID(ctx, playerID)
}

func normalizeCampaignPlayerForSave(player *domain.NewCampaignPlayer) error {
	player.PlayerName = strings.TrimSpace(player.PlayerName)
	if player.PlayerName == "" {
		return fmt.Errorf("player name is required")
	}
	player.Character.Name = strings.TrimSpace(player.Character.Name)
	if player.Character.Name == "" {
		return fmt.Errorf("character name is required for player %q", player.PlayerName)
	}
	if player.Character.ID == "" {
		player.Character.ID = uuid.NewString()
	}
	player.Character.Side = domain.SideParty
	player.Character.XP = 0
	if err := domain.ValidateCombatant(player.Character, domain.CombatantValidationOptions{
		Label:       fmt.Sprintf("player %q", player.PlayerName),
		RequireName: true,
		MinLevel:    1,
	}); err != nil {
		return err
	}
	domain.NormalizeCombatantHP(&player.Character)
	if player.Special.IsZero() {
		player.Special = domain.DefaultSpecialValues()
	}
	if err := player.Special.Validate(); err != nil {
		return fmt.Errorf("player %q: %w", player.PlayerName, err)
	}
	return nil
}

func updateActivePlayerCharacter(ctx context.Context, qtx *dbgen.Queries, characterID string, player domain.NewCampaignPlayer) error {
	return qtx.UpdateActivePlayerCharacterByID(ctx, updateActivePlayerCharacterParams(characterID, player))
}

func normalizeNameKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *EncounterStore) GetActiveCampaign(ctx context.Context) (*domain.Campaign, error) {
	ctx = normalizeContext(ctx)
	if err := s.q.EnsureAppStateRow(ctx); err != nil {
		return nil, fmt.Errorf("ensure app state: %w", err)
	}
	row, err := s.q.GetActiveCampaign(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrCampaignNotInitialized
		}
		return nil, fmt.Errorf("get active campaign: %w", err)
	}
	campaign := campaignFromRow(row)
	return &campaign, nil
}

func (s *EncounterStore) ListCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	ctx = normalizeContext(ctx)
	rows, err := s.q.ListCampaigns(ctx)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	result := make([]domain.Campaign, 0, len(rows))
	for _, row := range rows {
		result = append(result, campaignFromListRow(row))
	}
	return result, nil
}

func (s *EncounterStore) ActivateCampaign(ctx context.Context, campaignID string) error {
	ctx = normalizeContext(ctx)
	if err := s.q.EnsureAppStateRow(ctx); err != nil {
		return fmt.Errorf("ensure app state: %w", err)
	}
	affected, err := s.q.SetActiveCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("activate campaign: %w", err)
	}
	if affected == 0 {
		return domain.ErrCampaignNotFound
	}
	return nil
}

func (s *EncounterStore) UpdateCampaignResources(ctx context.Context, campaignID string, resources domain.Resources) error {
	ctx = normalizeContext(ctx)
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	resources.Normalize()
	affected, err := s.q.UpdateCampaignResourcesByID(ctx, dbgen.UpdateCampaignResourcesByIDParams{
		PartyAp:    int64(resources.PartyAP),
		GmThreat:   int64(resources.GMThreat),
		CampaignID: campaignID,
	})
	if err != nil {
		return fmt.Errorf("update campaign resources: %w", err)
	}
	if affected == 0 {
		return domain.ErrCampaignNotFound
	}
	return nil
}
