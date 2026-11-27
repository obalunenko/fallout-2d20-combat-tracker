package sqlite

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDBPathUsesEnvironmentOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom", "tracker.db")
	t.Setenv(dbPathEnvKey, want)

	got, err := ResolveDBPath()
	require.NoError(t, err)

	assert.Equal(t, want, got)
	assert.DirExists(t, filepath.Dir(got))
}

func TestResolveDBPathFallsBackToDefaultWhenEnvironmentIsEmpty(t *testing.T) {
	t.Setenv(dbPathEnvKey, "")
	setUserConfigDirEnv(t, t.TempDir())

	want, err := DefaultDBPath()
	require.NoError(t, err)

	got, err := ResolveDBPath()
	require.NoError(t, err)

	assert.Equal(t, want, got)
	assert.True(t, strings.HasSuffix(got, filepath.Join("fallout-tracker", "tracker.db")))
}

func setUserConfigDirEnv(t *testing.T, dir string) {
	t.Helper()

	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
}

func TestSQLiteDSNEnablesForeignKeysAndBusyTimeout(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name   string
		dbPath string
	}{
		{name: "plain path", dbPath: filepath.Join(dir, "plain.db")},
		{name: "file URI", dbPath: "file:" + filepath.Join(dir, "uri.db") + "?mode=rwc"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite", sqliteDSN(tc.dbPath))
			require.NoError(t, err)
			t.Cleanup(func() {
				assert.NoError(t, db.Close())
			})

			assert.Equal(t, int64(1), queryInt64(t, db, `PRAGMA foreign_keys`))
			assert.Equal(t, int64(5000), queryInt64(t, db, `PRAGMA busy_timeout`))
		})
	}
}
