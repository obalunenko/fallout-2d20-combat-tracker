package sqlite

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

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
