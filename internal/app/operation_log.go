package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/obalunenko/fallout/internal/domain"
)

func (s *Service) ListEncounterLogs(ctx context.Context, encounterID string) ([]domain.EncounterLog, error) {
	ctx, cancel := s.contextForOperation(ctx)
	defer cancel()
	if encounterID == "" {
		return nil, fmt.Errorf("encounter id is required")
	}
	return s.repo.ListEncounterLogs(ctx, encounterID)
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
