package sqlite

import (
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncounterStoreAppendAndListLogs(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-1",
		Name:       "Alpha",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c1", Name: "A", Side: domain.SideParty, Initiative: 10, Active: true}},
	}))

	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-1", 1, "Encounter created"))
	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-1", 2, "Turn advanced -> A"))

	logs, err := store.ListEncounterLogs(t.Context(), "enc-1")
	require.NoError(t, err)
	require.Len(t, logs, 2)
	assert.Equal(t, 2, logs[0].Round)
	assert.Contains(t, logs[0].Message, "Turn advanced")
	assert.False(t, logs[0].CreatedAt.IsZero())
	assert.Equal(t, 1, logs[1].Round)
}

func TestEncounterStoreMaintainsEncounterLogAuditFields(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-log-audit",
		Name:       "Log Audit",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c-log-audit", Name: "Scout", Side: domain.SideParty, Initiative: 10, HP: 5, MaxHP: 5, Active: true}},
	}))

	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-log-audit", 1, "Audit log entry"))
	fields := queryEncounterLogAuditFields(t, store.db, "enc-log-audit", "Audit log entry")

	assert.True(t, fields.createdAt.Valid)
	assert.True(t, fields.updatedAt.Valid)
	assert.False(t, fields.deletedAt.Valid)
	assert.False(t, fields.updatedAt.Time.Before(fields.createdAt.Time))
}
