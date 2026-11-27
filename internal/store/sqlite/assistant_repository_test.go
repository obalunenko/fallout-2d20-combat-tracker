package sqlite

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssistantConnectionSettingsMissingRowIsNotFound(t *testing.T) {
	store := newTestStore(t)

	got, found, err := store.GetAssistantConnectionSettings(t.Context())
	require.NoError(t, err)

	assert.False(t, found)
	assert.Equal(t, domain.AssistantConnectionSettings{}, got)
}

func TestAssistantConnectionSettingsSaveAndLoadRoundTrip(t *testing.T) {
	store := newTestStore(t)

	first := domain.AssistantConnectionSettings{Enabled: true, Port: 48000, AccessSecret: "first-test-secret"}
	require.NoError(t, store.SaveAssistantConnectionSettings(t.Context(), first))
	assertStoredAssistantSettings(t, store, first)

	second := domain.AssistantConnectionSettings{Enabled: false, Port: domain.DefaultAssistantPort, AccessSecret: "second-test-secret"}
	require.NoError(t, store.SaveAssistantConnectionSettings(t.Context(), second))
	assertStoredAssistantSettings(t, store, second)

	assert.Equal(t, int64(1), queryInt64(t, store.db, `SELECT COUNT(*) FROM assistant_connection_settings`))
	fields := queryAuditFieldsBySQL(t, store.db, `SELECT created_at, updated_at, NULL FROM assistant_connection_settings WHERE id = 1`)
	assert.True(t, fields.createdAt.Valid)
	assert.True(t, fields.updatedAt.Valid)
	assert.False(t, fields.updatedAt.Time.Before(fields.createdAt.Time))
}

func TestAssistantConnectionSettingsSaveRejectsInvalidSettings(t *testing.T) {
	store := newTestStore(t)
	const secret = "rejected-test-secret"

	err := store.SaveAssistantConnectionSettings(t.Context(), domain.AssistantConnectionSettings{Port: 80, AccessSecret: secret})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), secret)

	_, found, err := store.GetAssistantConnectionSettings(t.Context())
	require.NoError(t, err)
	assert.False(t, found)
}

func TestAssistantConnectionSettingsTableRejectsInvalidCheckValues(t *testing.T) {
	store := newTestStore(t)
	const insert = `INSERT INTO assistant_connection_settings (id, enabled, port, access_secret) VALUES (?, ?, ?, ?)`

	tests := []struct {
		name    string
		id      int
		enabled int
		port    int
		secret  string
	}{
		{name: "second row", id: 2, enabled: 0, port: domain.DefaultAssistantPort, secret: "check-secret"},
		{name: "enabled is not a bool", id: 1, enabled: 2, port: domain.DefaultAssistantPort, secret: "check-secret"},
		{name: "privileged port", id: 1, enabled: 0, port: 1023, secret: "check-secret"},
		{name: "port above range", id: 1, enabled: 0, port: 65536, secret: "check-secret"},
		{name: "blank secret", id: 1, enabled: 0, port: domain.DefaultAssistantPort, secret: "   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.db.Exec(insert, tc.id, tc.enabled, tc.port, tc.secret)
			assert.ErrorContains(t, err, "CHECK constraint failed")
		})
	}
}

func TestAssistantActivityKeepsNewestThousandEntries(t *testing.T) {
	store := newTestStore(t)
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

	const total = 1001
	for i := range total {
		require.NoError(t, store.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{
			ID:         fmt.Sprintf("activity-%04d", i),
			OccurredAt: base.Add(time.Duration(i) * time.Millisecond),
			Operation:  "apply_damage",
			Outcome:    domain.OutcomeDone,
		}))
	}

	assert.Equal(t, int64(1000), queryInt64(t, store.db, `SELECT COUNT(*) FROM assistant_activity`))
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM assistant_activity WHERE id = ?`, "activity-0000"))

	entries, err := store.ListAssistantActivity(t.Context(), 2*total)
	require.NoError(t, err)
	require.Len(t, entries, 1000)
	assert.Equal(t, "activity-1000", entries[0].ID)
	assert.Equal(t, "activity-0001", entries[len(entries)-1].ID)
	for i := 1; i < len(entries); i++ {
		assert.Truef(t, entries[i-1].OccurredAt.After(entries[i].OccurredAt), "entry %d is not older than entry %d", i, i-1)
	}
}

func TestAssistantActivityListsNewestFirstWithinLimit(t *testing.T) {
	store := newTestStore(t)
	sameInstant := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)

	for _, id := range []string{"tie-1", "tie-2", "tie-3"} {
		require.NoError(t, store.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{
			ID:         id,
			OccurredAt: sameInstant,
			Operation:  "next_turn",
			Outcome:    domain.OutcomeDone,
		}))
	}

	entries, err := store.ListAssistantActivity(t.Context(), 2)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "tie-3", entries[0].ID)
	assert.Equal(t, "tie-2", entries[1].ID)

	_, err = store.ListAssistantActivity(t.Context(), 0)
	require.Error(t, err)
}

func TestAssistantActivityRoundTripsFieldsAndFillsDefaults(t *testing.T) {
	store := newTestStore(t)
	occurredAt := time.Date(2020, 1, 2, 9, 30, 15, 123_456_789, time.FixedZone("UTC+2", 2*60*60))

	full := domain.AssistantActivityEntry{
		ID:          "activity-full",
		OccurredAt:  occurredAt,
		Operation:   "apply_damage",
		EntityKind:  domain.EntityCombatant,
		EntityID:    "raider-2",
		EntityLabel: "Raider #2",
		Outcome:     domain.OutcomeDone,
		Detail:      "7 physical -> torso, applied 4",
	}
	require.NoError(t, store.AppendAssistantActivity(t.Context(), full))

	before := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, store.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{
		Operation: "connection",
		Outcome:   domain.OutcomeRefused,
		Detail:    "bad_secret",
	}))
	after := time.Now().UTC()

	entries, err := store.ListAssistantActivity(t.Context(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	defaulted := entries[0]
	_, err = uuid.Parse(defaulted.ID)
	require.NoError(t, err)
	assert.False(t, defaulted.OccurredAt.Before(before))
	assert.False(t, defaulted.OccurredAt.After(after))
	assert.Equal(t, "connection", defaulted.Operation)
	assert.Empty(t, defaulted.EntityKind)
	assert.Empty(t, defaulted.EntityID)
	assert.Equal(t, domain.OutcomeRefused, defaulted.Outcome)
	assert.Equal(t, int64(1), queryInt64(t, store.db,
		`SELECT COUNT(*) FROM assistant_activity WHERE id = ? AND entity_kind IS NULL AND entity_id IS NULL`, defaulted.ID))

	want := full
	want.OccurredAt = occurredAt.UTC().Truncate(time.Millisecond)
	assert.Equal(t, want, entries[1])
	assert.Equal(t, time.UTC, entries[1].OccurredAt.Location())
}

func TestAssistantActivityClearRemovesAllEntries(t *testing.T) {
	store := newTestStore(t)
	for range 3 {
		require.NoError(t, store.AppendAssistantActivity(t.Context(), domain.AssistantActivityEntry{
			Operation: "add_combatant",
			Outcome:   domain.OutcomeDone,
		}))
	}

	require.NoError(t, store.ClearAssistantActivity(t.Context()))

	entries, err := store.ListAssistantActivity(t.Context(), 10)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM assistant_activity`))
	require.NoError(t, store.ClearAssistantActivity(t.Context()))
}

