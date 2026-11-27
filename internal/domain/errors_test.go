package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorKindsKeepTheMessage(t *testing.T) {
	base := errors.New("strength must be at least 1")

	invalid := InvalidInput(base)
	require.EqualError(t, invalid, "strength must be at least 1", "the message is unchanged")
	require.ErrorIs(t, invalid, ErrInvalidInput)
	require.ErrorIs(t, invalid, base, "the original error stays in the chain")
	require.NotErrorIs(t, invalid, ErrNotFound)

	missing := NotFound(fmt.Errorf("combatant %q not found", "x"))
	require.ErrorIs(t, missing, ErrNotFound)
	require.NotErrorIs(t, missing, ErrInvalidInput)

	require.NoError(t, InvalidInput(nil))
	require.NoError(t, NotFound(nil))
	require.Same(t, invalid, InvalidInput(invalid), "already marked errors are returned as is")
	require.Same(t, missing, InvalidInput(missing), "a not-found error stays not-found")
}

func TestNotFoundSentinelsAreNotFound(t *testing.T) {
	require.ErrorIs(t, ErrEncounterNotFound, ErrNotFound)
	require.ErrorIs(t, ErrCampaignNotFound, ErrNotFound)
	require.EqualError(t, ErrEncounterNotFound, "encounter not found")
	require.EqualError(t, ErrCampaignNotFound, "campaign not found")
}

func TestEncounterRulesMarkUnknownCombatants(t *testing.T) {
	enc := NewEncounter("e", "Fight", []Combatant{{ID: "a", Name: "A", Side: SideNPC, HP: 5, MaxHP: 5}})
	_, err := enc.ApplyDamage("ghost", DamagePhysical, BodyTorso, 1)
	require.ErrorIs(t, err, ErrNotFound)
	_, err = enc.Heal("ghost", 1)
	require.ErrorIs(t, err, ErrNotFound)
}
