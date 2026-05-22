package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

type EncounterRepository interface {
	Get(ctx context.Context) (*domain.Encounter, error)
	Save(ctx context.Context, encounter *domain.Encounter) error
	List(ctx context.Context) ([]domain.EncounterSummary, error)
	GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error)
	UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error)
	ListPartyMembers(ctx context.Context) ([]domain.Combatant, error)
	ListMonsterTemplates(ctx context.Context) ([]domain.Combatant, error)
	UpsertMonsterTemplate(ctx context.Context, monster domain.Combatant) (domain.Combatant, error)
	CreateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	GetActiveCampaign(ctx context.Context) (*domain.Campaign, error)
	ListCampaigns(ctx context.Context) ([]domain.Campaign, error)
	ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error)
	ActivateCampaign(ctx context.Context, campaignID string) error
	Activate(ctx context.Context, encounterID string) error
	SoftDelete(ctx context.Context, encounterID string) error
	AppendEncounterLog(ctx context.Context, encounterID string, round int, message string) error
	ListEncounterLogs(ctx context.Context, encounterID string) ([]domain.EncounterLog, error)
}

type CreateCampaignCommand struct {
	ID        string
	Name      string
	StartDate time.Time
	Players   []domain.NewCampaignPlayer
}

type UpdateCampaignCommand struct {
	CampaignID string
	Name       string
	StartDate  time.Time
	Players    []domain.NewCampaignPlayer
}

type CreateEncounterCommand struct {
	ID         string
	Name       string
	Combatants []domain.Combatant
}

type UpdateEncounterCommand struct {
	EncounterID string
	Name        string
	Combatants  []domain.Combatant
}

type ApplyDamageCommand struct {
	CombatantID string
	DamageType  domain.DamageType
	Location    domain.BodyLocation
	Amount      int
}

type HealCommand struct {
	CombatantID string
	Amount      int
}

type SaveMonsterTemplatesCommand struct {
	Monsters []domain.Combatant
}

type Service struct {
	repo EncounterRepository
	logf func(string, ...any)

	sideEffectFailuresMu sync.Mutex
	sideEffectFailures   map[string]uint64
	operationTimeout     time.Duration
}

type sideEffectCategory string

const (
	sideEffectCategoryAudit         sideEffectCategory = "audit"
	sideEffectCategoryTelemetry     sideEffectCategory = "telemetry"
	sideEffectCategoryNotifications sideEffectCategory = "notifications"

	sideEffectNameAppendEncounterLog = "append_encounter_log"
	defaultOperationTimeout          = 5 * time.Second
)

func NewService(repo EncounterRepository) *Service {
	return NewServiceWithLogfAndTimeout(repo, log.Printf, defaultOperationTimeout)
}

func NewServiceWithLogf(repo EncounterRepository, logf func(string, ...any)) *Service {
	return NewServiceWithLogfAndTimeout(repo, logf, defaultOperationTimeout)
}

func NewServiceWithLogfAndTimeout(repo EncounterRepository, logf func(string, ...any), operationTimeout time.Duration) *Service {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Service{
		repo:               repo,
		logf:               logf,
		sideEffectFailures: make(map[string]uint64),
		operationTimeout:   operationTimeout,
	}
}

func (s *Service) contextForOperation(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s.operationTimeout <= 0 {
		return ctx, func() {}
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.operationTimeout)
}

func (s *Service) GetEncounter(ctx context.Context) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.Get(ctx)
}

func (s *Service) ListEncounters(ctx context.Context) ([]domain.EncounterSummary, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.List(ctx)
}

func (s *Service) ListEncounterLogs(ctx context.Context, encounterID string) ([]domain.EncounterLog, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.ListEncounterLogs(ctx, encounterID)
}

