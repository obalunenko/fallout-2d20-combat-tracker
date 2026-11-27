package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration44UpDownUpPreservesEncounterLogs(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migration-44.db")
	db, err := sql.Open("sqlite", sqliteDSN(dbPath))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})

	goose.SetBaseFS(migrationsFS)
	require.NoError(t, goose.SetDialect("sqlite3"))
	require.NoError(t, goose.UpTo(db, "migrations", 43))
	const encounterLogsSchemaSQL = `SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = 'encounter_logs'`
	schemaBefore := queryString(t, db, encounterLogsSchemaSQL)

	_, err = db.Exec(`
		INSERT INTO campaigns (id, name, start_date)
		VALUES ('m44-campaign', 'Migration 44', '2026-01-01 00:00:00');
		INSERT INTO encounters (id, campaign_id, name, round, turn_index)
		VALUES ('m44-encounter', 'm44-campaign', 'Migration 44 Fight', 1, 0);
		INSERT INTO encounter_logs (id, encounter_id, round, message, created_at, updated_at) VALUES
			('m44-log-1', 'm44-encounter', 1, 'Encounter created', '2026-01-01 10:00:00.000', '2026-01-01 10:00:00.000'),
			('m44-log-2', 'm44-encounter', 2, 'Turn advanced', '2026-01-01 10:01:00.000', '2026-01-01 10:01:00.000');
	`)
	require.NoError(t, err)

	require.NoError(t, goose.UpTo(db, "migrations", 44))
	assert.Equal(t, "m44-log-1:desktop,m44-log-2:desktop", encounterLogSources(t, db))
	assertAssistantTablesPresent(t, db, true)

	_, err = db.Exec(`
		INSERT INTO encounter_logs (id, encounter_id, round, message, created_at, source)
		VALUES ('m44-log-3', 'm44-encounter', 3, 'Assistant entry', '2026-01-01 10:02:00.000', 'assistant');
		INSERT INTO assistant_connection_settings (id, enabled, port, access_secret)
		VALUES (1, 1, 48000, 'migration-test-secret');
		INSERT INTO assistant_activity (id, occurred_at, operation, outcome)
		VALUES ('m44-activity', '2026-01-01 10:02:00.000', 'apply_damage', 'done');
	`)
	require.NoError(t, err)

	require.NoError(t, goose.DownTo(db, "migrations", 43))
	version, err := goose.GetDBVersion(db)
	require.NoError(t, err)
	assert.Equal(t, int64(43), version)

	assertAssistantTablesPresent(t, db, false)
	assert.Equal(t, schemaBefore, queryString(t, db, encounterLogsSchemaSQL))
	assert.Equal(t,
		[]string{"id", "encounter_id", "round", "message", "created_at", "updated_at", "deleted_at"},
		queryColumnNames(t, db, "encounter_logs"),
	)
	assert.Equal(t,
		"m44-log-1:1:Encounter created:2026-01-01 10:00:00.000,"+
			"m44-log-2:2:Turn advanced:2026-01-01 10:01:00.000,"+
			"m44-log-3:3:Assistant entry:2026-01-01 10:02:00.000",
		queryString(t, db, `
			SELECT GROUP_CONCAT(id || ':' || round || ':' || message || ':' || created_at, ',')
			FROM (SELECT id, round, message, created_at FROM encounter_logs ORDER BY id)`),
	)
	t.Run("constraints after down", func(t *testing.T) {
		assertEncounterLogsConstraints(t, db)
	})

	require.NoError(t, goose.UpTo(db, "migrations", 44))
	assert.Equal(t, "m44-log-1:desktop,m44-log-2:desktop,m44-log-3:desktop", encounterLogSources(t, db))
	assertAssistantTablesPresent(t, db, true)
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM assistant_connection_settings`))
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM assistant_activity`))
	t.Run("constraints after up", func(t *testing.T) {
		assertEncounterLogsConstraints(t, db)
	})

	_, err = db.Exec(`INSERT INTO encounter_logs (id, encounter_id, round, message, source)
		VALUES ('m44-robot', 'm44-encounter', 1, 'Robot entry', 'robot')`)
	assert.ErrorContains(t, err, "CHECK constraint failed")
}

