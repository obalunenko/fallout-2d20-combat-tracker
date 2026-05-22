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

type EncounterStore struct {
	db *sql.DB
	q  *dbgen.Queries
}

func NewEncounterStore(db *sql.DB) *EncounterStore {
	return &EncounterStore{
		db: db,
		q:  dbgen.New(db),
	}
}

func NewEncounterStoreWithContext(db *sql.DB, _ context.Context) *EncounterStore {
	return NewEncounterStore(db)
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

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

	combatantRows, err := s.q.ListCombatantsByEncounterID(ctx, encRow.ID)
	if err != nil {
		return nil, fmt.Errorf("read combatants: %w", err)
	}

	return encounterFromLatestRow(encRow, combatantsFromRows(combatantRows)), nil
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)
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
				if err = updateActivePlayerCharacter(ctx, qtx, playerCharacterID, enc.CampaignID, c, false); err != nil {
					return fmt.Errorf("update campaign character %s: %w", playerCharacterID, err)
				}
				if err = upsertPlayerCharacterNormalizedStats(ctx, qtx, playerCharacterID, c); err != nil {
					return fmt.Errorf("sync campaign character stats %s: %w", playerCharacterID, err)
				}
			}
		}
		enc.Combatants[i] = c
	}

	if err = validateUniquePlayerCharacters(enc.Combatants); err != nil {
		return err
	}

	metrics := domain.EvaluateEncounterDifficulty(enc.Combatants)
	if err = qtx.UpsertEncounter(ctx, dbgen.UpsertEncounterParams{
		ID:              enc.ID,
		CampaignID:      enc.CampaignID,
		Name:            enc.Name,
		Round:           int64(enc.Round),
		TurnIndex:       int64(enc.TurnIndex),
		PartyAp:         int64(enc.Resources.PartyAP),
		GmThreat:        int64(enc.Resources.GMThreat),
		DifficultyLabel: string(metrics.Label),
		DifficultyScore: metrics.Score,
		PartyCount:      int64(metrics.PartyCount),
		PartyAvgLevel:   metrics.PartyAvgLevel,
		PartyXpBudget:   int64(metrics.PartyXPBudget),
		EnemyCount:      int64(metrics.EnemyCount),
		EnemyAvgLevel:   metrics.EnemyAvgLevel,
		EnemyTotalXp:    int64(metrics.EnemyTotalXP),
	}); err != nil {
		return fmt.Errorf("upsert encounter: %w", err)
	}

	if err = qtx.DeleteCombatantsByEncounterID(ctx, enc.ID); err != nil {
		return fmt.Errorf("clear combatants: %w", err)
	}

	for i, c := range enc.Combatants {
		domain.NormalizeCombatantHP(&c)
		enc.Combatants[i] = c
		if err = qtx.InsertCombatant(ctx, insertCombatantParams(enc.ID, i, c)); err != nil {
			return fmt.Errorf("insert combatant %s: %w", c.ID, err)
		}
		if err = upsertCombatantNormalizedStats(ctx, qtx, c.ID, c); err != nil {
			return fmt.Errorf("sync normalized combatant stats %s: %w", c.ID, err)
		}
	}

	if err = qtx.TouchCampaign(ctx, enc.CampaignID); err != nil {
		return fmt.Errorf("touch campaign: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
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
	result := make(map[string]domain.Combatant, len(rows))
	for _, row := range rows {
		if row.AvailabilityStatus == playerCharacterAvailabilityInactive {
			continue
		}
		result[row.ID] = partyCombatantFromRow(row)
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
		summaries = append(summaries, encounterSummaryFromRow(r))
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
	row, err := s.q.GetEncounterByIDByCampaignID(ctx, dbgen.GetEncounterByIDByCampaignIDParams{
		CampaignID:  campaignID,
		EncounterID: encounterID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrEncounterNotFound
		}
		return nil, fmt.Errorf("read encounter by id: %w", err)
	}
	combatantRows, err := s.q.ListCombatantsByEncounterID(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("read combatants: %w", err)
	}
	return encounterFromByIDRow(row, combatantsFromRows(combatantRows)), nil
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
	party := make([]domain.Combatant, 0, len(activeCharacters))
	for _, row := range activeCharacters {
		if row.AvailabilityStatus == playerCharacterAvailabilityInactive {
			continue
		}
		party = append(party, partyCombatantFromRow(row))
	}
	return party, nil
}

func (s *EncounterStore) ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error) {
	ctx = normalizeContext(ctx)
	rows, err := s.q.ListMonsterTemplates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monster templates: %w", err)
	}
	monsters := make([]domain.Combatant, 0, len(rows))
	for _, row := range rows {
		monsters = append(monsters, monsterTemplateFromRow(row))
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Combatant{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	qtx := s.q.WithTx(tx)
	if err = qtx.UpsertMonsterTemplate(ctx, upsertMonsterTemplateParams(monster)); err != nil {
		return domain.Combatant{}, fmt.Errorf("upsert monster template: %w", err)
	}
	templateID, err := qtx.GetMonsterTemplateIDByNameKey(ctx, normalizeNameKey(monster.Name))
	if err != nil {
		return domain.Combatant{}, fmt.Errorf("get monster template id: %w", err)
	}
	monster.ID = templateID
	if err = upsertMonsterTemplateNormalizedStats(ctx, qtx, templateID, monster); err != nil {
		return domain.Combatant{}, fmt.Errorf("sync normalized monster template stats: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return domain.Combatant{}, fmt.Errorf("commit tx: %w", err)
	}
	return monster, nil
}

func (s *EncounterStore) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	ctx = normalizeContext(ctx)
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	rows, err := s.q.ListActivePartyCharactersByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign players: %w", err)
	}
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, r := range rows {
		players = append(players, campaignPlayerFromRow(r))
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

func (s *EncounterStore) AppendEncounterLog(ctx context.Context, encounterID string, round int, message string) error {
	ctx = normalizeContext(ctx)
	if encounterID == "" {
		return fmt.Errorf("encounter id is required")
	}
	if message == "" {
		return fmt.Errorf("log message is required")
	}

	if err := s.q.InsertEncounterLog(ctx, dbgen.InsertEncounterLogParams{
		ID:          uuid.NewString(),
		EncounterID: encounterID,
		Round:       int64(round),
		Message:     message,
	}); err != nil {
		return fmt.Errorf("insert encounter log: %w", err)
	}
	return nil
}

func (s *EncounterStore) ListEncounterLogs(ctx context.Context, encounterID string) ([]domain.EncounterLog, error) {
	ctx = normalizeContext(ctx)
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}

	rows, err := s.q.ListEncounterLogsByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list encounter logs: %w", err)
	}

	logs := make([]domain.EncounterLog, 0, len(rows))
	for _, r := range rows {
		logs = append(logs, encounterLogFromRow(r))
	}
	return logs, nil
}

func (s *EncounterStore) activeCampaignID(ctx context.Context) (string, error) {
	if err := s.q.EnsureAppStateRow(ctx); err != nil {
		return "", fmt.Errorf("ensure app state: %w", err)
	}
	row, err := s.q.GetActiveCampaign(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", domain.ErrCampaignNotInitialized
		}
		return "", fmt.Errorf("get active campaign: %w", err)
	}
	return row.ID, nil
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func interfaceToString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case sql.NullString:
		if typed.Valid {
			return typed.String
		}
		return ""
	default:
		return ""
	}
}
