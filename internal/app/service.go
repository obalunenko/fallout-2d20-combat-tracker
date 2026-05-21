package app

import (
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

type EncounterRepository interface {
	Get() (*domain.Encounter, error)
	Save(encounter *domain.Encounter) error
	List() ([]domain.EncounterSummary, error)
	GetEncounterByID(encounterID string) (*domain.Encounter, error)
	UpdateEncounter(encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error)
	ListPartyMembers() ([]domain.Combatant, error)
	CreateCampaign(campaignID, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	UpdateCampaign(campaignID, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error)
	GetActiveCampaign() (*domain.Campaign, error)
	ListCampaigns() ([]domain.Campaign, error)
	ListCampaignPlayers(campaignID string) ([]domain.NewCampaignPlayer, error)
	ActivateCampaign(campaignID string) error
	Activate(encounterID string) error
	SoftDelete(encounterID string) error
	AppendEncounterLog(encounterID string, round int, message string) error
	ListEncounterLogs(encounterID string) ([]domain.EncounterLog, error)
}

type Service struct {
	repo EncounterRepository
	logf func(string, ...any)
}

func NewService(repo EncounterRepository) *Service {
	return NewServiceWithLogf(repo, log.Printf)
}

func NewServiceWithLogf(repo EncounterRepository, logf func(string, ...any)) *Service {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &Service{
		repo: repo,
		logf: logf,
	}
}

func (s *Service) GetEncounter() (*domain.Encounter, error) {
	return s.repo.Get()
}

func (s *Service) ListEncounters() ([]domain.EncounterSummary, error) {
	return s.repo.List()
}

func (s *Service) ListEncounterLogs(encounterID string) ([]domain.EncounterLog, error) {
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.ListEncounterLogs(encounterID)
}

func (s *Service) ListPartyMembers() ([]domain.Combatant, error) {
	return s.repo.ListPartyMembers()
}

func (s *Service) GetEncounterByID(encounterID string) (*domain.Encounter, error) {
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.GetEncounterByID(encounterID)
}

func (s *Service) CreateCampaign(id, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if strings.TrimSpace(startDate) == "" {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}
	for i := range players {
		if strings.TrimSpace(players[i].PlayerName) == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if strings.TrimSpace(players[i].Character.Name) == "" {
			return nil, fmt.Errorf("character name is required for player %q", players[i].PlayerName)
		}
		if players[i].Character.Level < 1 {
			return nil, fmt.Errorf("invalid level for player %q", players[i].PlayerName)
		}
		if players[i].Character.HP < 0 {
			return nil, fmt.Errorf("invalid HP for player %q", players[i].PlayerName)
		}
		if players[i].Character.MaxHP <= 0 {
			if players[i].Character.HP > 0 {
				players[i].Character.MaxHP = players[i].Character.HP
			} else {
				players[i].Character.MaxHP = 1
			}
		}
		if players[i].Character.HP > players[i].Character.MaxHP {
			return nil, fmt.Errorf("current HP cannot exceed max HP for player %q", players[i].PlayerName)
		}
		if players[i].Character.Initiative < 0 {
			return nil, fmt.Errorf("invalid initiative for player %q", players[i].PlayerName)
		}
		if players[i].Character.Defense < 0 {
			return nil, fmt.Errorf("invalid defense for player %q", players[i].PlayerName)
		}
		if strings.TrimSpace(players[i].Character.ID) == "" {
			players[i].Character.ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&players[i].Character)
		players[i].Character.Side = domain.SideParty
		players[i].Character.XP = 0
	}
	return s.repo.CreateCampaign(id, name, startDate, players)
}

func (s *Service) GetActiveCampaign() (*domain.Campaign, error) {
	return s.repo.GetActiveCampaign()
}

func (s *Service) ListCampaigns() ([]domain.Campaign, error) {
	return s.repo.ListCampaigns()
}

func (s *Service) ListCampaignPlayers(campaignID string) ([]domain.NewCampaignPlayer, error) {
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return s.repo.ListCampaignPlayers(campaignID)
}

func (s *Service) ActivateCampaign(campaignID string) (*domain.Campaign, error) {
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if err := s.repo.ActivateCampaign(campaignID); err != nil {
		return nil, err
	}
	return s.repo.GetActiveCampaign()
}

func (s *Service) UpdateCampaign(campaignID, name, startDate string, players []domain.NewCampaignPlayer) (*domain.Campaign, error) {
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("campaign name is required")
	}
	if strings.TrimSpace(startDate) == "" {
		return nil, fmt.Errorf("campaign start date is required")
	}
	if len(players) == 0 {
		return nil, fmt.Errorf("add at least one player")
	}
	for i := range players {
		if strings.TrimSpace(players[i].PlayerName) == "" {
			return nil, fmt.Errorf("player name is required")
		}
		if strings.TrimSpace(players[i].Character.Name) == "" {
			return nil, fmt.Errorf("character name is required for player %q", players[i].PlayerName)
		}
		if players[i].Character.Level < 1 {
			return nil, fmt.Errorf("invalid level for player %q", players[i].PlayerName)
		}
		if players[i].Character.HP < 0 {
			return nil, fmt.Errorf("invalid HP for player %q", players[i].PlayerName)
		}
		if players[i].Character.MaxHP <= 0 {
			if players[i].Character.HP > 0 {
				players[i].Character.MaxHP = players[i].Character.HP
			} else {
				players[i].Character.MaxHP = 1
			}
		}
		if players[i].Character.HP > players[i].Character.MaxHP {
			return nil, fmt.Errorf("current HP cannot exceed max HP for player %q", players[i].PlayerName)
		}
		if players[i].Character.Initiative < 0 {
			return nil, fmt.Errorf("invalid initiative for player %q", players[i].PlayerName)
		}
		if players[i].Character.Defense < 0 {
			return nil, fmt.Errorf("invalid defense for player %q", players[i].PlayerName)
		}
		if strings.TrimSpace(players[i].Character.ID) == "" {
			players[i].Character.ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&players[i].Character)
		players[i].Character.Side = domain.SideParty
		players[i].Character.XP = 0
	}
	return s.repo.UpdateCampaign(campaignID, name, startDate, players)
}

func (s *Service) ActivateEncounter(encounterID string) (*domain.Encounter, error) {
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if err := s.repo.Activate(encounterID); err != nil {
		return nil, err
	}
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, "Encounter activated"); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) RestartEncounter(encounterID string) (*domain.Encounter, error) {
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if err := s.repo.Activate(encounterID); err != nil {
		return nil, err
	}

	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}

	restarted := domain.NewEncounter(enc.ID, enc.Name, enc.Combatants)
	if err := s.repo.Save(restarted); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(restarted, "Encounter restarted"); err != nil {
		return nil, err
	}
	return restarted, nil
}