func encounterLogSources(t *testing.T, db *sql.DB) string {
	t.Helper()
	return queryString(t, db, `
		SELECT GROUP_CONCAT(id || ':' || source, ',')
		FROM (SELECT id, source FROM encounter_logs ORDER BY id)`)
}

func assertAssistantTablesPresent(t *testing.T, db *sql.DB, want bool) {
	t.Helper()

	count := int64(0)
	if want {
		count = 1
	}
	for _, object := range []struct{ kind, name string }{
		{kind: "table", name: "assistant_connection_settings"},
		{kind: "table", name: "assistant_activity"},
		{kind: "index", name: "idx_assistant_activity_occurred"},
	} {
		assert.Equalf(t, count,
			queryInt64(t, db, `SELECT COUNT(*) FROM sqlite_schema WHERE type = ? AND name = ?`, object.kind, object.name),
			"%s %s", object.kind, object.name,
		)
	}
	if want {
		assertIndexColumns(t, db, "idx_assistant_activity_occurred", []indexColumn{{name: "occurred_at", desc: true}})
	}
}

// assertEncounterLogsConstraints checks the foreign key, CHECK constraints and
// index that encounter_logs must keep however migration 00044 last ran.
func assertEncounterLogsConstraints(t *testing.T, db *sql.DB) {
	t.Helper()

	assert.Equal(t, []foreignKey{{table: "encounters", from: "encounter_id", to: "id", onDelete: "CASCADE"}},
		queryForeignKeys(t, db, "encounter_logs"))
	assertIndexColumns(t, db, "idx_encounter_logs_encounter_created", []indexColumn{
		{name: "encounter_id"},
		{name: "created_at", desc: true},
		{name: "id", desc: true},
	})

	const insertLog = `INSERT INTO encounter_logs (id, encounter_id, round, message) VALUES (?, ?, ?, ?)`
	for _, tc := range []struct {
		name        string
		id          string
		encounterID string
		round       int
		message     string
		wantErr     string
	}{
		{name: "blank id", id: " ", encounterID: "m44-encounter", round: 1, message: "Entry", wantErr: "CHECK constraint failed"},
		{name: "round below one", id: "m44-bad-round", encounterID: "m44-encounter", round: 0, message: "Entry", wantErr: "CHECK constraint failed"},
		{name: "blank message", id: "m44-bad-message", encounterID: "m44-encounter", round: 1, message: "  ", wantErr: "CHECK constraint failed"},
		{name: "unknown encounter", id: "m44-bad-encounter", encounterID: "missing-encounter", round: 1, message: "Entry", wantErr: "FOREIGN KEY constraint failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(insertLog, tc.id, tc.encounterID, tc.round, tc.message)
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}

	_, err := db.Exec(`
		INSERT INTO encounters (id, campaign_id, name, round, turn_index)
		VALUES ('m44-cascade', 'm44-campaign', 'Cascade Fight', 1, 0);
		INSERT INTO encounter_logs (id, encounter_id, round, message)
		VALUES ('m44-cascade-log', 'm44-cascade', 1, 'Cascade entry');
		DELETE FROM encounters WHERE id = 'm44-cascade';
	`)
	require.NoError(t, err)
	assert.Equal(t, int64(0), queryInt64(t, db, `SELECT COUNT(*) FROM encounter_logs WHERE id = 'm44-cascade-log'`))
}

type foreignKey struct {
	table    string
	from     string
	to       string
	onDelete string
}

func queryForeignKeys(t *testing.T, db *sql.DB, table string) []foreignKey {
	t.Helper()

	rows, err := db.Query(fmt.Sprintf(`PRAGMA foreign_key_list(%s)`, table))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, rows.Close())
	}()

	var keys []foreignKey
	for rows.Next() {
		var (
			id, seq         int
			fk              foreignKey
			onUpdate, match string
		)
		require.NoError(t, rows.Scan(&id, &seq, &fk.table, &fk.from, &fk.to, &onUpdate, &fk.onDelete, &match))
		keys = append(keys, fk)
	}
	require.NoError(t, rows.Err())
	return keys
}
