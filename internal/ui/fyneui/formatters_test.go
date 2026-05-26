package fyneui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestFormatBodyResistanceTableAlignsLocationRows(t *testing.T) {
	combatant := domain.Combatant{}
	require.NoError(t, combatant.SetLocationResistance(domain.DamagePhysical, domain.BodyHead, 4))
	require.NoError(t, combatant.SetLocationResistance(domain.DamageEnergy, domain.BodyTorso, 15))
	require.NoError(t, combatant.SetLocationResistance(domain.DamageRadiation, domain.BodyRightLeg, 9))

	got := formatBodyResistanceTable(combatant)

	want := strings.Join([]string{
		"Body Damage Resistance",
		"Location  | Physical | Energy | Radiation",
		strings.Repeat("-", 41),
		"Head      |        4 |      0 |         0",
		"Torso     |        0 |     15 |         0",
		"Left Arm  |        0 |      0 |         0",
		"Right Arm |        0 |      0 |         0",
		"Left Leg  |        0 |      0 |         0",
		"Right Leg |        0 |      0 |         9",
	}, "\n")
	assert.Equal(t, want, got)
}

func TestFormatActiveTargetDetailsUsesTables(t *testing.T) {
	enc := testEncounter()

	got := formatActiveTargetDetails(enc, enc.Combatants[0])

	assert.Contains(t, got, "Participant Details")
	assert.Contains(t, got, "Field")
	assert.Contains(t, got, "Value")
	assert.Contains(t, got, "Name")
	assert.Contains(t, got, "Alpha")
	assert.Contains(t, got, "Status")
	assert.Contains(t, got, "Active")
	assert.Contains(t, got, "Body Damage Resistance")
	assert.Contains(t, got, "Location")
	assert.Contains(t, got, "Physical")
}

func TestFormatExpandedCombatantDetailsMatchesActiveTargetTable(t *testing.T) {
	enc := testEncounter()

	got := formatExpandedCombatantDetails(enc, enc.Combatants[0])

	assert.Equal(t, formatActiveTargetDetails(enc, enc.Combatants[0]), got)
	assert.Contains(t, got, "Field")
	assert.Contains(t, got, "Body Damage Resistance")
}

func TestFormatCombatantStatusIncludesHealthAttention(t *testing.T) {
	combatant := domain.Combatant{HP: 5, MaxHP: 10}
	assert.Equal(t, "Ready, Wounded", formatCombatantStatus(combatant))

	combatant.Active = true
	combatant.HP = 2
	assert.Equal(t, "Active, Critical", formatCombatantStatus(combatant))

	combatant.HP = 0
	combatant.Defeated = true
	assert.Equal(t, "Defeated", formatCombatantStatus(combatant))
}

func TestFormatCombatantLineAddsHealthAttentionBadge(t *testing.T) {
	enc := testEncounter()
	enc.Combatants[0].HP = 2
	enc.Combatants[0].MaxHP = 8

	got := formatCombatantLine(enc, enc.Combatants[0])

	assert.Contains(t, got, "[CRITICAL]")
}
