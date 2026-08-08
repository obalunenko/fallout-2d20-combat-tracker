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

func TestFormatDifficultyPreviewShowsRequestedLabelsAndMetrics(t *testing.T) {
	tests := []domain.EncounterDifficulty{
		domain.EncounterDifficultyTrivial,
		domain.EncounterDifficultySimple,
		domain.EncounterDifficultyAverage,
		domain.EncounterDifficultyHard,
		domain.EncounterDifficultyDeadly,
	}

	for _, label := range tests {
		t.Run(string(label), func(t *testing.T) {
			got := formatDifficultyPreview(domain.EncounterDifficultyMetrics{
				Label:          label,
				PartyCount:     2,
				AveragePCLevel: 3,
				TotalMonsterXP: 70,
				XPBaseline:     35,
				EncounterLevel: 2,
				Difference:     -1,
			})

			assert.Contains(t, got, "Difficulty: "+string(label))
			assert.Contains(t, got, "party: 2")
			assert.Contains(t, got, "avg PC lvl 3")
			assert.Contains(t, got, "monster XP: 70")
			assert.Contains(t, got, "baseline 35.0")
			assert.Contains(t, got, "encounter lvl 2")
			assert.NotContains(t, got, "Easy")
			assert.NotContains(t, got, "Normal")
			assert.NotContains(t, got, "xp ratio")
		})
	}
}

func TestFormatDifficultyPreviewShowsUnknownReason(t *testing.T) {
	got := formatDifficultyPreview(domain.EncounterDifficultyMetrics{
		Label:             domain.EncounterDifficultyUnknown,
		UnavailableReason: "add at least one party member",
	})

	assert.Equal(t, "Difficulty: Unknown (add at least one party member)", got)
}

func TestFormatEncounterDifficultySummaryUsesNewMetrics(t *testing.T) {
	got := formatEncounterDifficultySummary(domain.EncounterSummary{
		Difficulty:           string(domain.EncounterDifficultySimple),
		PartyCount:           2,
		AveragePCLevel:       3,
		TotalMonsterXP:       70,
		XPBaseline:           35,
		EncounterLevel:       2,
		DifficultyDifference: -1,
	})

	assert.Contains(t, got, "Simple")
	assert.Contains(t, got, "party:2")
	assert.Contains(t, got, "avg PC lvl:3")
	assert.Contains(t, got, "monster XP:70")
	assert.Contains(t, got, "baseline:35.0")
	assert.Contains(t, got, "encounter lvl:2")
	assert.Contains(t, got, "diff:-1")
	assert.NotContains(t, got, "Easy")
	assert.NotContains(t, got, "Normal")
	assert.NotContains(t, got, "budget")
}
