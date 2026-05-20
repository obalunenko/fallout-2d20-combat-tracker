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

type EncounterStore struct {
	db  *sql.DB
	q   *dbgen.Queries
	ctx context.Context
}

func NewEncounterStore(db *sql.DB) *EncounterStore {
	return NewEncounterStoreWithContext(db, context.Background())
}

func NewEncounterStoreWithContext(db *sql.DB, ctx context.Context) *EncounterStore {
	if ctx == nil {
		ctx = context.Background()
	}
	return &EncounterStore{
		db:  db,
		q:   dbgen.New(db),
		ctx: ctx,
	}
}

func (s *EncounterStore) Get() (*domain.Encounter, error) {
	ctx := s.ctx
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

	combatants := make([]domain.Combatant, 0, len(combatantRows))
	for _, c := range combatantRows {
		combatants = append(combatants, domain.Combatant{
			ID:                      c.ID,
			Name:                    c.Name,
			Side:                    domain.Side(c.Side),
			TorsoOnly:               c.TorsoOnly == 1,
			Level:                   int(c.Level),
			XP:                      int(c.Xp),
			Initiative:              int(c.Initiative),
			HP:                      int(c.Hp),
			MaxHP:                   int(c.MaxHp),
			Defense:                 int(c.Defense),
			DefenseHead:             int(c.DefenseHead),
			DefenseTorso:            int(c.DefenseTorso),
			DefenseLeftArm:          int(c.DefenseLeftArm),
			DefenseRightArm:         int(c.DefenseRightArm),
			DefenseLeftLeg:          int(c.DefenseLeftLeg),
			DefenseRightLeg:         int(c.DefenseRightLeg),
			ResistPhysicalHead:      int(c.DamageResistancePhysicalHead),
			ResistPhysicalTorso:     int(c.DamageResistancePhysicalTorso),
			ResistPhysicalLeftArm:   int(c.DamageResistancePhysicalLeftArm),
			ResistPhysicalRightArm:  int(c.DamageResistancePhysicalRightArm),
			ResistPhysicalLeftLeg:   int(c.DamageResistancePhysicalLeftLeg),
			ResistPhysicalRightLeg:  int(c.DamageResistancePhysicalRightLeg),
			ResistEnergyHead:        int(c.DamageResistanceEnergyHead),
			ResistEnergyTorso:       int(c.DamageResistanceEnergyTorso),
			ResistEnergyLeftArm:     int(c.DamageResistanceEnergyLeftArm),
			ResistEnergyRightArm:    int(c.DamageResistanceEnergyRightArm),
			ResistEnergyLeftLeg:     int(c.DamageResistanceEnergyLeftLeg),
			ResistEnergyRightLeg:    int(c.DamageResistanceEnergyRightLeg),
			ResistRadiationHead:     int(c.DamageResistanceRadiationHead),
			ResistRadiationTorso:    int(c.DamageResistanceRadiationTorso),
			ResistRadiationLeftArm:  int(c.DamageResistanceRadiationLeftArm),
			ResistRadiationRightArm: int(c.DamageResistanceRadiationRightArm),
			ResistRadiationLeftLeg:  int(c.DamageResistanceRadiationLeftLeg),
			ResistRadiationRightLeg: int(c.DamageResistanceRadiationRightLeg),
			ResistPhysical:          int(c.DamageResistancePhysical),
			ResistEnergy:            int(c.DamageResistanceEnergy),
			ResistRadiation:         int(c.DamageResistanceRadiation),
			ResistPoison:            int(c.DamageResistancePoison),
			ImmunePhysical:          c.DamageResistancePhysicalImmune == 1,
			ImmuneEnergy:            c.DamageResistanceEnergyImmune == 1,
			ImmuneRadiation:         c.DamageResistanceRadiationImmune == 1,
			ImmunePoison:            c.DamageResistancePoisonImmune == 1,
			Active:                  c.Active == 1,
			Defeated:                c.Defeated == 1,
		})
	}

	return &domain.Encounter{
		ID:         encRow.ID,
		CampaignID: interfaceToString(encRow.CampaignID),
		Name:       encRow.Name,
		Round:      int(encRow.Round),
		TurnIndex:  int(encRow.TurnIndex),
		Combatants: combatants,
		Resources: domain.Resources{
			PartyAP:  int(encRow.PartyAp),
			GMThreat: int(encRow.GmThreat),
		},
	}, nil
}