func (s *Service) DeleteEncounter(encounterID string) error {
	if encounterID == "" {
		return fmt.Errorf("encounter id is required")
	}
	return s.repo.SoftDelete(encounterID)
}

func (s *Service) CreateEncounter(id, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	if len(combatants) == 0 {
		return nil, fmt.Errorf("cannot create encounter without combatants")
	}
	activeCampaign, err := s.repo.GetActiveCampaign()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}
	for i := range combatants {
		if strings.TrimSpace(combatants[i].ID) == "" {
			combatants[i].ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&combatants[i])
	}
	enc := domain.NewEncounter(id, name, combatants)
	enc.CampaignID = activeCampaign.ID
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Encounter created (%s)", name)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) UpdateEncounter(encounterID, name string, combatants []domain.Combatant) (*domain.Encounter, error) {
	if strings.TrimSpace(encounterID) == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("encounter name is required")
	}
	if len(combatants) == 0 {
		return nil, fmt.Errorf("cannot update encounter without combatants")
	}
	for i := range combatants {
		if strings.TrimSpace(combatants[i].ID) == "" {
			combatants[i].ID = uuid.NewString()
		}
		domain.NormalizeCombatantHP(&combatants[i])
	}
	enc, err := s.repo.UpdateEncounter(encounterID, name, combatants)
	if err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Encounter updated (%s)", name)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) AdvanceTurn() (*domain.Encounter, error) {
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if err := enc.AdvanceTurn(); err != nil {
		return nil, err
	}
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if active := enc.ActiveCombatant(); active != nil {
		if err := s.appendOperationLog(enc, fmt.Sprintf("Turn advanced -> %s", active.Name)); err != nil {
			return nil, err
		}
	}
	return enc, nil
}

func (s *Service) AddPartyAP(v int) (*domain.Encounter, error) {
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	enc.AddPartyAP(v)
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Party AP %+d (total: %d)", v, enc.Resources.PartyAP)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) SpendPartyAP(v int) (*domain.Encounter, error) {
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if err := enc.SpendPartyAP(v); err != nil {
		return nil, err
	}
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Party AP -%d (total: %d)", v, enc.Resources.PartyAP)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) AddThreat(v int) (*domain.Encounter, error) {
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	enc.AddThreat(v)
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("GM Threat %+d (total: %d)", v, enc.Resources.GMThreat)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) SpendThreat(v int) (*domain.Encounter, error) {
	enc, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	if err := enc.SpendThreat(v); err != nil {
		return nil, err
	}
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("GM Threat -%d (total: %d)", v, enc.Resources.GMThreat)); err != nil {
		return nil, err
	}
	return enc, nil
}

func (s *Service) ApplyDamage(combatantID string, damageType domain.DamageType, location domain.BodyLocation, amount int) (*domain.Encounter, int, error) {
	if combatantID == "" {
		return nil, 0, fmt.Errorf("combatant id is required")
	}
	enc, err := s.repo.Get()
	if err != nil {
		return nil, 0, err
	}

	applied, err := enc.ApplyDamage(combatantID, damageType, location, amount)
	if err != nil {
		return nil, 0, err
	}

	if err := s.repo.Save(enc); err != nil {
		return nil, 0, err
	}
	targetLabel := combatantID
	if combatant := findCombatantByID(enc, combatantID); combatant != nil {
		targetLabel = combatant.Name
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Damage -> %s type:%s location:%s raw:%d applied:%d", targetLabel, damageType, location, amount, applied)); err != nil {
		return nil, 0, err
	}
	return enc, applied, nil
}

func (s *Service) Heal(combatantID string, amount int) (*domain.Encounter, int, error) {
	if combatantID == "" {
		return nil, 0, fmt.Errorf("combatant id is required")
	}
	enc, err := s.repo.Get()
	if err != nil {
		return nil, 0, err
	}

	healed, err := enc.Heal(combatantID, amount)
	if err != nil {
		return nil, 0, err
	}

	if err := s.repo.Save(enc); err != nil {
		return nil, 0, err
	}
	targetLabel := combatantID
	if combatant := findCombatantByID(enc, combatantID); combatant != nil {
		targetLabel = combatant.Name
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Heal -> %s value:%d", targetLabel, healed)); err != nil {
		return nil, 0, err
	}
	return enc, healed, nil
}

func (s *Service) appendOperationLog(enc *domain.Encounter, message string) error {
	if enc == nil || enc.ID == "" || message == "" {
		return nil
	}
	if err := s.repo.AppendEncounterLog(enc.ID, enc.Round, message); err != nil {
		s.logf("append encounter log failed: encounter_id=%s round=%d message=%q err=%v", enc.ID, enc.Round, message, err)
		return nil
	}
	return nil
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