func TestAssistantActivityRejectsInvalidCheckValues(t *testing.T) {
	store := newTestStore(t)

	tests := []struct {
		name  string
		entry domain.AssistantActivityEntry
	}{
		{
			name:  "invalid outcome",
			entry: domain.AssistantActivityEntry{Operation: "apply_damage", Outcome: "exploded"},
		},
		{
			name:  "invalid entity kind",
			entry: domain.AssistantActivityEntry{Operation: "apply_damage", EntityKind: "vault", Outcome: domain.OutcomeDone},
		},
		{
			name:  "blank operation",
			entry: domain.AssistantActivityEntry{Operation: "   ", Outcome: domain.OutcomeDone},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := store.AppendAssistantActivity(t.Context(), tc.entry)
			assert.ErrorContains(t, err, "CHECK constraint failed")
		})
	}
	assert.Equal(t, int64(0), queryInt64(t, store.db, `SELECT COUNT(*) FROM assistant_activity`))
}

func TestEncounterLogSourceRoundTripsAndDefaultsToDesktop(t *testing.T) {
	store := newTestStore(t)
	require.NoError(t, store.Save(t.Context(), &domain.Encounter{
		ID:         "enc-source",
		Name:       "Source",
		Round:      1,
		TurnIndex:  0,
		Combatants: []domain.Combatant{{ID: "c-source", Name: "Scout", Side: domain.SideParty, Initiative: 10, Active: true}},
	}))

	// A row written without a source, as every row was before migration 00044.
	_, err := store.db.Exec(
		`INSERT INTO encounter_logs (id, encounter_id, round, message, created_at) VALUES (?, ?, ?, ?, ?)`,
		"legacy-log", "enc-source", 1, "Legacy entry", "2026-01-01 00:00:00.000",
	)
	require.NoError(t, err)
	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-source", 1, "Unsourced entry", ""))
	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-source", 2, "Desktop entry", domain.SourceDesktop))
	require.NoError(t, store.AppendEncounterLog(t.Context(), "enc-source", 2, "Assistant entry", domain.SourceAssistant))

	logs, err := store.ListEncounterLogs(t.Context(), "enc-source")
	require.NoError(t, err)

	got := make(map[string]domain.OperationSource, len(logs))
	for _, l := range logs {
		got[l.Message] = l.Source
	}
	assert.Equal(t, map[string]domain.OperationSource{
		"Legacy entry":    domain.SourceDesktop,
		"Unsourced entry": domain.SourceDesktop,
		"Desktop entry":   domain.SourceDesktop,
		"Assistant entry": domain.SourceAssistant,
	}, got)
	assert.Equal(t, "desktop", queryString(t, store.db, `SELECT source FROM encounter_logs WHERE message = ?`, "Unsourced entry"))

	err = store.AppendEncounterLog(t.Context(), "enc-source", 2, "Robot entry", "robot")
	assert.ErrorContains(t, err, "CHECK constraint failed")
}

func assertStoredAssistantSettings(t *testing.T, store *EncounterStore, want domain.AssistantConnectionSettings) {
	t.Helper()

	got, found, err := store.GetAssistantConnectionSettings(t.Context())
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, want.Enabled, got.Enabled)
	assert.Equal(t, want.Port, got.Port)
	// Compared without formatting so a failure never prints the secret.
	assert.True(t, got.AccessSecret == want.AccessSecret, "access secret did not round-trip")
}
