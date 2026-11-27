package app

import (
	"testing"
	"time"

	"github.com/obalunenko/fallout/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCampaignRejectsZeroStartDate(t *testing.T) {
	svc := newSQLiteService(t)

	_, err := svc.CreateCampaign(t.Context(), "camp-1", "Bad Date", time.Time{}, []domain.NewCampaignPlayer{
		{
			PlayerName: "June",
			Character: domain.Combatant{
				Name:       "Vault Dweller",
				Level:      1,
				Initiative: 5,
				HP:         8,
			},
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "campaign start date is required")
}

func TestPrepareCampaignPlayerDefaultsSPECIALAndPreservesNotes(t *testing.T) {
	player := domain.NewCampaignPlayer{
		PlayerName: "June",
		Notes:      "  multiline\nnotes  ",
		Character:  domain.Combatant{Name: "Vault Dweller", Level: 1, HP: 8, MaxHP: 8},
	}

	require.NoError(t, prepareCampaignPlayer(&player))
	assert.Equal(t, domain.DefaultSpecialValues(), player.Special)
	assert.Equal(t, "  multiline\nnotes  ", player.Notes)
}

func TestPrepareCampaignPlayerRejectsInvalidSPECIAL(t *testing.T) {
	player := domain.NewCampaignPlayer{
		PlayerName: "June",
		Character:  domain.Combatant{Name: "Vault Dweller", Level: 1, HP: 8, MaxHP: 8},
		Special:    domain.DefaultSpecialValues(),
	}
	player.Special.Luck = 0

	err := prepareCampaignPlayer(&player)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "luck must be at least 1")
}

func TestCampaignRosterRejectsDuplicatePlayerNamesInAnyScript(t *testing.T) {
	svc := newSQLiteService(t)
	start := testCampaignStartDate(t)
	pc := func(name string) domain.NewCampaignPlayer {
		return domain.NewCampaignPlayer{PlayerName: name, Character: domain.Combatant{Name: "PC " + name, Level: 1, HP: 5, MaxHP: 5}}
	}
	for _, names := range [][2]string{{"Ann", " ann "}, {"Иван", "иван"}} {
		_, err := svc.CreateCampaign(t.Context(), "", "Twins", start, []domain.NewCampaignPlayer{pc(names[0]), pc(names[1])})
		require.ErrorIs(t, err, domain.ErrInvalidInput, "%v", names)
		require.ErrorContains(t, err, "duplicate player name")
	}

	c, err := svc.CreateCampaign(t.Context(), "", "Solo", start, []domain.NewCampaignPlayer{pc("Иван")})
	require.NoError(t, err)
	_, err = svc.UpdateCampaign(t.Context(), c.ID, c.Name, start, []domain.NewCampaignPlayer{pc("Иван"), pc("ИВАН")})
	require.ErrorIs(t, err, domain.ErrInvalidInput, "the desktop editor gets the same rule")
	campaigns, err := svc.ListCampaigns(t.Context())
	require.NoError(t, err)
	for _, stored := range campaigns {
		require.NotEqual(t, "Twins", stored.Name, "a rejected roster is not stored")
	}
}