func (s *Service) ListPartyMembers(ctx context.Context) ([]domain.Combatant, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.ListPartyMembers(ctx)
}

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
		monster := cmd.Monsters[i]
		monster.Name = strings.TrimSpace(monster.Name)
		if monster.Name == "" {
			return nil, fmt.Errorf("monster name is required")
		}
		key := strings.ToLower(monster.Name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if monster.Level < 1 {
			return nil, fmt.Errorf("monster %q: invalid level", monster.Name)
		}
		if monster.XP < 0 {
			return nil, fmt.Errorf("monster %q: invalid XP", monster.Name)
		}
		if monster.Initiative < 0 {
			return nil, fmt.Errorf("monster %q: invalid initiative", monster.Name)
		}
		if monster.HP < 0 {
			return nil, fmt.Errorf("monster %q: invalid HP", monster.Name)
		}
		if monster.MaxHP <= 0 {
			if monster.HP > 0 {
				monster.MaxHP = monster.HP
			} else {
				monster.MaxHP = 1
			}
		}
		if monster.HP > monster.MaxHP {
			return nil, fmt.Errorf("monster %q: current HP cannot exceed max HP", monster.Name)
		}
		if monster.Defense < 0 {
			return nil, fmt.Errorf("monster %q: invalid defense", monster.Name)
		}
		if strings.TrimSpace(monster.ID) == "" {
			monster.ID = uuid.NewString()
		}
		monster.Side = domain.SideNPC
		monster.PlayerCharacterID = ""
		monster.Active = false
		monster.Defeated = false
		domain.NormalizeCombatantHP(&monster)

		created, err := s.repo.UpsertMonsterTemplate(ctx, monster)
		if err != nil {
			return nil, err
		}
		saved = append(saved, created)
	}
	return saved, nil
}

func (s *Service) GetEncounterByID(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.GetEncounterByID(ctx, encounterID)
}

func (s *Service) CreateCampaign(ctx context.Context, id, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return s.ExecuteCreateCampaign(ctx, CreateCampaignCommand{
		ID:        id,
		Name:      name,
		StartDate: startDate,
		Players:   players,
	})
}

