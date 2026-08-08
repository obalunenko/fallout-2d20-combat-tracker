package fyneui

import (
	"strconv"
	"testing"

	"fyne.io/fyne/v2/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/obalunenko/fallout/internal/domain"
)

func TestCampaignPlayerInputRowPrefillAndResetDetails(t *testing.T) {
	test.NewTempApp(t)
	row := newCampaignPlayerInputRow(func(*campaignPlayerInputRow) {})
	player := domain.NewCampaignPlayer{
		PlayerName: "June",
		Notes:      "  exact\nnotes  ",
		Special: domain.SpecialValues{
			Strength: 2, Perception: 3, Endurance: 4, Charisma: 5,
			Intelligence: 6, Agility: 7, Luck: 8,
		},
		Character: domain.Combatant{
			Name: "Vault Dweller", Level: 4, Initiative: 11, HP: 9, MaxHP: 12, Defense: 2,
			ResistEnergyTorso: 5, ImmunePoison: true,
		},
		Inactive: true,
	}

	populateCampaignPlayerInputRow(row, player)

	assert.Equal(t, player.Notes, row.notes.Text)
	for _, attribute := range domain.SpecialAttributes() {
		assert.Equal(t, strconv.Itoa(player.Special.Value(attribute)), row.special[attribute].Text)
	}
	assert.False(t, row.active.Checked)
	assert.False(t, row.details.Visible())
	collected, err := collectCampaignPlayersFromRows([]*campaignPlayerInputRow{row})
	require.NoError(t, err)
	require.Len(t, collected, 1)
	assert.Equal(t, player.Notes, collected[0].Notes)
	assert.Equal(t, player.Special, collected[0].Special)

	resetCampaignPlayerInputRow(row)
	assert.Empty(t, row.notes.Text)
	assert.Equal(t, domain.DefaultSpecialValues().Luck, mustEntryInt(t, row.special[domain.SpecialLuck].Text))
	assert.True(t, row.active.Checked)
}

func mustEntryInt(t *testing.T, value string) int {
	t.Helper()
	parsed, err := strconv.Atoi(value)
	require.NoError(t, err)
	return parsed
}
