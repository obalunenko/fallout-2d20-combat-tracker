package sqlite

import (
	"context"
	"database/sql"
	"fmt"

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
