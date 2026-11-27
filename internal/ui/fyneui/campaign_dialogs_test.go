package fyneui

import (
	"context"
	"strconv"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsvc "github.com/obalunenko/fallout/internal/app"
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

type renamedCampaignRepo struct {
	*refresherRepo
	campaigns []domain.Campaign
}

func (r *renamedCampaignRepo) ListCampaigns(context.Context) ([]domain.Campaign, error) {
	return r.campaigns, nil
}

func entryTexts(obj fyne.CanvasObject) []string {
	var out []string
	var walk func(fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Entry:
			out = append(out, v.Text)
		case *fyne.Container:
			for _, child := range v.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(v).Objects() {
				walk(child)
			}
		}
	}
	walk(obj)
	return out
}

func TestCampaignEditorOpensWithStoredState(t *testing.T) {
	test.NewTempApp(t).Settings().SetTheme(newPipBoyTheme())
	w := test.NewTempWindow(t, widget.NewLabel("main"))
	w.Resize(fyne.NewSize(1100, 700))
	// The list showed "Old Name"; the assistant renamed it since.
	repo := &renamedCampaignRepo{refresherRepo: &refresherRepo{}, campaigns: []domain.Campaign{{ID: "c1", Name: "Renamed by AI"}}}
	svc := appsvc.NewServiceWithLogfAndTimeout(repo, func(string, ...any) {}, 0)

	openCampaignEditor(t.Context(), w, svc, "c1", func() {})

	var entries []string
	for _, o := range w.Canvas().Overlays().List() {
		entries = append(entries, entryTexts(o)...)
	}
	require.Contains(t, entries, "Renamed by AI", "the editor reads the campaign, not the list's copy")
}
