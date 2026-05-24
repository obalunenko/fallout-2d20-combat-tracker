package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *EncounterStore {
	t.Helper()
	return newTestStoreWithContext(t, t.Context())
}

func newTestStoreWithContext(t *testing.T, ctx context.Context) *EncounterStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "repo-test.db")
	db, err := OpenAndMigrate(dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	store := NewEncounterStoreWithContext(db, ctx)
	_, err = store.CreateCampaign(t.Context(), "repo-test-campaign", "Repo Test Campaign", testCampaignStartDate(t), []domain.NewCampaignPlayer{
		{
			PlayerName: "Player 1",
			Character: domain.Combatant{
				ID:         "repo-char-1",
				Name:       "Scout",
				Side:       domain.SideParty,
				Level:      1,
				Initiative: 7,
				HP:         6,
				Defense:    1,
			},
		},
	})
	require.NoError(t, err)
	return store
}

func testCampaignStartDate(t *testing.T) time.Time {
	t.Helper()
	startDate, err := domain.ParseCampaignStartDate("2026-01-01")
	require.NoError(t, err)
	return startDate
}

func dropNormalizedSyncTriggers(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_update;
		DROP TRIGGER IF EXISTS trg_sync_player_character_stats_after_insert;
		DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_update;
		DROP TRIGGER IF EXISTS trg_sync_combatant_stats_after_insert;
	`)
	require.NoError(t, err)
}

func queryInt64(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var v int64
	require.NoError(t, db.QueryRow(query, args...).Scan(&v))
	return v
}

type auditFields struct {
	createdAt auditTime
	updatedAt auditTime
	deletedAt auditTime
}

type auditTime struct {
	Time  time.Time
	Valid bool
}

func queryAuditFields(t *testing.T, db *sql.DB, table, id string) auditFields {
	t.Helper()
	return queryAuditFieldsBySQL(t, db, fmt.Sprintf(`SELECT created_at, updated_at, deleted_at FROM %s WHERE id = ?`, table), id)
}

func queryEncounterLogAuditFields(t *testing.T, db *sql.DB, encounterID, message string) auditFields {
	t.Helper()
	return queryAuditFieldsBySQL(
		t,
		db,
		`SELECT created_at, updated_at, deleted_at FROM encounter_logs WHERE encounter_id = ? AND message = ?`,
		encounterID,
		message,
	)
}

func queryAuditFieldsBySQL(t *testing.T, db *sql.DB, query string, args ...any) auditFields {
	t.Helper()
	var createdAt, updatedAt, deletedAt sql.NullString
	require.NoError(t, db.QueryRow(query, args...).Scan(&createdAt, &updatedAt, &deletedAt))
	return auditFields{
		createdAt: parseAuditTime(t, createdAt),
		updatedAt: parseAuditTime(t, updatedAt),
		deletedAt: parseAuditTime(t, deletedAt),
	}
}

func parseAuditTime(t *testing.T, raw sql.NullString) auditTime {
	t.Helper()
	if !raw.Valid {
		return auditTime{}
	}

	value := strings.TrimSpace(raw.String)
	if value == "" {
		return auditTime{}
	}

	for _, layout := range []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
		time.RFC3339Nano,
	} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return auditTime{Time: parsed, Valid: true}
		}
	}

	require.Failf(t, "parse audit timestamp", "value %q did not match known layouts", value)
	return auditTime{}
}

func queryString(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var v string
	require.NoError(t, db.QueryRow(query, args...).Scan(&v))
	return v
}

func queryColumnType(t *testing.T, db *sql.DB, table, column string) string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk))
		if name == column {
			return columnType
		}
	}
	require.NoError(t, rows.Err())
	require.Failf(t, "column not found", "%s.%s", table, column)
	return ""
}

func queryColumnNames(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	var columns []string
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		require.NoError(t, rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk))
		columns = append(columns, name)
	}
	require.NoError(t, rows.Err())
	return columns
}

type indexColumn struct {
	name string
	desc bool
}

func assertIndexColumns(t *testing.T, db *sql.DB, indexName string, want []indexColumn) {
	t.Helper()

	assert.Equal(t, int64(1), queryInt64(t, db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = 'index' AND name = ?`, indexName))
	assert.Equal(t, want, queryIndexColumns(t, db, indexName))
}

func queryIndexColumns(t *testing.T, db *sql.DB, indexName string) []indexColumn {
	t.Helper()

	rows, err := db.Query(fmt.Sprintf(`PRAGMA index_xinfo(%s)`, indexName))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	var columns []indexColumn
	for rows.Next() {
		var (
			seqno int
			cid   int
			name  sql.NullString
			desc  int
			coll  sql.NullString
			key   int
		)
		require.NoError(t, rows.Scan(&seqno, &cid, &name, &desc, &coll, &key))
		if key != 1 {
			continue
		}
		columns = append(columns, indexColumn{
			name: name.String,
			desc: desc == 1,
		})
	}
	require.NoError(t, rows.Err())
	return columns
}
