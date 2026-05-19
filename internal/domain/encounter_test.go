package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEncounterSortsAndSetsActive(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{
		{ID: "c3", Name: "Gamma", Initiative: 5, Side: SideNPC},
		{ID: "c1", Name: "Alpha", Initiative: 7, Side: SideParty},
		{ID: "c2", Name: "Beta", Initiative: 7, Side: SideParty},
	})

	assert.Equal(t, 1, e.Round)
	assert.Equal(t, 0, e.TurnIndex)
	require.Len(t, e.Combatants, 3)
	assert.Equal(t, "c1", e.Combatants[0].ID)
	assert.Equal(t, "c2", e.Combatants[1].ID)
	assert.Equal(t, "c3", e.Combatants[2].ID)
	assert.True(t, e.Combatants[0].Active)
	assert.False(t, e.Combatants[1].Active)
	assert.False(t, e.Combatants[2].Active)
}

func TestAdvanceTurnMovesAndIncrementsRound(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{
		{ID: "c1", Name: "Alpha", Initiative: 10},
		{ID: "c2", Name: "Beta", Initiative: 8},
	})

	require.NoError(t, e.AdvanceTurn())
	assert.Equal(t, 1, e.Round)
	assert.Equal(t, 1, e.TurnIndex)
	assert.True(t, e.Combatants[1].Active)

	require.NoError(t, e.AdvanceTurn())
	assert.Equal(t, 2, e.Round)
	assert.Equal(t, 0, e.TurnIndex)
	assert.True(t, e.Combatants[0].Active)
}

func TestAdvanceTurnSkipsDefeated(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{
		{ID: "c1", Name: "Alpha", Initiative: 12},
		{ID: "c2", Name: "Beta", Initiative: 10, Defeated: true},
		{ID: "c3", Name: "Gamma", Initiative: 8},
	})

	require.NoError(t, e.AdvanceTurn())
	assert.Equal(t, 2, e.TurnIndex)
	assert.True(t, e.Combatants[2].Active)
}

func TestAdvanceTurnReturnsErrorWhenNoValidNextCombatant(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, Defeated: true,
	}})

	require.Error(t, e.AdvanceTurn())
}

func TestPartyAPBoundaries(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{ID: "c1", Name: "Alpha", Initiative: 10}})
	e.AddPartyAP(2)
	assert.Equal(t, 2, e.Resources.PartyAP)

	require.Error(t, e.SpendPartyAP(3))
	require.Error(t, e.SpendPartyAP(-1))
	require.NoError(t, e.SpendPartyAP(1))
	assert.Equal(t, 1, e.Resources.PartyAP)

	e.AddPartyAP(-100)
	assert.Equal(t, 0, e.Resources.PartyAP)
}

func TestThreatBoundaries(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{ID: "c1", Name: "Alpha", Initiative: 10}})
	e.AddThreat(2)
	assert.Equal(t, 2, e.Resources.GMThreat)

	require.Error(t, e.SpendThreat(3))
	require.Error(t, e.SpendThreat(-1))
	require.NoError(t, e.SpendThreat(1))
	assert.Equal(t, 1, e.Resources.GMThreat)

	e.AddThreat(-100)
	assert.Equal(t, 0, e.Resources.GMThreat)
}

func TestApplyDamageReducesHPByResistance(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10,
		HP: 10, ResistPhysical: 3,
	}})

	applied, err := e.ApplyDamage("c1", DamagePhysical, 8)
	require.NoError(t, err)
	assert.Equal(t, 5, applied)
	assert.Equal(t, 5, e.Combatants[0].HP)
	assert.False(t, e.Combatants[0].Defeated)
}

func TestApplyDamageRespectsImmunity(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10,
		HP: 10, ResistEnergy: 2, ImmuneEnergy: true,
	}})

	applied, err := e.ApplyDamage("c1", DamageEnergy, 999)
	require.NoError(t, err)
	assert.Equal(t, 0, applied)
	assert.Equal(t, 10, e.Combatants[0].HP)
	assert.False(t, e.Combatants[0].Defeated)
}

func TestApplyDamageMarksDefeatedWhenHPZeroOrLess(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10,
		HP: 4, ResistPoison: 1,
	}})

	applied, err := e.ApplyDamage("c1", DamagePoison, 10)
	require.NoError(t, err)
	assert.Equal(t, 9, applied)
	assert.Equal(t, -5, e.Combatants[0].HP)
	assert.True(t, e.Combatants[0].Defeated)
}

func TestApplyDamageValidation(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: 4,
	}})

	_, err := e.ApplyDamage("", DamagePhysical, 1)
	require.Error(t, err)
	_, err = e.ApplyDamage("c1", DamagePhysical, -1)
	require.Error(t, err)
	_, err = e.ApplyDamage("missing", DamagePhysical, 1)
	require.Error(t, err)
	_, err = e.ApplyDamage("c1", DamageType("invalid"), 1)
	require.Error(t, err)
}

func TestHealIncreasesHP(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: 5,
	}})

	healed, err := e.Heal("c1", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, healed)
	assert.Equal(t, 8, e.Combatants[0].HP)
	assert.False(t, e.Combatants[0].Defeated)
}

func TestHealCanReviveCombatant(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: -2, Defeated: true,
	}})

	healed, err := e.Heal("c1", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, healed)
	assert.Equal(t, 1, e.Combatants[0].HP)
	assert.False(t, e.Combatants[0].Defeated)
}

func TestHealValidation(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: 4,
	}})

	_, err := e.Heal("", 1)
	require.Error(t, err)
	_, err = e.Heal("c1", -1)
	require.Error(t, err)
	_, err = e.Heal("missing", 1)
	require.Error(t, err)
}
