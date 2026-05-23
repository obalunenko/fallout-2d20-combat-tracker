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
		HP: 10, ResistPhysicalTorso: 2,
	}})

	applied, err := e.ApplyDamage("c1", DamagePhysical, BodyTorso, 8)
	require.NoError(t, err)
	assert.Equal(t, 6, applied)
	assert.Equal(t, 4, e.Combatants[0].HP)
	assert.False(t, e.Combatants[0].Defeated)
}

func TestApplyDamageUsesGlobalOrLocationResistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		damageType DamageType
		location   BodyLocation
		combatant  Combatant
		want       int
	}{
		{
			name:       "physical uses location in body-location mode",
			damageType: DamagePhysical,
			location:   BodyTorso,
			combatant:  Combatant{ResistPhysical: 3, ResistPhysicalTorso: 2},
			want:       8,
		},
		{
			name:       "energy uses location in body-location mode",
			damageType: DamageEnergy,
			location:   BodyLeftArm,
			combatant:  Combatant{ResistEnergy: 2, ResistEnergyLeftArm: 4},
			want:       6,
		},
		{
			name:       "radiation uses location in body-location mode",
			damageType: DamageRadiation,
			location:   BodyRightLeg,
			combatant:  Combatant{ResistRadiation: 1, ResistRadiationRightLeg: 5},
			want:       5,
		},
		{
			name:       "torso-only physical uses torso location and ignores global",
			damageType: DamagePhysical,
			location:   BodyTorso,
			combatant:  Combatant{TorsoOnly: true, ResistPhysical: 3, ResistPhysicalTorso: 2},
			want:       8,
		},
		{
			name:       "torso-only energy uses torso location",
			damageType: DamageEnergy,
			location:   BodyTorso,
			combatant:  Combatant{TorsoOnly: true, ResistEnergyTorso: 4},
			want:       6,
		},
		{
			name:       "poison ignores location resistance",
			damageType: DamagePoison,
			location:   BodyHead,
			combatant:  Combatant{ResistPoison: 4, ResistRadiationHead: 5},
			want:       6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.combatant.ID = "c1"
			tt.combatant.Name = "Alpha"
			tt.combatant.Initiative = 10
			tt.combatant.HP = 20
			tt.combatant.MaxHP = 20
			e := NewEncounter("enc-1", "test", []Combatant{tt.combatant})

			applied, err := e.ApplyDamage("c1", tt.damageType, tt.location, 10)
			require.NoError(t, err)
			assert.Equal(t, tt.want, applied)
			assert.Equal(t, 20-tt.want, e.Combatants[0].HP)
		})
	}
}

func TestApplyDamagePhysicalIgnoresDefense(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10,
		HP: 10, Defense: 5, ResistPhysicalTorso: 0,
	}})

	applied, err := e.ApplyDamage("c1", DamagePhysical, BodyTorso, 8)
	require.NoError(t, err)
	assert.Equal(t, 8, applied)
	assert.Equal(t, 2, e.Combatants[0].HP)
}

func TestApplyDamageRespectsImmunity(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10,
		HP: 10, ImmunePoison: true,
	}})

	applied, err := e.ApplyDamage("c1", DamagePoison, BodyHead, 999)
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

	applied, err := e.ApplyDamage("c1", DamagePoison, BodyLeftLeg, 10)
	require.NoError(t, err)
	assert.Equal(t, 9, applied)
	assert.Equal(t, 0, e.Combatants[0].HP)
	assert.True(t, e.Combatants[0].Defeated)
}

func TestApplyDamageValidation(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: 4,
	}})

	_, err := e.ApplyDamage("", DamagePhysical, BodyTorso, 1)
	require.Error(t, err)
	_, err = e.ApplyDamage("c1", DamagePhysical, BodyTorso, -1)
	require.Error(t, err)
	_, err = e.ApplyDamage("missing", DamagePhysical, BodyTorso, 1)
	require.Error(t, err)
	_, err = e.ApplyDamage("c1", DamageType("invalid"), BodyTorso, 1)
	require.Error(t, err)
	_, err = e.ApplyDamage("c1", DamagePhysical, BodyLocation("invalid"), 1)
	require.Error(t, err)
}

func TestHealIncreasesHP(t *testing.T) {
	t.Parallel()

	e := NewEncounter("enc-1", "test", []Combatant{{
		ID: "c1", Name: "Alpha", Initiative: 10, HP: 5,
		MaxHP: 10,
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
		ID: "c1", Name: "Alpha", Initiative: 10, HP: -2, MaxHP: 6, Defeated: true,
	}})

	healed, err := e.Heal("c1", 3)
	require.NoError(t, err)
	assert.Equal(t, 3, healed)
	assert.Equal(t, 3, e.Combatants[0].HP)
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

func TestEvaluateEncounterDifficultyUnknownWhenSidesMissing(t *testing.T) {
	t.Parallel()

	onlyParty := EvaluateEncounterDifficulty([]Combatant{
		{Name: "Hero", Side: SideParty, Level: 2},
	})
	assert.Equal(t, EncounterDifficultyUnknown, onlyParty.Label)

	onlyNPC := EvaluateEncounterDifficulty([]Combatant{
		{Name: "Raider", Side: SideNPC, Level: 2, XP: 30},
	})
	assert.Equal(t, EncounterDifficultyUnknown, onlyNPC.Label)
}

func TestEvaluateEncounterDifficultyUsesPartyAndEnemyStats(t *testing.T) {
	t.Parallel()

	metrics := EvaluateEncounterDifficulty([]Combatant{
		{Name: "P1", Side: SideParty, Level: 2},
		{Name: "P2", Side: SideParty, Level: 2},
		{Name: "Raider A", Side: SideNPC, Level: 2, XP: 60},
		{Name: "Raider B", Side: SideNPC, Level: 2, XP: 60},
	})

	assert.Equal(t, EncounterDifficultyHard, metrics.Label)
	assert.Equal(t, 2, metrics.PartyCount)
	assert.Equal(t, 2.0, metrics.PartyAvgLevel)
	assert.Equal(t, 60, metrics.PartyXPBudget)
	assert.Equal(t, 2, metrics.EnemyCount)
	assert.Equal(t, 2.0, metrics.EnemyAvgLevel)
	assert.Equal(t, 120, metrics.EnemyTotalXP)
	assert.Equal(t, 2.0, metrics.Score)
}