func (s *EncounterStore) Save(enc *domain.Encounter) error {
	if enc == nil {
		return fmt.Errorf("encounter cannot be nil")
	}

	ctx := s.ctx
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
		if err = qtx.InsertCombatant(ctx, dbgen.InsertCombatantParams{
			ID:          c.ID,
			EncounterID: enc.ID,
			Name:        c.Name,
			Side:        string(c.Side),
			TorsoOnly:   boolToInt64(c.TorsoOnly),
			Level:       int64(c.Level),
			Xp:          int64(c.XP),
			Initiative:  int64(c.Initiative),
			Hp:          int64(c.HP),
			MaxHp:       int64(c.MaxHP),
			Defense:     int64(c.Defense),
			Active:      boolToInt64(c.Active),
			Defeated:    boolToInt64(c.Defeated),
			Position:    int64(i),
		}); err != nil {
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

func (s *EncounterStore) List() ([]domain.EncounterSummary, error) {
	ctx := s.ctx
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
		summaries = append(summaries, domain.EncounterSummary{
			ID:              r.ID,
			CampaignID:      interfaceToString(r.CampaignID),
			Name:            r.Name,
			Round:           int(r.Round),
			Combatants:      int(r.Combatants),
			Difficulty:      r.DifficultyLabel,
			DifficultyScore: r.DifficultyScore,
			PartyCount:      int(r.PartyCount),
			PartyAvgLevel:   r.PartyAvgLevel,
			PartyXPBudget:   int(r.PartyXpBudget),
			EnemyCount:      int(r.EnemyCount),
			EnemyAvgLevel:   r.EnemyAvgLevel,
			EnemyTotalXP:    int(r.EnemyTotalXp),
			UpdatedAt:       r.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	return summaries, nil
}

func (s *EncounterStore) GetEncounterByID(encounterID string) (*domain.Encounter, error) {
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	ctx := s.ctx
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}
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
	combatants := make([]domain.Combatant, 0, len(combatantRows))
	for _, c := range combatantRows {
		combatants = append(combatants, domain.Combatant{
			ID:                      c.ID,
			Name:                    c.Name,
			Side:                    domain.Side(c.Side),
			TorsoOnly:               c.TorsoOnly == 1,
			Level:                   int(c.Level),
			XP:                      int(c.Xp),
			Initiative:              int(c.Initiative),
			HP:                      int(c.Hp),
			MaxHP:                   int(c.MaxHp),
			Defense:                 int(c.Defense),
			DefenseHead:             int(c.DefenseHead),
			DefenseTorso:            int(c.DefenseTorso),
			DefenseLeftArm:          int(c.DefenseLeftArm),
			DefenseRightArm:         int(c.DefenseRightArm),
			DefenseLeftLeg:          int(c.DefenseLeftLeg),
			DefenseRightLeg:         int(c.DefenseRightLeg),
			ResistPhysicalHead:      int(c.DamageResistancePhysicalHead),
			ResistPhysicalTorso:     int(c.DamageResistancePhysicalTorso),
			ResistPhysicalLeftArm:   int(c.DamageResistancePhysicalLeftArm),
			ResistPhysicalRightArm:  int(c.DamageResistancePhysicalRightArm),
			ResistPhysicalLeftLeg:   int(c.DamageResistancePhysicalLeftLeg),
			ResistPhysicalRightLeg:  int(c.DamageResistancePhysicalRightLeg),
			ResistEnergyHead:        int(c.DamageResistanceEnergyHead),
			ResistEnergyTorso:       int(c.DamageResistanceEnergyTorso),
			ResistEnergyLeftArm:     int(c.DamageResistanceEnergyLeftArm),
			ResistEnergyRightArm:    int(c.DamageResistanceEnergyRightArm),
			ResistEnergyLeftLeg:     int(c.DamageResistanceEnergyLeftLeg),
			ResistEnergyRightLeg:    int(c.DamageResistanceEnergyRightLeg),
			ResistRadiationHead:     int(c.DamageResistanceRadiationHead),
			ResistRadiationTorso:    int(c.DamageResistanceRadiationTorso),
			ResistRadiationLeftArm:  int(c.DamageResistanceRadiationLeftArm),
			ResistRadiationRightArm: int(c.DamageResistanceRadiationRightArm),
			ResistRadiationLeftLeg:  int(c.DamageResistanceRadiationLeftLeg),
			ResistRadiationRightLeg: int(c.DamageResistanceRadiationRightLeg),
			ResistPhysical:          int(c.DamageResistancePhysical),
			ResistEnergy:            int(c.DamageResistanceEnergy),
			ResistRadiation:         int(c.DamageResistanceRadiation),
			ResistPoison:            int(c.DamageResistancePoison),
			ImmunePhysical:          c.DamageResistancePhysicalImmune == 1,
			ImmuneEnergy:            c.DamageResistanceEnergyImmune == 1,
			ImmuneRadiation:         c.DamageResistanceRadiationImmune == 1,
			ImmunePoison:            c.DamageResistancePoisonImmune == 1,
			Active:                  c.Active == 1,
			Defeated:                c.Defeated == 1,
		})
	}
	return &domain.Encounter{
		ID:         row.ID,
		CampaignID: interfaceToString(row.CampaignID),
		Name:       row.Name,
		Round:      int(row.Round),
		TurnIndex:  int(row.TurnIndex),
		Combatants: combatants,
		Resources: domain.Resources{
			PartyAP:  int(row.PartyAp),
			GMThreat: int(row.GmThreat),
		},
	}, nil
}

func (s *EncounterStore) UpdateEncounter(encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	existing, err := s.GetEncounterByID(encounterID)
	if err != nil {
		return nil, err
	}
	updated := domain.NewEncounter(encounterID, name, combatants)
	updated.CampaignID = existing.CampaignID
	updated.Round = existing.Round
	if updated.Round < 1 {
		updated.Round = 1
	}
	updated.Resources = existing.Resources
	if err := s.Save(updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *EncounterStore) ListPartyMembers() ([]domain.Combatant, error) {
	ctx := s.ctx
	campaignID, err := s.activeCampaignID(ctx)
	if err != nil {
		return nil, err
	}

	activeCharacters, err := s.q.ListActivePartyCharactersByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign characters: %w", err)
	}
	if len(activeCharacters) > 0 {
		party := make([]domain.Combatant, 0, len(activeCharacters))
		for _, r := range activeCharacters {
			party = append(party, domain.Combatant{
				ID:                      r.ID,
				Name:                    r.CharacterName,
				Side:                    domain.SideParty,
				TorsoOnly:               r.TorsoOnly == 1,
				Level:                   int(r.Level),
				XP:                      0,
				Initiative:              int(r.Initiative),
				HP:                      int(r.Hp),
				MaxHP:                   int(r.MaxHp),
				Defense:                 int(r.Defense),
				DefenseHead:             int(r.DefenseHead),
				DefenseTorso:            int(r.DefenseTorso),
				DefenseLeftArm:          int(r.DefenseLeftArm),
				DefenseRightArm:         int(r.DefenseRightArm),
				DefenseLeftLeg:          int(r.DefenseLeftLeg),
				DefenseRightLeg:         int(r.DefenseRightLeg),
				ResistPhysicalHead:      int(r.DamageResistancePhysicalHead),
				ResistPhysicalTorso:     int(r.DamageResistancePhysicalTorso),
				ResistPhysicalLeftArm:   int(r.DamageResistancePhysicalLeftArm),
				ResistPhysicalRightArm:  int(r.DamageResistancePhysicalRightArm),
				ResistPhysicalLeftLeg:   int(r.DamageResistancePhysicalLeftLeg),
				ResistPhysicalRightLeg:  int(r.DamageResistancePhysicalRightLeg),
				ResistEnergyHead:        int(r.DamageResistanceEnergyHead),
				ResistEnergyTorso:       int(r.DamageResistanceEnergyTorso),
				ResistEnergyLeftArm:     int(r.DamageResistanceEnergyLeftArm),
				ResistEnergyRightArm:    int(r.DamageResistanceEnergyRightArm),
				ResistEnergyLeftLeg:     int(r.DamageResistanceEnergyLeftLeg),
				ResistEnergyRightLeg:    int(r.DamageResistanceEnergyRightLeg),
				ResistRadiationHead:     int(r.DamageResistanceRadiationHead),
				ResistRadiationTorso:    int(r.DamageResistanceRadiationTorso),
				ResistRadiationLeftArm:  int(r.DamageResistanceRadiationLeftArm),
				ResistRadiationRightArm: int(r.DamageResistanceRadiationRightArm),
				ResistRadiationLeftLeg:  int(r.DamageResistanceRadiationLeftLeg),
				ResistRadiationRightLeg: int(r.DamageResistanceRadiationRightLeg),
				ResistPhysical:          int(r.DamageResistancePhysical),
				ResistEnergy:            int(r.DamageResistanceEnergy),
				ResistRadiation:         int(r.DamageResistanceRadiation),
				ResistPoison:            int(r.DamageResistancePoison),
				ImmunePhysical:          r.DamageResistancePhysicalImmune == 1,
				ImmuneEnergy:            r.DamageResistanceEnergyImmune == 1,
				ImmuneRadiation:         r.DamageResistanceRadiationImmune == 1,
				ImmunePoison:            r.DamageResistancePoisonImmune == 1,
			})
		}
		return party, nil
	}
	return []domain.Combatant{}, nil
}

func (s *EncounterStore) ListCampaignPlayers(campaignID string) ([]domain.NewCampaignPlayer, error) {
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	rows, err := s.q.ListActivePartyCharactersByCampaignID(s.ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list campaign players: %w", err)
	}
	players := make([]domain.NewCampaignPlayer, 0, len(rows))
	for _, r := range rows {
		players = append(players, domain.NewCampaignPlayer{
			PlayerName: r.PlayerName,
			Character: domain.Combatant{
				ID:                      r.ID,
				Name:                    r.CharacterName,
				Side:                    domain.SideParty,
				TorsoOnly:               r.TorsoOnly == 1,
				Level:                   int(r.Level),
				Initiative:              int(r.Initiative),
				HP:                      int(r.Hp),
				MaxHP:                   int(r.MaxHp),
				Defense:                 int(r.Defense),
				DefenseHead:             int(r.DefenseHead),
				DefenseTorso:            int(r.DefenseTorso),
				DefenseLeftArm:          int(r.DefenseLeftArm),
				DefenseRightArm:         int(r.DefenseRightArm),
				DefenseLeftLeg:          int(r.DefenseLeftLeg),
				DefenseRightLeg:         int(r.DefenseRightLeg),
				ResistPhysicalHead:      int(r.DamageResistancePhysicalHead),
				ResistPhysicalTorso:     int(r.DamageResistancePhysicalTorso),
				ResistPhysicalLeftArm:   int(r.DamageResistancePhysicalLeftArm),
				ResistPhysicalRightArm:  int(r.DamageResistancePhysicalRightArm),
				ResistPhysicalLeftLeg:   int(r.DamageResistancePhysicalLeftLeg),
				ResistPhysicalRightLeg:  int(r.DamageResistancePhysicalRightLeg),
				ResistEnergyHead:        int(r.DamageResistanceEnergyHead),
				ResistEnergyTorso:       int(r.DamageResistanceEnergyTorso),
				ResistEnergyLeftArm:     int(r.DamageResistanceEnergyLeftArm),
				ResistEnergyRightArm:    int(r.DamageResistanceEnergyRightArm),
				ResistEnergyLeftLeg:     int(r.DamageResistanceEnergyLeftLeg),
				ResistEnergyRightLeg:    int(r.DamageResistanceEnergyRightLeg),
				ResistRadiationHead:     int(r.DamageResistanceRadiationHead),
				ResistRadiationTorso:    int(r.DamageResistanceRadiationTorso),
				ResistRadiationLeftArm:  int(r.DamageResistanceRadiationLeftArm),
				ResistRadiationRightArm: int(r.DamageResistanceRadiationRightArm),
				ResistRadiationLeftLeg:  int(r.DamageResistanceRadiationLeftLeg),
				ResistRadiationRightLeg: int(r.DamageResistanceRadiationRightLeg),
				ResistPhysical:          int(r.DamageResistancePhysical),
				ResistEnergy:            int(r.DamageResistanceEnergy),
				ResistRadiation:         int(r.DamageResistanceRadiation),
				ResistPoison:            int(r.DamageResistancePoison),
				ImmunePhysical:          r.DamageResistancePhysicalImmune == 1,
				ImmuneEnergy:            r.DamageResistanceEnergyImmune == 1,
				ImmuneRadiation:         r.DamageResistanceRadiationImmune == 1,
				ImmunePoison:            r.DamageResistancePoisonImmune == 1,
			},
		})
	}
	return players, nil
}

func (s *EncounterStore) CreateCampaign(campaignID, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	ctx := s.ctx
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
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
		StartDate: startDate,
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
		if err = qtx.InsertPlayerCharacter(ctx, dbgen.InsertPlayerCharacterParams{
			ID:         charID,
			PlayerID:   playerID,
			CampaignID: campaignID,
			Name:       strings.TrimSpace(p.Character.Name),
			Level:      int64(p.Character.Level),
			Initiative: int64(p.Character.Initiative),
			Hp:         int64(p.Character.HP),
			MaxHp:      int64(p.Character.MaxHP),
			Defense:    int64(p.Character.Defense),
			TorsoOnly:  boolToInt64(p.Character.TorsoOnly),
			Active:     1,
		}); err != nil {
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

func (s *EncounterStore) UpdateCampaign(campaignID, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	ctx := s.ctx
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
		StartDate:  startDate,
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
			if err = qtx.InsertPlayerCharacter(ctx, dbgen.InsertPlayerCharacterParams{
				ID:         charID,
				PlayerID:   playerID,
				CampaignID: campaignID,
				Name:       targetCharacterName,
				Level:      int64(p.Character.Level),
				Initiative: int64(p.Character.Initiative),
				Hp:         int64(p.Character.HP),
				MaxHp:      int64(p.Character.MaxHP),
				Defense:    int64(p.Character.Defense),
				TorsoOnly:  boolToInt64(p.Character.TorsoOnly),
				Active:     1,
			}); err != nil {
				return nil, fmt.Errorf("insert player character: %w", err)
			}
		} else {
			if err = updateActivePlayerCharacter(ctx, qtx, charID, campaignID, p.Character); err != nil {
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

func updateActivePlayerCharacter(ctx context.Context, qtx *dbgen.Queries, characterID, campaignID string, c domain.Combatant) error {
	return qtx.UpdateActivePlayerCharacterByID(ctx, dbgen.UpdateActivePlayerCharacterByIDParams{
		CharacterID: characterID,
		CampaignID:  campaignID,
		Name:        strings.TrimSpace(c.Name),
		Level:       int64(c.Level),
		Initiative:  int64(c.Initiative),
		Hp:          int64(c.HP),
		MaxHp:       int64(c.MaxHP),
		Defense:     int64(c.Defense),
		TorsoOnly:   boolToInt64(c.TorsoOnly),
	})
}

func normalizeNameKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (s *EncounterStore) GetActiveCampaign() (*domain.Campaign, error) {
	ctx := s.ctx
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
	return &domain.Campaign{
		ID:        row.ID,
		Name:      row.Name,
		StartDate: row.StartDate,
		UpdatedAt: row.UpdatedAt.Format("2006-01-02 15:04:05.000"),
	}, nil
}

func (s *EncounterStore) ListCampaigns() ([]domain.Campaign, error) {
	ctx := s.ctx
	rows, err := s.q.ListCampaigns(ctx)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	result := make([]domain.Campaign, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Campaign{
			ID:        row.ID,
			Name:      row.Name,
			StartDate: row.StartDate,
			UpdatedAt: row.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	return result, nil
}

func (s *EncounterStore) ActivateCampaign(campaignID string) error {
	ctx := s.ctx
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

func (s *EncounterStore) Activate(encounterID string) error {
	ctx := s.ctx
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

func (s *EncounterStore) SoftDelete(encounterID string) error {
	ctx := s.ctx
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

func (s *EncounterStore) AppendEncounterLog(encounterID string, round int, message string) error {
	if encounterID == "" {
		return fmt.Errorf("encounter id is required")
	}
	if message == "" {
		return fmt.Errorf("log message is required")
	}

	ctx := s.ctx
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

func (s *EncounterStore) ListEncounterLogs(encounterID string) ([]domain.EncounterLog, error) {
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}

	ctx := s.ctx
	rows, err := s.q.ListEncounterLogsByEncounterID(ctx, encounterID)
	if err != nil {
		return nil, fmt.Errorf("list encounter logs: %w", err)
	}

	logs := make([]domain.EncounterLog, 0, len(rows))
	for _, r := range rows {
		logs = append(logs, domain.EncounterLog{
			Round:     int(r.Round),
			Message:   r.Message,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	return logs, nil
}

const (
	bodyLocationHead     int64 = 1
	bodyLocationTorso    int64 = 2
	bodyLocationLeftArm  int64 = 3
	bodyLocationRightArm int64 = 4
	bodyLocationLeftLeg  int64 = 5
	bodyLocationRightLeg int64 = 6

	damageTypePhysical  int64 = 1
	damageTypeEnergy    int64 = 2
	damageTypeRadiation int64 = 3
	damageTypePoison    int64 = 4
)

func upsertCombatantNormalizedStats(ctx context.Context, qtx *dbgen.Queries, combatantID string, c domain.Combatant) error {
	defenseByLocation := []struct {
		bodyLocationID int64
		defense        int64
	}{
		{bodyLocationHead, int64(c.DefenseHead)},
		{bodyLocationTorso, int64(c.DefenseTorso)},
		{bodyLocationLeftArm, int64(c.DefenseLeftArm)},
		{bodyLocationRightArm, int64(c.DefenseRightArm)},
		{bodyLocationLeftLeg, int64(c.DefenseLeftLeg)},
		{bodyLocationRightLeg, int64(c.DefenseRightLeg)},
	}
	for _, stat := range defenseByLocation {
		if err := qtx.UpsertCombatantDefenseByLocation(ctx, dbgen.UpsertCombatantDefenseByLocationParams{
			CombatantID:    combatantID,
			BodyLocationID: stat.bodyLocationID,
			Defense:        stat.defense,
		}); err != nil {
			return fmt.Errorf("upsert defense by location: %w", err)
		}
	}

	resistanceGlobal := []struct {
		damageTypeID int64
		resistance   int64
		immune       int64
	}{
		{damageTypePhysical, int64(c.ResistPhysical), boolToInt64(c.ImmunePhysical)},
		{damageTypeEnergy, int64(c.ResistEnergy), boolToInt64(c.ImmuneEnergy)},
		{damageTypeRadiation, int64(c.ResistRadiation), boolToInt64(c.ImmuneRadiation)},
		{damageTypePoison, int64(c.ResistPoison), boolToInt64(c.ImmunePoison)},
	}
	for _, stat := range resistanceGlobal {
		if err := qtx.UpsertCombatantResistanceGlobal(ctx, dbgen.UpsertCombatantResistanceGlobalParams{
			CombatantID:  combatantID,
			DamageTypeID: stat.damageTypeID,
			Resistance:   stat.resistance,
			Immune:       stat.immune,
		}); err != nil {
			return fmt.Errorf("upsert global resistance: %w", err)
		}
	}

	bodyLocationIDs := []int64{
		bodyLocationHead,
		bodyLocationTorso,
		bodyLocationLeftArm,
		bodyLocationRightArm,
		bodyLocationLeftLeg,
		bodyLocationRightLeg,
	}
	resistanceByLocation := []struct {
		damageTypeID int64
		values       []int64
	}{
		{damageTypePhysical, []int64{
			int64(c.ResistPhysicalHead),
			int64(c.ResistPhysicalTorso),
			int64(c.ResistPhysicalLeftArm),
			int64(c.ResistPhysicalRightArm),
			int64(c.ResistPhysicalLeftLeg),
			int64(c.ResistPhysicalRightLeg),
		}},
		{damageTypeEnergy, []int64{
			int64(c.ResistEnergyHead),
			int64(c.ResistEnergyTorso),
			int64(c.ResistEnergyLeftArm),
			int64(c.ResistEnergyRightArm),
			int64(c.ResistEnergyLeftLeg),
			int64(c.ResistEnergyRightLeg),
		}},
		{damageTypeRadiation, []int64{
			int64(c.ResistRadiationHead),
			int64(c.ResistRadiationTorso),
			int64(c.ResistRadiationLeftArm),
			int64(c.ResistRadiationRightArm),
			int64(c.ResistRadiationLeftLeg),
			int64(c.ResistRadiationRightLeg),
		}},
	}
	for _, stat := range resistanceByLocation {
		for idx, bodyLocationID := range bodyLocationIDs {
			if err := qtx.UpsertCombatantResistanceByLocation(ctx, dbgen.UpsertCombatantResistanceByLocationParams{
				CombatantID:    combatantID,
				DamageTypeID:   stat.damageTypeID,
				BodyLocationID: bodyLocationID,
				Resistance:     stat.values[idx],
			}); err != nil {
				return fmt.Errorf("upsert location resistance: %w", err)
			}
		}
	}
	return nil
}

func upsertPlayerCharacterNormalizedStats(ctx context.Context, qtx *dbgen.Queries, playerCharacterID string, c domain.Combatant) error {
	defenseByLocation := []struct {
		bodyLocationID int64
		defense        int64
	}{
		{bodyLocationHead, int64(c.DefenseHead)},
		{bodyLocationTorso, int64(c.DefenseTorso)},
		{bodyLocationLeftArm, int64(c.DefenseLeftArm)},
		{bodyLocationRightArm, int64(c.DefenseRightArm)},
		{bodyLocationLeftLeg, int64(c.DefenseLeftLeg)},
		{bodyLocationRightLeg, int64(c.DefenseRightLeg)},
	}
	for _, stat := range defenseByLocation {
		if err := qtx.UpsertPlayerCharacterDefenseByLocation(ctx, dbgen.UpsertPlayerCharacterDefenseByLocationParams{
			PlayerCharacterID: playerCharacterID,
			BodyLocationID:    stat.bodyLocationID,
			Defense:           stat.defense,
		}); err != nil {
			return fmt.Errorf("upsert defense by location: %w", err)
		}
	}

	resistanceGlobal := []struct {
		damageTypeID int64
		resistance   int64
		immune       int64
	}{
		{damageTypePhysical, int64(c.ResistPhysical), boolToInt64(c.ImmunePhysical)},
		{damageTypeEnergy, int64(c.ResistEnergy), boolToInt64(c.ImmuneEnergy)},
		{damageTypeRadiation, int64(c.ResistRadiation), boolToInt64(c.ImmuneRadiation)},
		{damageTypePoison, int64(c.ResistPoison), boolToInt64(c.ImmunePoison)},
	}
	for _, stat := range resistanceGlobal {
		if err := qtx.UpsertPlayerCharacterResistanceGlobal(ctx, dbgen.UpsertPlayerCharacterResistanceGlobalParams{
			PlayerCharacterID: playerCharacterID,
			DamageTypeID:      stat.damageTypeID,
			Resistance:        stat.resistance,
			Immune:            stat.immune,
		}); err != nil {
			return fmt.Errorf("upsert global resistance: %w", err)
		}
	}

	bodyLocationIDs := []int64{
		bodyLocationHead,
		bodyLocationTorso,
		bodyLocationLeftArm,
		bodyLocationRightArm,
		bodyLocationLeftLeg,
		bodyLocationRightLeg,
	}
	resistanceByLocation := []struct {
		damageTypeID int64
		values       []int64
	}{
		{damageTypePhysical, []int64{
			int64(c.ResistPhysicalHead),
			int64(c.ResistPhysicalTorso),
			int64(c.ResistPhysicalLeftArm),
			int64(c.ResistPhysicalRightArm),
			int64(c.ResistPhysicalLeftLeg),
			int64(c.ResistPhysicalRightLeg),
		}},
		{damageTypeEnergy, []int64{
			int64(c.ResistEnergyHead),
			int64(c.ResistEnergyTorso),
			int64(c.ResistEnergyLeftArm),
			int64(c.ResistEnergyRightArm),
			int64(c.ResistEnergyLeftLeg),
			int64(c.ResistEnergyRightLeg),
		}},
		{damageTypeRadiation, []int64{
			int64(c.ResistRadiationHead),
			int64(c.ResistRadiationTorso),
			int64(c.ResistRadiationLeftArm),
			int64(c.ResistRadiationRightArm),
			int64(c.ResistRadiationLeftLeg),
			int64(c.ResistRadiationRightLeg),
		}},
	}
	for _, stat := range resistanceByLocation {
		for idx, bodyLocationID := range bodyLocationIDs {
			if err := qtx.UpsertPlayerCharacterResistanceByLocation(ctx, dbgen.UpsertPlayerCharacterResistanceByLocationParams{
				PlayerCharacterID: playerCharacterID,
				DamageTypeID:      stat.damageTypeID,
				BodyLocationID:    bodyLocationID,
				Resistance:        stat.values[idx],
			}); err != nil {
				return fmt.Errorf("upsert location resistance: %w", err)
			}
		}
	}
	return nil
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
	default:
		return ""
	}
}