func (s *Service) ExecuteCreateCampaign(ctx context.Context, cmd CreateCampaignCommand) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	if strings.TrimSpace(cmd.ID) == "" {
		cmd.ID = uuid.NewString()
	}
	for i := range cmd.Players {
		if strings.TrimSpace(cmd.Players[i].PlayerName) == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if strings.TrimSpace(cmd.Players[i].Character.Name) == "" {
			return nil, fmt.Errorf("character name is required for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Level < 1 {
			return nil, fmt.Errorf("invalid level for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.HP < 0 {
			return nil, fmt.Errorf("invalid HP for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.MaxHP <= 0 {
			if cmd.Players[i].Character.HP > 0 {
				cmd.Players[i].Character.MaxHP = cmd.Players[i].Character.HP
			} else {
				cmd.Players[i].Character.MaxHP = 1
			}
		}
		if cmd.Players[i].Character.HP > cmd.Players[i].Character.MaxHP {
			return nil, fmt.Errorf("current HP cannot exceed max HP for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Initiative < 0 {
			return nil, fmt.Errorf("invalid initiative for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Defense < 0 {
			return nil, fmt.Errorf("invalid defense for player %q", cmd.Players[i].PlayerName)
		}
		if strings.TrimSpace(cmd.Players[i].Character.ID) == "" {
			cmd.Players[i].Character.ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&cmd.Players[i].Character)
		cmd.Players[i].Character.Side = domain.SideParty
		cmd.Players[i].Character.XP = 0
	}
	return s.repo.CreateCampaign(ctx, cmd.ID, cmd.Name, cmd.StartDate, cmd.Players)
}

func (s *Service) GetActiveCampaign(ctx context.Context) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.GetActiveCampaign(ctx)
}

func (s *Service) ListCampaigns(ctx context.Context) ([]domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	return s.repo.ListCampaigns(ctx)
}

func (s *Service) ListCampaignPlayers(ctx context.Context, campaignID string) ([]domain.NewCampaignPlayer, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return s.repo.ListCampaignPlayers(ctx, campaignID)
}

func (s *Service) ActivateCampaign(ctx context.Context, campaignID string) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if err := s.repo.ActivateCampaign(ctx, campaignID); err != nil {
		return nil, err
	}
	return s.repo.GetActiveCampaign(ctx)
}

func (s *Service) UpdateCampaign(ctx context.Context, campaignID, name string, startDate time.Time, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	return s.ExecuteUpdateCampaign(ctx, UpdateCampaignCommand{
		CampaignID: campaignID,
		Name:       name,
		StartDate:  startDate,
		Players:    players,
	})
}

func (s *Service) ExecuteUpdateCampaign(ctx context.Context, cmd UpdateCampaignCommand) (*domain.Campaign, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.CampaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if cmd.StartDate.IsZero() {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(cmd.Players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	for i := range cmd.Players {
		if strings.TrimSpace(cmd.Players[i].PlayerName) == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if strings.TrimSpace(cmd.Players[i].Character.Name) == "" {
			return nil, fmt.Errorf("character name is required for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Level < 1 {
			return nil, fmt.Errorf("invalid level for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.HP < 0 {
			return nil, fmt.Errorf("invalid HP for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.MaxHP <= 0 {
			if cmd.Players[i].Character.HP > 0 {
				cmd.Players[i].Character.MaxHP = cmd.Players[i].Character.HP
			} else {
				cmd.Players[i].Character.MaxHP = 1
			}
		}
		if cmd.Players[i].Character.HP > cmd.Players[i].Character.MaxHP {
			return nil, fmt.Errorf("current HP cannot exceed max HP for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Initiative < 0 {
			return nil, fmt.Errorf("invalid initiative for player %q", cmd.Players[i].PlayerName)
		}
		if cmd.Players[i].Character.Defense < 0 {
			return nil, fmt.Errorf("invalid defense for player %q", cmd.Players[i].PlayerName)
		}
		if strings.TrimSpace(cmd.Players[i].Character.ID) == "" {
			cmd.Players[i].Character.ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&cmd.Players[i].Character)
		cmd.Players[i].Character.Side = domain.SideParty
		cmd.Players[i].Character.XP = 0
	}
	return s.repo.UpdateCampaign(ctx, cmd.CampaignID, cmd.Name, cmd.StartDate, cmd.Players)
}

func (s *Service) ActivateEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if err := s.repo.Activate(ctx, encounterID); err != nil {
		return nil, err
	}
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, "Encounter activated")
	return enc, nil
}

func (s *Service) RestartEncounter(ctx context.Context, encounterID string) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if err := s.repo.Activate(ctx, encounterID); err != nil {
		return nil, err
	}

	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	for i := range enc.Combatants {
		if enc.Combatants[i].Side != domain.SideNPC {
			continue
		}
		domain.NormalizeCombatantHP(&enc.Combatants[i])
		enc.Combatants[i].HP = enc.Combatants[i].MaxHP
		enc.Combatants[i].Defeated = false
	}
	restarted := domain.NewEncounter(enc.ID, enc.Name, enc.Combatants)
	if err := s.repo.Save(ctx, restarted); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, restarted, "Encounter restarted")
	return restarted, nil
}

func (s *Service) DeleteEncounter(ctx context.Context, encounterID string) error {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return fmt.Errorf("encounter id is required")
	}
	return s.repo.SoftDelete(ctx, encounterID)
}

func (s *Service) CreateEncounter(ctx context.Context, id, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteCreateEncounter(ctx, CreateEncounterCommand{
		ID:         id,
		Name:       name,
		Combatants: combatants,
	})
}

func (s *Service) ExecuteCreateEncounter(ctx context.Context, cmd CreateEncounterCommand) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if len(cmd.Combatants) == 0 {
		return nil, fmt.Errorf("cannot create encounter without combatants")
	}
	activeCampaign, err := s.repo.GetActiveCampaign(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cmd.ID) == "" {
		cmd.ID = uuid.NewString()
	}
	for i := range cmd.Combatants {
		if strings.TrimSpace(cmd.Combatants[i].ID) == "" {
			cmd.Combatants[i].ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&cmd.Combatants[i])
	}
	enc := domain.NewEncounter(cmd.ID, cmd.Name, cmd.Combatants)
	enc.CampaignID = activeCampaign.ID
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Encounter created (%s)", cmd.Name))
	return enc, nil
}

func (s *Service) UpdateEncounter(ctx context.Context, encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	return s.ExecuteUpdateEncounter(ctx, UpdateEncounterCommand{
		EncounterID: encounterID,
		Name:        name,
		Combatants:  combatants,
	})
}

func (s *Service) ExecuteUpdateEncounter(ctx context.Context, cmd UpdateEncounterCommand) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if strings.TrimSpace(cmd.EncounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, fmt.Errorf("encounter name is required")
	}
	if len(cmd.Combatants) == 0 {
		return nil, fmt.Errorf("cannot update encounter without combatants")
	}
	for i := range cmd.Combatants {
		if strings.TrimSpace(cmd.Combatants[i].ID) == "" {
			cmd.Combatants[i].ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&cmd.Combatants[i])
	}
	enc, err := s.repo.UpdateEncounter(ctx, cmd.EncounterID, cmd.Name, cmd.Combatants)
	if err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Encounter updated (%s)", cmd.Name))
	return enc, nil
}

func (s *Service) AdvanceTurn(ctx context.Context) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := enc.AdvanceTurn(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	if active := enc.ActiveCombatant(); active != nil {
		s.appendOperationLog(ctx, enc, fmt.Sprintf("Turn advanced -> %s", active.Name))
	}
	return enc, nil
}

func (s *Service) AddPartyAP(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	enc.AddPartyAP(v)
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Party AP %+d (total: %d)", v, enc.Resources.PartyAP))
	return enc, nil
}

func (s *Service) SpendPartyAP(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := enc.SpendPartyAP(v); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("Party AP -%d (total: %d)", v, enc.Resources.PartyAP))
	return enc, nil
}

