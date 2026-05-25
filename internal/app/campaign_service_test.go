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
