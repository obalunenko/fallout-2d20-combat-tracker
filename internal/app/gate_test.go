package app

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/require"
)

// newSQLiteEncounter creates an active encounter with the seeded party member
// and one NPC that has npcHP hit points and no damage resistance.
func newSQLiteEncounter(t *testing.T, svc *Service, npcHP int) *domain.Encounter {
	t.Helper()
	enc, err := svc.CreateEncounter(t.Context(), "", "Gate Fight", []domain.Combatant{
		{ID: "char-1", PlayerCharacterID: "char-1", Name: "Vault Dweller", Side: domain.SideParty, Level: 1, Initiative: 9, HP: 7, MaxHP: 7},
		{Name: "Raider", Side: domain.SideNPC, Level: 1, XP: 10, Initiative: 5, HP: npcHP, MaxHP: npcHP},
	})
	require.NoError(t, err)
	return enc
}

func npcID(t *testing.T, enc *domain.Encounter) string {
	t.Helper()
	for _, c := range enc.Combatants {
		if c.Side == domain.SideNPC {
			return c.ID
		}
	}
	t.Fatal("encounter has no NPC")
	return ""
}

func TestGateSerializesConcurrentWrites(t *testing.T) {
	svc := newSQLiteService(t)
	enc := newSQLiteEncounter(t, svc, 1000)
	raider := npcID(t, enc)

	const hits, turns = 50, 20
	// An explicit deadline replaces the 5 s default, which also bounds the
	// wait for the gate: under -race SQLite is slow enough that 70 queued
	// writers would otherwise correctly report ErrTrackerBusy.
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, hits+turns)
	for range hits {
		wg.Go(func() {
			_, _, err := svc.ApplyDamage(ctx, raider, domain.DamagePhysical, domain.BodyTorso, 1)
			errs <- err
		})
	}
	for range turns {
		wg.Go(func() {
			_, err := svc.AdvanceTurn(ctx)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	got, err := svc.GetEncounterByID(t.Context(), enc.ID)
	require.NoError(t, err)
	require.Equal(t, 1000-hits, requireCombatantByID(t, got, raider).HP, "no damage may be lost")
	// Two combatants: 20 turn advances end on round 11, turn index 0.
	require.Equal(t, 1+turns/2, got.Round)
	require.Equal(t, 0, got.TurnIndex)
}

func TestGateReturnsBusyWhenWriterHoldsIt(t *testing.T) {
	svc := newSQLiteService(t)
	_, op, err := svc.beginWrite(t.Context())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_, err = svc.ActivateCampaign(ctx, "test-campaign")
	require.ErrorIs(t, err, domain.ErrTrackerBusy)

	op.end()
	_, err = svc.ActivateCampaign(t.Context(), "test-campaign")
	require.NoError(t, err)
}

func TestGateWrapperPairsDoNotDeadlock(t *testing.T) {
	svc := newSQLiteService(t)
	call := func(name string, fn func(ctx context.Context) error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		err := fn(ctx)
		require.False(t, errors.Is(err, domain.ErrTrackerBusy), "%s deadlocked on the gate", name)
		require.NoError(t, err, name)
	}

	var enc *domain.Encounter
	call("CreateEncounter", func(ctx context.Context) (err error) {
		enc, err = svc.CreateEncounter(ctx, "", "Pairs", []domain.Combatant{
			{ID: "char-1", PlayerCharacterID: "char-1", Name: "Vault Dweller", Side: domain.SideParty, Level: 1, Initiative: 9, HP: 7, MaxHP: 7},
			{Name: "Raider", Side: domain.SideNPC, Level: 1, Initiative: 5, HP: 10, MaxHP: 10},
		})
		return err
	})
	raider := npcID(t, enc)
	call("UpdateEncounter", func(ctx context.Context) error {
		_, err := svc.UpdateEncounter(ctx, enc.ID, "Pairs Renamed", enc.Combatants)
		return err
	})
	call("ApplyDamage", func(ctx context.Context) error {
		_, _, err := svc.ApplyDamage(ctx, raider, domain.DamagePhysical, domain.BodyTorso, 2)
		return err
	})
	call("Heal", func(ctx context.Context) error {
		_, _, err := svc.Heal(ctx, raider, 1)
		return err
	})
	call("SaveMonsterTemplates", func(ctx context.Context) error {
		_, err := svc.SaveMonsterTemplates(ctx, []domain.Combatant{{Name: "Mirelurk", Level: 2, HP: 12, MaxHP: 12}})
		return err
	})
	players, err := svc.ListCampaignPlayers(t.Context(), "test-campaign")
	require.NoError(t, err)
	call("UpdateCampaign", func(ctx context.Context) error {
		_, err := svc.UpdateCampaign(ctx, "test-campaign", "Renamed", testCampaignStartDate(t), players)
		return err
	})
	call("CreateCampaign", func(ctx context.Context) error {
		_, err := svc.CreateCampaign(ctx, "", "Second", testCampaignStartDate(t), []domain.NewCampaignPlayer{
			{PlayerName: "P2", Character: domain.Combatant{Name: "Nora", Side: domain.SideParty, Level: 1, HP: 5}},
		})
		return err
	})
}