func (s *Service) AddThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	enc.AddThreat(v)
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("GM Threat %+d (total: %d)", v, enc.Resources.GMThreat))
	return enc, nil
}

func (s *Service) SpendThreat(ctx context.Context, v int) (*domain.Encounter, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	enc, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if err := enc.SpendThreat(v); err != nil {
		return nil, err
	}
	if err := s.repo.Save(ctx, enc); err != nil {
		return nil, err
	}
	s.appendOperationLog(ctx, enc, fmt.Sprintf("GM Threat -%d (total: %d)", v, enc.Resources.GMThreat))
	return enc, nil
}

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

func (s *Service) appendOperationLog(ctx context.Context, enc *domain.Encounter, message string) {
	if enc == nil || enc.ID == "" || message == "" {
		return
	}
	s.runNonCriticalSideEffect(sideEffectCategoryAudit, sideEffectNameAppendEncounterLog, func() error {
		if err := s.repo.AppendEncounterLog(ctx, enc.ID, enc.Round, message); err != nil {
			return fmt.Errorf("encounter_id=%s round=%d message=%q: %w", enc.ID, enc.Round, message, err)
		}
		return nil
	})
}

func (s *Service) runNonCriticalSideEffect(category sideEffectCategory, name string, run func() error) {
	if strings.TrimSpace(string(category)) == "" || strings.TrimSpace(name) == "" || run == nil {
		return
	}

	sideEffectID := fmt.Sprintf("%s.%s", category, name)
	if err := run(); err != nil {
		failures := s.recordSideEffectFailure(sideEffectID)
		s.logf("non-critical side effect failed: side_effect=%s failures=%d err=%v", sideEffectID, failures, err)
	}
}

func (s *Service) recordSideEffectFailure(sideEffectID string) uint64 {
	if strings.TrimSpace(sideEffectID) == "" {
		return 0
	}
	s.sideEffectFailuresMu.Lock()
	defer s.sideEffectFailuresMu.Unlock()

	s.sideEffectFailures[sideEffectID]++
	return s.sideEffectFailures[sideEffectID]
}

func (s *Service) sideEffectFailureCount(sideEffectID string) uint64 {
	if strings.TrimSpace(sideEffectID) == "" {
		return 0
	}
	s.sideEffectFailuresMu.Lock()
	defer s.sideEffectFailuresMu.Unlock()

	return s.sideEffectFailures[sideEffectID]
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
