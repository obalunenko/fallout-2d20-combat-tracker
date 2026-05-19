package app

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
)

type EncounterRepository interface {
	Get() (*domain.Encounter, error)
	Save(encounter *domain.Encounter) error
	List() ([]domain.EncounterSummary, error)
	ListPartyMembers() ([]domain.Combatant, error)
	Activate(encounterID string) error
	SoftDelete(encounterID string) error
	AppendEncounterLog(encounterID string, round int, message string) error
	ListEncounterLogs(encounterID string) ([]domain.EncounterLog, error)
}

type Service struct {
	repo EncounterRepository
}

func NewService(repo EncounterRepository) *Service {
	return &Service{repo: repo}
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
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}
	for i := range combatants {
		if strings.TrimSpace(combatants[i].ID) == "" {
			combatants[i].ID = uuid.NewString()
		}
	}
	enc := domain.NewEncounter(id, name, combatants)
	if err := s.repo.Save(enc); err != nil {
		return nil, err
	}
	if err := s.appendOperationLog(enc, fmt.Sprintf("Encounter created (%s)", name)); err != nil {
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

func (s *Service) ApplyDamage(combatantID string, damageType domain.DamageType, amount int) (*domain.Encounter, int, error) {
	if combatantID == "" {
		return nil, 0, fmt.Errorf("combatant id is required")
	}
	enc, err := s.repo.Get()
	if err != nil {
		return nil, 0, err
	}

	applied, err := enc.ApplyDamage(combatantID, damageType, amount)
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
	if err := s.appendOperationLog(enc, fmt.Sprintf("Damage -> %s type:%s raw:%d applied:%d", targetLabel, damageType, amount, applied)); err != nil {
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
	return s.repo.AppendEncounterLog(enc.ID, enc.Round, message)
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
