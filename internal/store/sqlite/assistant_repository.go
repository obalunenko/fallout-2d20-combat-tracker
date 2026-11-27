package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/obalunenko/fallout/internal/store/sqlite/dbgen"
)

// assistantActivityTimeLayout matches STRFTIME('%Y-%m-%d %H:%M:%f'), so
// occurred_at sorts the same way as the timestamps SQLite writes itself.
const assistantActivityTimeLayout = "2006-01-02 15:04:05.000"

// GetAssistantConnectionSettings loads the single settings row. found is false,
// with no error, when the row has not been saved yet.
func (s *EncounterStore) GetAssistantConnectionSettings(ctx context.Context) (domain.AssistantConnectionSettings, bool, error) {
	ctx = normalizeContext(ctx)

	row, err := s.q.GetAssistantConnectionSettings(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AssistantConnectionSettings{}, false, nil
		}
		return domain.AssistantConnectionSettings{}, false, fmt.Errorf("get assistant connection settings: %w", err)
	}
	return assistantConnectionSettingsFromRow(row), true, nil
}

// SaveAssistantConnectionSettings creates or replaces the single settings row.
func (s *EncounterStore) SaveAssistantConnectionSettings(ctx context.Context, settings domain.AssistantConnectionSettings) error {
	ctx = normalizeContext(ctx)
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("validate assistant connection settings: %w", err)
	}

	if err := s.q.UpsertAssistantConnectionSettings(ctx, upsertAssistantConnectionSettingsParams(settings)); err != nil {
		return fmt.Errorf("save assistant connection settings: %w", err)
	}
	return nil
}

// AppendAssistantActivity records one entry and prunes the log to the newest
// 1,000 entries in the same transaction. An empty ID or zero OccurredAt is
// filled in.
func (s *EncounterStore) AppendAssistantActivity(ctx context.Context, entry domain.AssistantActivityEntry) error {
	ctx = normalizeContext(ctx)
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = time.Now().UTC()
	}

	return s.runInTx(ctx, func(q *dbgen.Queries) error {
		if err := q.InsertAssistantActivity(ctx, insertAssistantActivityParams(entry)); err != nil {
			return fmt.Errorf("insert assistant activity: %w", err)
		}
		if err := q.PruneAssistantActivity(ctx); err != nil {
			return fmt.Errorf("prune assistant activity: %w", err)
		}
		return nil
	})
}

// ListAssistantActivity returns up to limit entries, newest first.
func (s *EncounterStore) ListAssistantActivity(ctx context.Context, limit int) ([]domain.AssistantActivityEntry, error) {
	ctx = normalizeContext(ctx)
	if limit <= 0 {
		return nil, fmt.Errorf("assistant activity limit must be positive, got %d", limit)
	}

	rows, err := s.q.ListAssistantActivity(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("list assistant activity: %w", err)
	}

	entries := make([]domain.AssistantActivityEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, assistantActivityFromRow(r))
	}
	return entries, nil
}

// ClearAssistantActivity deletes every activity entry.
func (s *EncounterStore) ClearAssistantActivity(ctx context.Context) error {
	ctx = normalizeContext(ctx)
	if err := s.q.ClearAssistantActivity(ctx); err != nil {
		return fmt.Errorf("clear assistant activity: %w", err)
	}
	return nil
}
