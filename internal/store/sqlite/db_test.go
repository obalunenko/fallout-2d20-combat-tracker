package sqlite

import (
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
