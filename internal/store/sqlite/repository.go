package sqlite

import (
	"context"
	"database/sql"
	"fmt"

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

	encRow, err := s.q.GetLatestEncounter(ctx)
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
			ID:              c.ID,
			Name:            c.Name,
			Side:            domain.Side(c.Side),
			Level:           int(c.Level),
			XP:              int(c.Xp),
			Initiative:      int(c.Initiative),
			HP:              int(c.Hp),
			Defense:         int(c.Defense),
			ResistPhysical:  int(c.DamageResistancePhysical),
			ResistEnergy:    int(c.DamageResistanceEnergy),
			ResistRadiation: int(c.DamageResistanceRadiation),
			ResistPoison:    int(c.DamageResistancePoison),
			ImmunePhysical:  c.DamageResistancePhysicalImmune == 1,
			ImmuneEnergy:    c.DamageResistanceEnergyImmune == 1,
			ImmuneRadiation: c.DamageResistanceRadiationImmune == 1,
			ImmunePoison:    c.DamageResistancePoisonImmune == 1,
			Active:          c.Active == 1,
			Defeated:        c.Defeated == 1,
		})
	}

	return &domain.Encounter{
		ID:         encRow.ID,
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
	if err = qtx.UpsertEncounter(ctx, dbgen.UpsertEncounterParams{
		ID:        enc.ID,
		Name:      enc.Name,
		Round:     int64(enc.Round),
		TurnIndex: int64(enc.TurnIndex),
		PartyAp:   int64(enc.Resources.PartyAP),
		GmThreat:  int64(enc.Resources.GMThreat),
	}); err != nil {
		return fmt.Errorf("upsert encounter: %w", err)
	}

	if err = qtx.DeleteCombatantsByEncounterID(ctx, enc.ID); err != nil {
		return fmt.Errorf("clear combatants: %w", err)
	}

	for i, c := range enc.Combatants {
		if err = qtx.InsertCombatant(ctx, dbgen.InsertCombatantParams{
			ID:                              c.ID,
			EncounterID:                     enc.ID,
			Name:                            c.Name,
			Side:                            string(c.Side),
			Level:                           int64(c.Level),
			Xp:                              int64(c.XP),
			Initiative:                      int64(c.Initiative),
			Hp:                              int64(c.HP),
			Defense:                         int64(c.Defense),
			DamageResistance:                int64(c.ResistPhysical),
			DamageResistancePhysical:        int64(c.ResistPhysical),
			DamageResistanceEnergy:          int64(c.ResistEnergy),
			DamageResistanceRadiation:       int64(c.ResistRadiation),
			DamageResistancePoison:          int64(c.ResistPoison),
			DamageResistancePhysicalImmune:  boolToInt64(c.ImmunePhysical),
			DamageResistanceEnergyImmune:    boolToInt64(c.ImmuneEnergy),
			DamageResistanceRadiationImmune: boolToInt64(c.ImmuneRadiation),
			DamageResistancePoisonImmune:    boolToInt64(c.ImmunePoison),
			Active:                          boolToInt64(c.Active),
			Defeated:                        boolToInt64(c.Defeated),
			Position:                        int64(i),
		}); err != nil {
			return fmt.Errorf("insert combatant %s: %w", c.ID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (s *EncounterStore) List() ([]domain.EncounterSummary, error) {
	ctx := s.ctx
	rows, err := s.q.ListEncounterSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list encounters: %w", err)
	}

	summaries := make([]domain.EncounterSummary, 0, len(rows))
	for _, r := range rows {
		summaries = append(summaries, domain.EncounterSummary{
			ID:         r.ID,
			Name:       r.Name,
			Round:      int(r.Round),
			Combatants: int(r.Combatants),
			UpdatedAt:  r.UpdatedAt.Format("2006-01-02 15:04:05.000"),
		})
	}
	return summaries, nil
}

func (s *EncounterStore) Activate(encounterID string) error {
	ctx := s.ctx
	affected, err := s.q.ActivateEncounter(ctx, encounterID)
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
	affected, err := s.q.SoftDeleteEncounter(ctx, encounterID)
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

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
