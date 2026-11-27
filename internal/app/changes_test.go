package app

import (
	"sync"
	"testing"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

type eventRecorder struct {
	mu     sync.Mutex
	events []ChangeEvent
}

func (r *eventRecorder) record(ev ChangeEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
}

func (r *eventRecorder) take() []ChangeEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.events
	r.events = nil
	return out
}

func TestChangeEventsPublishedOncePerSuccessfulMutation(t *testing.T) {
	svc := newSQLiteService(t)
	enc := newSQLiteEncounter(t, svc, 20)
	rec := &eventRecorder{}
	unsubscribe := svc.SubscribeChanges(rec.record)

	_, _, err := svc.ApplyDamage(t.Context(), npcID(t, enc), domain.DamagePhysical, domain.BodyTorso, 3)
	require.NoError(t, err)
	events := rec.take()
	require.Len(t, events, 1)
	require.Equal(t, domain.SourceDesktop, events[0].Source)
	require.Equal(t, enc.ID, events[0].EncounterID)
	require.Contains(t, events[0].Kinds, domain.EntityEncounter)

	// Reads publish nothing.
	_, err = svc.ListEncounters(t.Context())
	require.NoError(t, err)
	require.Empty(t, rec.take())

	// A failed mutation publishes nothing.
	_, err = svc.SpendThreat(t.Context(), 99)
	require.Error(t, err)
	require.Empty(t, rec.take())

	// The source comes from the context.
	_, err = svc.AddThreat(WithSource(t.Context(), domain.SourceAssistant), 1)
	require.NoError(t, err)
	events = rec.take()
	require.Len(t, events, 1)
	require.Equal(t, domain.SourceAssistant, events[0].Source)

	unsubscribe()
	_, err = svc.AddThreat(t.Context(), 1)
	require.NoError(t, err)
	require.Empty(t, rec.take())
}

func TestChangeEventPublishedAfterGateRelease(t *testing.T) {
	svc := newSQLiteService(t)
	newSQLiteEncounter(t, svc, 20)
	var readErr error
	called := false
	svc.SubscribeChanges(func(ChangeEvent) {
		called = true
		// A subscriber may call back into the Service: the gate is released.
		_, readErr = svc.ListEncounters(t.Context())
	})

	_, err := svc.AdvanceTurn(t.Context())
	require.NoError(t, err)
	require.True(t, called)
	require.NoError(t, readErr)
}
