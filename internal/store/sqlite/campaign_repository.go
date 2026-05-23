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
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, r := range rows {
		player := campaignPlayerFromRow(r)
		applyPlayerCharacterResistanceProfile(&player.Character, profiles)
		players = append(players, player)
	}
	return players, nil
}

func (s *EncounterStore) removeInactiveCampaignCharactersFromEncounters(ctx context.Context, campaignID string) error {
	inactiveIDs, err := s.q.ListInactiveCurrentPlayerCharacterIDsByCampaignID(ctx, campaignID)
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

	encounterIDs, err := s.q.ListEncounterIDsByCampaignID(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("list campaign encounters: %w", err)
	}
	for _, encounterID := range encounterIDs {
		enc, err := s.getEncounterByIDByCampaign(ctx, campaignID, encounterID)
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
		if err := s.Save(ctx, updated); err != nil {
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)
	if err = qtx.EnsureAppStateRow(ctx); err != nil {
		return nil, fmt.Errorf("ensure app state: %w", err)
	}
	if err = qtx.InsertCampaign(ctx, dbgen.InsertCampaignParams{
		ID:        campaignID,
		Name:      name,
		StartDate: formatCampaignStartDateForDB(startDate),
	}); err != nil {
		return nil, fmt.Errorf("insert campaign: %w", err)
	}

	for _, p := range players {
		playerID := uuid.NewString()
		if err = qtx.InsertPlayer(ctx, dbgen.InsertPlayerParams{
			ID:         playerID,
			CampaignID: campaignID,
			Name:       strings.TrimSpace(p.PlayerName),
		}); err != nil {
			return nil, fmt.Errorf("insert player: %w", err)
		}

		charID := strings.TrimSpace(p.Character.ID)
		if charID == "" {
			charID = uuid.NewString()
		}
		if err = qtx.InsertPlayerCharacter(ctx, insertPlayerCharacterParams(charID, playerID, campaignID, p.Character, p.Inactive)); err != nil {
			return nil, fmt.Errorf("insert player character: %w", err)
		}
		if err = upsertPlayerCharacterNormalizedStats(ctx, qtx, charID, p.Character); err != nil {
			return nil, fmt.Errorf("sync normalized player character stats: %w", err)
		}
	}

	if _, activeErr := qtx.GetActiveCampaign(ctx); activeErr == sql.ErrNoRows {
		affected, setErr := qtx.SetActiveCampaign(ctx, campaignID)
		if setErr != nil {
			return nil, fmt.Errorf("set active campaign: %w", setErr)
		}
		if affected == 0 {
			return nil, domain.ErrCampaignNotFound
		}
	} else if activeErr != nil {
		return nil, fmt.Errorf("check active campaign: %w", activeErr)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	qtx := s.q.WithTx(tx)
	affected, err := qtx.UpdateCampaignByID(ctx, dbgen.UpdateCampaignByIDParams{
		Name:       name,
		StartDate:  formatCampaignStartDateForDB(startDate),
		CampaignID: campaignID,
	})
	if err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}
	if affected == 0 {
		return nil, domain.ErrCampaignNotFound
	}

	playerIDsByName, err := listCampaignPlayerIDsByName(ctx, qtx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign players: %w", err)
	}

	updatedPlayers := make(map[string]struct{}, len(players))
	for _, p := range players {
		playerName := strings.TrimSpace(p.PlayerName)
		playerKey := normalizeNameKey(playerName)
		if playerKey == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if _, exists := updatedPlayers[playerKey]; exists {
			return nil, fmt.Errorf("player %q already has an active character in this campaign", playerName)
		}
		updatedPlayers[playerKey] = struct{}{}

		playerID, exists := playerIDsByName[playerKey]
		if !exists {
			playerID = uuid.NewString()
			if err = qtx.InsertPlayer(ctx, dbgen.InsertPlayerParams{
				ID:         playerID,
				CampaignID: campaignID,
				Name:       playerName,
			}); err != nil {
				return nil, fmt.Errorf("insert player: %w", err)
			}
			playerIDsByName[playerKey] = playerID
		}

		activeChar, activeErr := getActivePlayerCharacterByPlayerID(ctx, qtx, playerID)
		if activeErr != nil {
			return nil, fmt.Errorf("load active player character: %w", activeErr)
		}

		targetCharacterName := strings.TrimSpace(p.Character.Name)
		charID := activeChar.ID
		shouldInsertNewCharacter := activeChar.ID == "" || normalizeNameKey(activeChar.Name) != normalizeNameKey(targetCharacterName)
		if shouldInsertNewCharacter {
			if err = deactivateActiveCharactersByPlayerID(ctx, qtx, playerID); err != nil {
				return nil, fmt.Errorf("deactivate player characters: %w", err)
			}
			charID = strings.TrimSpace(p.Character.ID)
			if charID == "" {
				charID = uuid.NewString()
			}
			if err = qtx.InsertPlayerCharacter(ctx, insertPlayerCharacterParams(charID, playerID, campaignID, p.Character, p.Inactive)); err != nil {
				return nil, fmt.Errorf("insert player character: %w", err)
			}
		} else {
			if err = updateActivePlayerCharacter(ctx, qtx, charID, campaignID, p.Character, p.Inactive); err != nil {
				return nil, fmt.Errorf("update player character: %w", err)
			}
		}

		if err = upsertPlayerCharacterNormalizedStats(ctx, qtx, charID, p.Character); err != nil {
			return nil, fmt.Errorf("sync normalized player character stats: %w", err)
		}
	}

	for playerKey, playerID := range playerIDsByName {
		if _, keep := updatedPlayers[playerKey]; keep {
			continue
		}
		if err = deactivateActiveCharactersByPlayerID(ctx, qtx, playerID); err != nil {
			return nil, fmt.Errorf("deactivate removed campaign players: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	if err = s.removeInactiveCampaignCharactersFromEncounters(ctx, campaignID); err != nil {
		return nil, err
	}
	return &domain.Campaign{
		ID:        campaignID,
		Name:      name,
		StartDate: startDate,
	}, nil
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

func updateActivePlayerCharacter(ctx context.Context, qtx *dbgen.Queries, characterID, campaignID string, c domain.Combatant, inactive bool) error {
	return qtx.UpdateActivePlayerCharacterByID(ctx, updateActivePlayerCharacterParams(characterID, campaignID, c, inactive))
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
