package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterLogsArePersistentAndIncludeRound(t *testing.T) {
	svc := newSQLiteService(t)
	created, err := svc.CreateEncounter(t.Context(), "enc-1", "Alpha", []domain.Combatant{
		{ID: "p1", Name: "Player", Initiative: 10, Side: domain.SideParty, HP: 10},
		{ID: "n1", Name: "Raider", Initiative: 8, Side: domain.SideNPC, HP: 6},
	})
	require.NoError(t, err)

	_, err = svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	_, _, err = svc.ApplyDamage(t.Context(), "n1", domain.DamagePhysical, domain.BodyLeftArm, 3)
	require.NoError(t, err)
	_, _, err = svc.Heal(t.Context(), "n1", 2)
	require.NoError(t, err)

	logs, err := svc.ListEncounterLogs(t.Context(), created.ID)
	require.NoError(t, err)
	require.NotEmpty(t, logs)
	assert.Contains(t, logs[0].Message, "Heal")
	assert.False(t, logs[0].CreatedAt.IsZero())

	hasTurnAdvanced := false
	for _, l := range logs {
		assert.GreaterOrEqual(t, l.Round, 1)
		if strings.Contains(l.Message, "Turn advanced") {
			hasTurnAdvanced = true
		}
	}
	assert.True(t, hasTurnAdvanced, "expected at least one turn advance log entry")
}

func TestAddPartyAPSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
			},
			Resources: domain.Resources{
				PartyAP:  0,
				GMThreat: 0,
			},
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, err := svc.AddPartyAP(t.Context(), 2)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 2, updated.Resources.PartyAP)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 2, persisted.Resources.PartyAP, "state change is persisted even when log write fails")
}

func TestAddPartyAPLogsWhenLogWriteFails(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
			},
			Resources: domain.Resources{
				PartyAP:  0,
				GMThreat: 0,
			},
		},
		appendErr: errors.New("append log failed"),
	}

	logEntries := make([]string, 0, 1)
	svc := NewServiceWithLogf(repo, func(format string, args ...any) {
		logEntries = append(logEntries, fmt.Sprintf(format, args...))
	})

	updated, err := svc.AddPartyAP(t.Context(), 1)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotEmpty(t, logEntries)
	assert.Contains(t, logEntries[0], "non-critical side effect failed")
	assert.Contains(t, logEntries[0], "side_effect=audit.append_encounter_log")
	assert.Contains(t, logEntries[0], "failures=1")
	assert.Contains(t, logEntries[0], "encounter_id=enc-1")
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("audit.append_encounter_log"))
}

func TestRunNonCriticalSideEffectLogsErrorAndDoesNotBubbleUp(t *testing.T) {
	logEntries := make([]string, 0, 1)
	svc := NewServiceWithLogf(nil, func(format string, args ...any) {
		logEntries = append(logEntries, fmt.Sprintf(format, args...))
	})

	svc.runNonCriticalSideEffect(sideEffectCategoryTelemetry, "demo_side_effect", func() error {
		return errors.New("boom")
	})

	require.Len(t, logEntries, 1)
	assert.Contains(t, logEntries[0], "non-critical side effect failed")
	assert.Contains(t, logEntries[0], "side_effect=telemetry.demo_side_effect")
	assert.Contains(t, logEntries[0], "failures=1")
	assert.Contains(t, logEntries[0], "boom")
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("telemetry.demo_side_effect"))
}

func TestRunNonCriticalSideEffectCountsFailuresPerType(t *testing.T) {
	svc := NewServiceWithLogf(nil, func(string, ...any) {})
	for range 2 {
		svc.runNonCriticalSideEffect(sideEffectCategoryNotifications, "dispatch_update", func() error {
			return errors.New("network timeout")
		})
	}
	svc.runNonCriticalSideEffect(sideEffectCategoryAudit, "append_encounter_log", func() error {
		return errors.New("write failed")
	})

	assert.Equal(t, uint64(2), svc.sideEffectFailureCount("notifications.dispatch_update"))
	assert.Equal(t, uint64(1), svc.sideEffectFailureCount("audit.append_encounter_log"))
}

func TestAdvanceTurnSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "c1", Name: "One", Initiative: 10, Active: true},
				{ID: "c2", Name: "Two", Initiative: 8, Active: false},
			},
			TurnIndex: 0,
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, err := svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 1, updated.TurnIndex)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 1, persisted.TurnIndex)
}

func TestApplyDamageSucceedsWhenLogWriteFailsAndStateIsSaved(t *testing.T) {
	repo := &logFailingRepo{
		encounter: &domain.Encounter{
			ID:    "enc-1",
			Name:  "Alpha",
			Round: 1,
			Combatants: []domain.Combatant{
				{ID: "n1", Name: "Raider", Initiative: 8, HP: 10, MaxHP: 10, Side: domain.SideNPC},
			},
			TurnIndex: 0,
		},
		appendErr: errors.New("append log failed"),
	}
	svc := NewService(repo)

	updated, applied, err := svc.ApplyDamage(t.Context(), "n1", domain.DamagePoison, domain.BodyHead, 3)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, 3, applied)
	assert.Equal(t, 7, updated.Combatants[0].HP)
	assert.Equal(t, 1, repo.saveCalls)
	assert.Equal(t, 1, repo.appendCalls)

	persisted, getErr := repo.Get(t.Context())
	require.NoError(t, getErr)
	assert.Equal(t, 7, persisted.Combatants[0].HP)
}
