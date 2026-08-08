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
