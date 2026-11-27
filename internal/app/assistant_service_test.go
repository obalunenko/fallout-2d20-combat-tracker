package app

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAssistantSettingsDefaultsAndLazySecret(t *testing.T) {
	svc := newSQLiteService(t)

	first, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)
	require.False(t, first.Enabled, "the connection is off by default (FR-003)")
	require.Equal(t, domain.DefaultAssistantPort, first.Port)
	require.NotEmpty(t, first.AccessSecret)

	second, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)
	require.Equal(t, first.AccessSecret, second.AccessSecret, "the secret is generated once")

	regenerated, err := svc.RegenerateAccessSecret(t.Context())
	require.NoError(t, err)
	require.NotEqual(t, first.AccessSecret, regenerated.AccessSecret)
	require.Equal(t, first.Port, regenerated.Port)

	stored, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)
	require.Equal(t, regenerated.AccessSecret, stored.AccessSecret)
}

func TestSaveAssistantSettingsValidates(t *testing.T) {
	svc := newSQLiteService(t)
	settings, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)

	settings.Port = 80
	require.Error(t, svc.SaveAssistantSettings(t.Context(), settings))

	settings.Port = 50000
	settings.Enabled = true
	require.NoError(t, svc.SaveAssistantSettings(t.Context(), settings))
	stored, err := svc.GetAssistantSettings(t.Context())
	require.NoError(t, err)
	require.True(t, stored.Enabled)
	require.Equal(t, 50000, stored.Port)
}

func TestAssistantActivityListAndClear(t *testing.T) {
	svc := newSQLiteService(t)
	for i := range 3 {
		svc.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{
			Operation: fmt.Sprintf("op-%d", i), Outcome: domain.OutcomeDone,
		})
	}
	entries, err := svc.ListAssistantActivity(t.Context(), 2)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	_, err = svc.ListAssistantActivity(t.Context(), 0)
	require.Error(t, err)
	_, err = svc.ListAssistantActivity(t.Context(), 201)
	require.Error(t, err)

	require.NoError(t, svc.ClearAssistantActivity(t.Context()))
	entries, err = svc.ListAssistantActivity(t.Context(), 10)
	require.NoError(t, err)
	require.Empty(t, entries)
}

type activityFailingRepo struct{ *logFailingRepo }

func (activityFailingRepo) AppendAssistantActivity(context.Context, domain.AssistantActivityEntry) error {
	return errors.New("disk full")
}

func TestAssistantActivityFailureIsNonCritical(t *testing.T) {
	var logged []string
	svc := NewServiceWithLogf(activityFailingRepo{&logFailingRepo{}}, func(format string, args ...any) {
		logged = append(logged, fmt.Sprintf(format, args...))
	})

	svc.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{Operation: "apply_damage", Outcome: domain.OutcomeDone})

	require.Equal(t, uint64(1), svc.sideEffectFailureCount("audit."+sideEffectNameAppendAssistantActivity))
	require.Len(t, logged, 1)
}
